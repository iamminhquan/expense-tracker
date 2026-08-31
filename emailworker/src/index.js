import PostalMime from "postal-mime";
import { sign } from "./sign.js";
import { truncateUtf8 } from "./truncate.js";

// Doc toi da 64KB phan text/plain. Thu ngan hang chi vai KB; cap nay de mot
// thu qua kho khong lam day database free tier.
const MAX_BODY = 64 * 1024;

// Ban MIME goc duoc gui kem, khong phai thay the: `text` la thu parser doc,
// `raw` la thu de chay lai khau boc khi khau do sai -- ma no da sai mot lan roi
// (email chi co HTML, entity khong duoc giai ma). Cap lon hon MAX_BODY vi ban
// goc mang theo header, the va ma hoa truyen tai quanh cung noi dung do.
const MAX_RAW = 2 * MAX_BODY;

// Caps subject and from, which are never bounded by MAX_BODY. Unlike the
// body they normally run a handful of bytes, but nothing stops a hostile or
// malformed message from carrying a header of arbitrary length -- and an
// oversized one would push the request past the handler's own 4x MaxBodyBytes
// ceiling (internal/handlers/inbox_webhook.go), losing the email with only a
// Worker log to show for it.
const MAX_HEADER = 1000;

// Strips tags from an HTML part down to readable text, for the messages that
// carry only text/html -- an email that arrives with an empty body is
// indistinguishable from one that never arrived, so a rough plain text beats
// sending nothing.
// Cac entity co ten hay gap trong email ngan hang. Chu tieng Viet gan nhu
// luon toi duoi dang so (&#7843;) nen bang nay chi can phu Latin-1 va vai dau
// cau -- khong can ca bang HTML5.
const NAMED_ENTITIES = {
  amp: "&", lt: "<", gt: ">", quot: '"', apos: "'", nbsp: " ",
  ndash: "\u2013", mdash: "\u2014", hellip: "\u2026", middot: "\u00b7",
  agrave: "\u00e0", aacute: "\u00e1", acirc: "\u00e2", atilde: "\u00e3",
  egrave: "\u00e8", eacute: "\u00e9", ecirc: "\u00ea",
  igrave: "\u00ec", iacute: "\u00ed",
  ograve: "\u00f2", oacute: "\u00f3", ocirc: "\u00f4", otilde: "\u00f5",
  ugrave: "\u00f9", uacute: "\u00fa", yacute: "\u00fd",
  dgrave: "\u0111", agrave_: "\u00e0",
};

