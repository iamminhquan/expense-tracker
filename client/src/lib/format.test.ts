import { describe, expect, it } from 'vitest'
import { formatDayLabel } from './format'

describe('formatDayLabel', () => {
  const today = new Date(2026, 9, 7)

  it('names today and yesterday', () => {
    expect(formatDayLabel('2026-10-07', false, today)).toBe('Today')
    expect(formatDayLabel('2026-10-06', false, today)).toBe('Yesterday')
  })

  it('falls back to a short date, with the year when asked', () => {
    expect(formatDayLabel('2026-09-30', false, today)).toMatch(/^Wed, 30 Sep/)
    expect(formatDayLabel('2026-09-30', true, today)).toMatch(/^Wed, 30 Sep\w* 2026$/)
  })
})
