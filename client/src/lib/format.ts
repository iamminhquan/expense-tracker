const vndFormatter = new Intl.NumberFormat('vi-VN')

export function formatVND(amount: number): string {
  return `${vndFormatter.format(Math.abs(amount))}₫`
}

export function formatVNDSigned(amount: number): string {
  if (amount === 0) return formatVND(0)
  return `${amount > 0 ? '+' : '−'}${formatVND(amount)}`
}

export function formatDateShort(isoDate: string): string {
  const d = new Date(isoDate + 'T00:00:00')
  return d.toLocaleDateString('en-GB', { day: '2-digit', month: 'short' })
}

export function formatDateLong(isoDate: string): string {
  const d = new Date(isoDate + 'T00:00:00')
  return d.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
}

export function formatTimestamp(iso: string): string {
  const d = new Date(iso)
  const date = d.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
  const time = d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', hour12: false })
  return `${date}, ${time}`
}

/** "Oct 2026" for a "2026-10" month value; null for anything else, such as "all". */
export function formatMonthShort(monthValue: string): string | null {
  if (!/^\d{4}-\d{2}$/.test(monthValue)) return null
  return new Date(monthValue + '-01T00:00:00').toLocaleDateString('en-GB', { month: 'short', year: 'numeric' })
}

function isoOf(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

/** "Today", "Yesterday", or a short weekday and date; `today` is injectable for tests. */
export function formatDayLabel(isoDate: string, showYear: boolean, today = new Date()): string {
  if (isoDate === isoOf(today)) return 'Today'
  const yesterday = new Date(today)
  yesterday.setDate(today.getDate() - 1)
  if (isoDate === isoOf(yesterday)) return 'Yesterday'
  const weekday = new Date(isoDate + 'T00:00:00').toLocaleDateString('en-GB', { weekday: 'short' })
  return `${weekday}, ${showYear ? formatDateLong(isoDate) : formatDateShort(isoDate)}`
}
