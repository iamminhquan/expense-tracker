// Ky HMAC-SHA256 tren raw body, tra ve hex thuong.
//
// Phai khop tung byte voi inbound.Sign trong Go (internal/inbound). Doi mot
// ben ma quen ben kia thi email dung lai im lang: app tra 403 va khong ai
// nhin thay, vi Worker khong co cho nao bao loi.
export async function sign(secret, body) {
  const enc = new TextEncoder();
  const key = await crypto.subtle.importKey(
    "raw",
    enc.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const mac = await crypto.subtle.sign("HMAC", key, enc.encode(body));
  return [...new Uint8Array(mac)]
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}
