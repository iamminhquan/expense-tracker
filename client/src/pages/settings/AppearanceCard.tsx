import { Monitor, Moon, Sun } from 'lucide-react'
import { SegmentedControl } from '../../components/ui/SegmentedControl'
import { useTheme } from '../../lib/theme/ThemeContext'
import type { Theme } from '../../lib/api/types'
import { Card } from './Card'

// The same switch as the account menu's; here because Settings is one tap away in the phone dock.
export function AppearanceCard() {
  const { theme, setTheme } = useTheme()
  return (
    <Card title="Appearance" description="Auto follows your device's light or dark setting.">
      <SegmentedControl<Theme>
        label="Theme"
        value={theme}
        onChange={setTheme}
        size="tall"
        fullWidth
        options={[
          { value: 'light', label: 'Light', icon: <Sun aria-hidden="true" /> },
          { value: 'auto', label: 'Auto', icon: <Monitor aria-hidden="true" /> },
          { value: 'dark', label: 'Dark', icon: <Moon aria-hidden="true" /> },
        ]}
      />
    </Card>
  )
}
