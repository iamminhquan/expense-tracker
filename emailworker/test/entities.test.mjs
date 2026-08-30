import { test } from "node:test";
import assert from "node:assert/strict";
import { readText } from "../src/index.js";

// Dung nguyen hinh dang email MB that: khong co phan text/plain, chu tieng
// Viet viet bang entity so, tron lan voi UTF-8 that.
function mbLikeHtmlMessage() {
  return [
    "From: mbebanking@mbbank.com.vn",
    "To: token@in.example.site",
    "Subject: Thong bao giao dich thanh cong",
    "MIME-Version: 1.0",
    'Content-Type: text/html; charset="utf-8"',
    "",
    "<html><body><p>C&#7843;m &#417;n Qu&#253; kh&#225;ch.</p>",
    "<table><tr><td>S&#7889; ti&#7873;n giao d&#7883;ch</td><td>(VND) 10,000.00</td></tr>",
    "<tr><td>N&#7897;i dung chuy&#7875;n ti&#7873;n</td><td>BUI MINH QUAN chuyen tien</td></tr>",
    "<tr><td>Lo&#7841;i giao d&#7883;ch</td><td>Chuyển tiền nội bộ MB</td></tr></table>",
    "<p>To&agrave; nh&agrave; MB &ndash; H&agrave; N&#7897;i &amp; nhieu noi khac</p>",
    "</body></html>",
    "",
  ].join("\r\n");
}

test("decodes the numeric entities MB writes Vietnamese with", async () => {
  const got = await readText({ raw: mbLikeHtmlMessage() });
  assert.match(got, /Cảm ơn Quý khách/);
  assert.match(got, /Số tiền giao dịch/);
  assert.match(got, /Nội dung chuyển tiền/);
});

test("leaves no undecoded entity behind", async () => {
  const got = await readText({ raw: mbLikeHtmlMessage() });
  assert.doesNotMatch(got, /&#[0-9]+;/, "con entity so chua giai ma");
  assert.doesNotMatch(got, /&(amp|ndash|agrave);/, "con entity ten chua giai ma");
});

test("decodes named entities and keeps real UTF-8 untouched", async () => {
  const got = await readText({ raw: mbLikeHtmlMessage() });
  assert.match(got, /Toà nhà MB – Hà Nội & nhieu noi khac/);
  assert.match(got, /Chuyển tiền nội bộ MB/);
});

test("does not cascade: &amp;#65; stays literal", async () => {
  const raw = [
    "From: x@mbbank.com.vn",
    "To: t@in.example.site",
    'Content-Type: text/html; charset="utf-8"',
    "",
    "<p>&amp;#65;</p>",
    "",
  ].join("\r\n");
  const got = await readText({ raw });
  assert.match(got, /&#65;/, "phai giu nguyen chu khong giai thanh A");
  assert.doesNotMatch(got, /(^|[^#])A([^0-9]|$)/);
});