// Giai ma HTML entity, ca dang so (&#7843; / &#x1EA3;) lan dang ten.
//
// Bat buoc phai co: email MB khong co phan text/plain, nen no di qua duong
// HTML, va MB viet chu tieng Viet bang entity so. Khong giai ma thi "Cam on"
// duoc luu thanh "C&#7843;m &#417;n" -- va parser cua lat sau se phai neo vao
// chinh mo rac do thay vi vao nhan that.
//
// Mot luot duy nhat, khong thay the noi tiep: neu lam nhieu luot thi &amp;#65;
// bi giai thanh "A" trong khi no phai giu nguyen la "&#65;".
function decodeEntities(s) {
  return s.replace(
    /&(#[0-9]{1,7}|#[xX][0-9a-fA-F]{1,6}|[a-zA-Z][a-zA-Z0-9]{1,31});/g,
    (whole, body) => {
      if (body[0] === "#") {
        const hex = body[1] === "x" || body[1] === "X";
        const cp = parseInt(hex ? body.slice(2) : body.slice(1), hex ? 16 : 10);
        if (!Number.isFinite(cp) || cp < 1 || cp > 0x10ffff) return whole;
        try {
          return String.fromCodePoint(cp);
        } catch {
          return whole;
        }
      }
      const named = NAMED_ENTITIES[body.toLowerCase()];
      return named === undefined ? whole : named;
    },
  );
}

function htmlToText(html) {
  const stripped = html
    .replace(/<(script|style)[^>]*>[\s\S]*?<\/\1>/gi, " ")
    .replace(/<br\s*\/?\s*>/gi, "\n")
    .replace(/<\/(p|div|tr|li|h[1-6])>/gi, "\n")
    .replace(/<[^>]+>/g, " ");
  // Giai ma sau khi bo the: mot entity nam trong thuoc tinh the khong bao gio
  // tro thanh van ban, va giai ma truoc co the sinh ra dau < > lam hong buoc
  // bo the.
  return decodeEntities(stripped)
    .replace(/[ \t]+/g, " ")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}


// Crude fallback used only when PostalMime itself fails to parse the
// message: slices the raw bytes after the first blank line. It does not
// decode base64/quoted-printable, so it is a last resort -- but a rough body
// beats dropping the message outright.
function crudeExtractText(buf) {
  const raw = new TextDecoder().decode(buf);
  const split = raw.indexOf("\r\n\r\n");
  return split >= 0 ? raw.slice(split + 4) : raw;
}

// extractText picks the body text out of an already-parsed message: the
// text/plain part, or html stripped to text if that is all there is, or a
// crude fallback over the raw bytes if both parts came back empty.
function extractText(email, buf) {
  if (email.text && email.text.trim() !== "") {
    return truncateUtf8(email.text, MAX_BODY);
  }
  if (email.html && email.html.trim() !== "") {
    return truncateUtf8(htmlToText(email.html), MAX_BODY);
  }
  // Both text and html parts are empty -- fall back to crude extraction
  // rather than returning nothing, so a message with an empty body is not
  // indistinguishable from one that never arrived.
  const crude = truncateUtf8(crudeExtractText(buf), MAX_BODY);
  if (crude.trim() === "") {
    console.log("ERROR: email has no readable content after parsing and fallback");
  }
  return crude;
}

// parseMessage buffers the raw message and parses it with PostalMime exactly
// once, returning both the decoded envelope (from/subject/messageId -- null
// if the parse itself failed) and the extracted body text. Everything the
// Worker sends downstream comes out of this one parse rather than a second
// one, because message.from is the SMTP envelope sender (rewritten by Gmail
// to the forwarding account, never the bank's address) and message.headers
// carries raw, possibly RFC 2047-encoded values -- PostalMime's decoded
// `from`/`subject`/`messageId` are the only fields that are actually right.
async function parseMessage(message) {
  // message.raw is a ReadableStream and can only be consumed once, so it is
  // buffered here and reused for both the primary parse and the crude
  // fallback -- PostalMime.parse also accepts an ArrayBuffer directly.
  const buf = await new Response(message.raw).arrayBuffer();

  // Ban goc di kem ket qua boc, cat theo bien gioi rune giong `text`.
  const raw = truncateUtf8(new TextDecoder().decode(buf), MAX_RAW);

  try {
    const email = await PostalMime.parse(buf);
    return { email, text: extractText(email, buf), raw };
  } catch (err) {
    // A real bank notice is usually multipart/alternative with
    // base64/quoted-printable parts; a MIME parse failure here is rare but
    // must not lose the message -- fall back to the pre-parser behaviour.
    // With no parse, from/subject/messageId fall back to raw headers too
    // (see default.email below).
    console.log("ERROR: MIME parse failed, falling back to crude extraction", String(err));
    return { email: null, text: truncateUtf8(crudeExtractText(buf), MAX_BODY), raw };
  }
}

export { parseMessage };

export default {
  async email(message, env, ctx) {
    // Never let anything below throw out of email(): a thrown error makes
    // Cloudflare bounce the message back to the sender, which here is the
    // bank. That includes transport failures -- DNS, TLS, or the app being
    // unreachable, which happens routinely on Render's free tier because the
    // service sleeps when idle and the first request after a sleep can fail
    // outright. The email stays safe in the owner's Gmail either way, so a
    // logged failure is recoverable and a bounce is not.
    try {
      if (!env.INBOUND_WEBHOOK_SECRET || !env.APP_BASE_URL) {
        console.log("ERROR: worker is missing INBOUND_WEBHOOK_SECRET or APP_BASE_URL");
        return;
      }

      // Local part cua dia chi nhan la token dinh danh tai khoan.
      const token = String(message.to || "").split("@")[0].toLowerCase();
      if (!token) {
        console.log("ERROR: no token in recipient", message.to);
        return;
      }

      const { email, text, raw } = await parseMessage(message);

      // message.from is the SMTP envelope sender, not the From: header --
      // Gmail rewrites it to the forwarding account on every message it
      // relays, so it is never the bank's address. The decoded From: header
      // from the one PostalMime parse above is the field isKnownBankSender
      // actually needs to check; the raw header is a fallback for the rare
      // case the parse itself failed. Same reasoning for subject/messageId:
      // a Vietnamese subject arrives RFC 2047 encoded, and the raw header is
      // what message.headers.get would otherwise hand over unencoded.
      const from = (email && email.from && email.from.address) || message.headers.get("from") || "";
      const subject = (email && email.subject) || message.headers.get("subject") || "";
      const messageId = (email && email.messageId) || message.headers.get("message-id") || "";

      const payload = JSON.stringify({
        from: truncateUtf8(from, MAX_HEADER),
        to: message.to,
        subject: truncateUtf8(subject, MAX_HEADER),
        messageId,
        text,
        raw,
      });

      const res = await fetch(
        `${env.APP_BASE_URL}/inbox/${encodeURIComponent(token)}`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-Inbox-Signature": await sign(env.INBOUND_WEBHOOK_SECRET, payload),
          },
          body: payload,
        },
      );

      // Khong throw: nem loi o day khien Cloudflare bounce thu ve nguoi gui,
      // ma nguoi gui la ngan hang. Log lai la du -- thu van con trong hop thu
      // Gmail cua chu tai khoan de forward lai.
      if (!res.ok) {
        console.log("ERROR: app rejected the email", res.status, await res.text());
      }
    } catch (err) {
      console.log("ERROR: email handler failed", String(err));
    }
  },
};
