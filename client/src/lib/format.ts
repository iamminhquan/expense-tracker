const vndFormatter = new Intl.NumberFormat('vi-VN')

export function formatVND(amount: number): string {
  return `${vndFormatter.format(Math.abs(amount))}₫`
}

export function formatVNDSigned(amount: number): string {
  if (amount === 0) return formatVND(0)
  return `${amount > 0 ? '+' : '-'}${formatVND(amount)}`
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
