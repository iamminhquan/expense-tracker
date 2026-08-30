// Caps a string at maxBytes measured in UTF-8, without splitting a
// multi-byte character.
//
// Mirrors inbound.TruncateBody on the Go side (internal/inbound), which
// needed the same fix: Vietnamese text runs 2-3 bytes per character in
// UTF-8, so a plain `.slice(0, n)` counts UTF-16 code units and can let the
// payload run up to ~3x over the intended cap. Cutting by raw bytes alone
// has the opposite problem -- it can split a multi-byte character in half,
// leaving invalid UTF-8 at the tail, which is exactly what corrupts a
// sample the next slice needs to parse against.
export function truncateUtf8(s, maxBytes) {
  const bytes = new TextEncoder().encode(s);
  if (bytes.length <= maxBytes) return s;

  // Trim back byte-by-byte until the prefix decodes cleanly. Only the tail
  // can be broken -- everything before it was already valid UTF-8 -- and a
  // UTF-8 character is at most 4 bytes, so this runs at most 3 times.
  const decoder = new TextDecoder("utf-8", { fatal: true });
  for (let end = maxBytes; end > 0; end--) {
    try {
      return decoder.decode(bytes.subarray(0, end));
    } catch {
      // `end` landed inside a multi-byte sequence; trim one more byte.
    }
  }
  return "";
}
