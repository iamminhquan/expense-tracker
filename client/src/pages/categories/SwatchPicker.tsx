import { Check } from 'lucide-react'
import { SWATCHES } from '../../lib/categorySwatches'

const NAMES: Record<string, string> = {
  '#D97757': 'Terracotta',
  '#5B8DEF': 'Blue',
  '#8B7BD8': 'Violet',
  '#6BA292': 'Sage',
  '#E0A82E': 'Amber',
  '#D97AA0': 'Pink',
  '#4FA871': 'Green',
  '#7CA65C': 'Olive',
}

export function SwatchPicker({ value, onChange }: { value: string; onChange: (color: string) => void }) {
  return (
    <div role="radiogroup" aria-label="Color" className="grid grid-cols-[repeat(auto-fill,44px)] gap-1.5">
      {SWATCHES.map((color) => {
        const selected = color === value
        return (
          <label key={color} className="relative flex size-11 cursor-pointer items-center justify-center">
            <input
              type="radio"
              name="category-color"
              value={color}
              checked={selected}
              onChange={() => onChange(color)}
              aria-label={NAMES[color]}
              className="peer sr-only"
            />
            <span
              aria-hidden="true"
              className="press flex size-9 items-center justify-center rounded-full ring-offset-[3px] ring-offset-surface peer-checked:scale-110 peer-checked:ring-2 peer-checked:ring-ink peer-focus-visible:outline-3 peer-focus-visible:outline-offset-4 peer-focus-visible:outline-accent"
              style={{ backgroundColor: color }}
            >
              {selected && <Check strokeWidth={3} className="size-[18px] text-on-swatch" />}
            </span>
          </label>
        )
      })}
    </div>
  )
}
