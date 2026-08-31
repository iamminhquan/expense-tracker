import PostalMime from "postal-mime";
import { sign } from "./sign.js";
import { truncateUtf8 } from "./truncate.js";

const MAX_BODY = 64 * 1024;
const MAX_RAW = 2 * MAX_BODY;
const MAX_HEADER = 1000;
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

  return decodeEntities(stripped)
    .replace(/[ \t]+/g, " ")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

function crudeExtractText(buf) {
  const raw = new TextDecoder().decode(buf);
  const split = raw.indexOf("\r\n\r\n");
  return split >= 0 ? raw.slice(split + 4) : raw;
}

function extractText(email, buf) {
  if (email.text && email.text.trim() !== "") {
    return truncateUtf8(email.text, MAX_BODY);
  }
  if (email.html && email.html.trim() !== "") {
    return truncateUtf8(htmlToText(email.html), MAX_BODY);
  }

  const crude = truncateUtf8(crudeExtractText(buf), MAX_BODY);
  if (crude.trim() === "") {
    console.log("ERROR: email has no readable content after parsing and fallback");
  }
  return crude;
}

async function parseMessage(message) {
  const buf = await new Response(message.raw).arrayBuffer();
  const raw = truncateUtf8(new TextDecoder().decode(buf), MAX_RAW);

  try {
    const email = await PostalMime.parse(buf);
    return { email, text: extractText(email, buf), raw };
  } catch (err) {
    console.log("ERROR: MIME parse failed, falling back to crude extraction", String(err));
    return { email: null, text: truncateUtf8(crudeExtractText(buf), MAX_BODY), raw };
  }
}

export { parseMessage };

export default {
  async email(message, env, ctx) {

    try {
      if (!env.INBOUND_WEBHOOK_SECRET || !env.APP_BASE_URL) {
        console.log("ERROR: worker is missing INBOUND_WEBHOOK_SECRET or APP_BASE_URL");
        return;
      }

      const token = String(message.to || "").split("@")[0].toLowerCase();
      if (!token) {
        console.log("ERROR: no token in recipient", message.to);
        return;
      }

      const { email, text, raw } = await parseMessage(message);
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

      if (!res.ok) {
        console.log("ERROR: app rejected the email", res.status, await res.text());
      }
    } catch (err) {
      console.log("ERROR: email handler failed", String(err));
    }
  },
};
