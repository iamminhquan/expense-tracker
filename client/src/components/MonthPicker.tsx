import type { MonthOption } from '../lib/api/types'

interface MonthPickerProps {
  value: string
  label: string
  currentMonthValue: string
  availableMonths: MonthOption[]
  onChange: (value: string) => void
  /** Only the transactions page offers "All months" -- see
   * server/internal/handlers's identical restriction and
   * .claude/rules/req-value-objects.md for why the dashboard never does. */
  allowAllMonths?: boolean
}

/** Mirrors month_picker.html: a dropdown of "This month" plus every other month with data. */
export function MonthPicker({ value, label, currentMonthValue, availableMonths, onChange, allowAllMonths }: MonthPickerProps) {
  return (
    <select
      aria-label="Month"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="rounded-[10px] border border-border-input bg-surface px-3 py-1.5 text-[13px] text-ink"
    >
      <option value={currentMonthValue}>{value === currentMonthValue ? label : 'This month'}</option>
      {availableMonths
        .filter((m) => m.value !== currentMonthValue)
        .map((m) => (
          <option key={m.value} value={m.value}>
            {m.label}
          </option>
        ))}
      {allowAllMonths && <option value="all">All months</option>}
    </select>
  )
}
