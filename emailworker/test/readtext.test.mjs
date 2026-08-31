import { test } from "node:test";
import assert from "node:assert/strict";
import { parseMessage } from "../src/index.js";

// A multipart/alternative message with quoted-printable text/plain part
// containing Vietnamese text with diacritics. Must use CRLF line endings for
// proper MIME parsing.
const multipartQuotedPrintable = "From: bank@example.com\r\nTo: user@example.com\r\nSubject: Balance Notification\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=\"boundary123\"\r\n\r\n--boundary123\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\nNg=C3=A2n h=C3=A0ng th=C3=B4ng b=C3=A1o: Giao d=E1=BB=8Bch -50,000 VND.\r\n\r\n--boundary123\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\nPHAAAAacD4gPg==\r\n--boundary123--\r\n";

// A message with a base64-encoded text/plain part. The body is valid base64
// (no padding mid-string) that decodes to "Ngân hàng báo Giao dịch -50,000
// VND.", so the test below can assert the actual decoded text rather than
// just "non-empty".
const base64Body = Buffer.from("Ngân hàng báo Giao dịch -50,000 VND.", "utf8").toString("base64");
const base64TextPart = `From: bank@example.com\r\nTo: user@example.com\r\nSubject: Test\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n${base64Body}\r\n`;

// A message with HTML only, no text/plain part.
const htmlOnlyMessage = "From: bank@example.com\r\nTo: user@example.com\r\nSubject: Test\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: 7bit\r\n\r\n<p>Ngân hàng thông báo</p><br /><p>Giao dịch -50,000 VND.</p>\r\n";

// Helper to create a mock message object with a ReadableStream raw property.
function createMessage(rawEmail) {
  const encoder = new TextEncoder();
  const buffer = encoder.encode(rawEmail);
  return {
    raw: ReadableStream.from([buffer]),
  };
}

test("multipart/alternative with quoted-printable text/plain decodes Vietnamese diacritics", async () => {
  const message = createMessage(multipartQuotedPrintable);
  const text = (await parseMessage(message)).text;

  // The expected decoded Vietnamese text with diacritics.
  assert.ok(text.includes("Ngân hàng thông báo"), "must contain decoded Vietnamese text");
  assert.ok(text.includes("Giao dịch -50,000 VND"), "must include the full message");
});

test("multipart/alternative output contains no MIME boundary markers", async () => {
  const message = createMessage(multipartQuotedPrintable);
  const text = (await parseMessage(message)).text;

  // The bug symptom: boundary lines or raw MIME artifacts present in output.
  assert.ok(!text.includes("--boundary"), "must not contain MIME boundary markers");
  assert.ok(!text.includes("Content-Type:"), "must not contain MIME headers");
  assert.ok(!text.includes("Content-Transfer-Encoding:"), "must not contain transfer encoding headers");
});

test("multipart/alternative output contains no base64 runs", async () => {
  const message = createMessage(multipartQuotedPrintable);
  const text = (await parseMessage(message)).text;

  // The bug symptom: long sequences of base64-looking characters.
  // This checks that we didn't just dump the raw MIME with undecoded parts.
  assert.ok(!text.includes("PHA"), "must not contain raw base64 sequences from HTML part");
});

test("base64-encoded text/plain part decodes correctly", async () => {
  const message = createMessage(base64TextPart);
  const text = (await parseMessage(message)).text;

  assert.equal(text, "Ngân hàng báo Giao dịch -50,000 VND.");
});

test("HTML-only message falls back to tag-stripped text", async () => {
  const message = createMessage(htmlOnlyMessage);
  const text = (await parseMessage(message)).text;

  // Tags should be stripped, leaving readable text.
  assert.ok(text.includes("Ngân hàng thông báo"), "must contain decoded content");
  assert.ok(text.includes("Giao dịch -50,000 VND"), "must include the message");
  assert.ok(!text.includes("<p>"), "must not contain HTML tags");
  assert.ok(!text.includes("</p>"), "must not contain HTML tags");
  assert.ok(!text.includes("<br"), "must not contain BR tags");
});
