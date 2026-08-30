import PostalMime from "postal-mime";
import { sign } from "./sign.js";
import { truncateUtf8 } from "./truncate.js";

// Doc toi da 64KB phan text/plain. Thu ngan hang chi vai KB; cap nay de mot
// thu qua kho khong lam day database free tier.
const MAX_BODY = 64 * 1024;

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
function htmlToText(html) {
  return html
    .replace(/<(script|style)[^>]*>[\s\S]*?<\/\1>/gi, " ")
    .replace(/<br\s*\/?\s*>/gi, "\n")
    .replace(/<\/(p|div|tr|li|h[1-6])>/gi, "\n")
    .replace(/<[^>]+>/g, " ")
    .replace(/&nbsp;/gi, " ")
    .replace(/&amp;/gi, "&")
    .replace(/&lt;/gi, "<")
    .replace(/&gt;/gi, ">")
    .replace(/&quot;/gi, '"')
    .replace(/&#39;/gi, "'")
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

  try {
    const email = await PostalMime.parse(buf);
    return { email, text: extractText(email, buf) };
  } catch (err) {
    // A real bank notice is usually multipart/alternative with
    // base64/quoted-printable parts; a MIME parse failure here is rare but
    // must not lose the message -- fall back to the pre-parser behaviour.
    // With no parse, from/subject/messageId fall back to raw headers too
    // (see default.email below).
    console.log("ERROR: MIME parse failed, falling back to crude extraction", String(err));
    return { email: null, text: truncateUtf8(crudeExtractText(buf), MAX_BODY) };
  }
}

// readText is a thin wrapper over parseMessage for callers (and tests) that
// only want the body text.
async function readText(message) {
  return (await parseMessage(message)).text;
}

export { readText, parseMessage };

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

      const { email, text } = await parseMessage(message);

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
