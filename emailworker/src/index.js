// Handler email() cua Cloudflare Email Routing.
//
// Tam thoi chi log. Muc dich duy nhat: xac nhan email forward toi
// <bat ky>@in.ttth-caothang.site thuc su chay toi day.
//
// Doc log bang: npx wrangler tail
export default {
  async email(message, env, ctx) {
    const headers = message.headers;

    console.log(
      JSON.stringify({
        from: message.from,
        to: message.to,
        subject: headers.get("subject") || "",
        messageId: headers.get("message-id") || "",
        size: message.rawSize,
      }),
    );

    // Gmail gui mot ma xac nhan khi ban them dia chi forward. Ma do nam
    // trong than thu, ma than thu chi doc duoc qua stream raw -- nen in
    // 2KB dau ra day de con lay ma o buoc cau hinh filter Gmail.
    try {
      const raw = await new Response(message.raw).text();
      console.log("RAW_HEAD:", raw.slice(0, 2000));
    } catch (err) {
      console.log("RAW_ERROR:", String(err));
    }
  },
};
