// Client-side equivalents of server/internal/format's display helpers --
// the server now ships raw numbers (see dashboard_handlers.go's comment on
// why) and leaves formatting to whichever client renders them.

const vndFormatter = new Intl.NumberFormat('vi-VN')

/** formatVND renders a đồng amount as "50.000₫". */
export function formatVND(amount: number): string {
  return `${vndFormatter.format(Math.abs(amount))}₫`
}

/** formatVNDSigned prefixes a non-zero amount with its sign: "-50.000₫", "+50.000₫". */
export function formatVNDSigned(amount: number): string {
  if (amount === 0) return formatVND(0)
  return `${amount > 0 ? '+' : '-'}${formatVND(amount)}`
}

/** formatDateShort renders "2006-01-02" as "02 Jan". */
export function formatDateShort(isoDate: string): string {
  const d = new Date(isoDate + 'T00:00:00')
  return d.toLocaleDateString('en-GB', { day: '2-digit', month: 'short' })
}

/** formatDateLong renders "2006-01-02" as "02 Jan 2026" -- used wherever the year is ambiguous (the "all months" transactions view). */
export function formatDateLong(isoDate: string): string {
  const d = new Date(isoDate + 'T00:00:00')
  return d.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
}

/** formatTimestamp renders an ISO timestamp as "02 Jan 2026, 15:04" (the active-sessions list). */
export function formatTimestamp(iso: string): string {
  const d = new Date(iso)
  const date = d.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
  const time = d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', hour12: false })
  return `${date}, ${time}`
}
