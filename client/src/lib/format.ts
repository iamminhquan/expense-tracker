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

function localISO(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

// "Today", "Yesterday", or "Mon, 06 Oct" (with the year when the list spans every month).
export function formatDayLabel(isoDate: string, withYear = false): string {
  const today = new Date()
  const yesterday = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 1)
  if (isoDate === localISO(today)) return 'Today'
  if (isoDate === localISO(yesterday)) return 'Yesterday'
  const d = new Date(isoDate + 'T00:00:00')
  return d.toLocaleDateString('en-GB', { weekday: 'short', day: '2-digit', month: 'short', ...(withYear ? { year: 'numeric' } : {}) })
}
