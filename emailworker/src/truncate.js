export function truncateUtf8(s, maxBytes) {
  const bytes = new TextEncoder().encode(s);
  if (bytes.length <= maxBytes) return s;

  const decoder = new TextDecoder("utf-8", { fatal: true });
  for (let end = maxBytes; end > 0; end--) {
    try {
      return decoder.decode(bytes.subarray(0, end));
    } catch {

    }
  }
  return "";
}
