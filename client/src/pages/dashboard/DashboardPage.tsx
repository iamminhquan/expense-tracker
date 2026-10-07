import { useSearchParams } from 'react-router-dom'
import { useDashboard } from '../../hooks/useDashboard'
import { MonthPicker } from '../../components/MonthPicker'
import { InlineError } from '../../components/ui/InlineError'
import { PageSkeleton } from '../../components/ui/PageSkeleton'
import { pageTitleClass } from '../../lib/formStyles'
import { KpiCards } from './KpiCards'
import { MonthlyBars } from './MonthlyBars'
import { SpendingDoughnut } from './SpendingDoughnut'

export function DashboardPage() {
  // The month lives in the URL, so reloads and links keep it.
  const [searchParams, setSearchParams] = useSearchParams()
  const month = searchParams.get('month') ?? undefined
  const { data, error, refetch } = useDashboard(month)

  function setMonth(value: string) {
    const next = new URLSearchParams(searchParams)
    next.set('month', value)
    setSearchParams(next, { replace: true })
  }

  if (!data) {
    if (error) return <InlineError message="Could not load the overview." onRetry={() => void refetch()} />
    return <PageSkeleton label="Loading the overview…" />
  }

  return (
    <div className="space-y-4 md:space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className={pageTitleClass}>Overview</h1>
        <MonthPicker
          value={data.monthValue}
          label={data.monthLabel}
          currentMonthValue={data.currentMonthValue}
          availableMonths={data.availableMonths}
          onChange={setMonth}
          size="lg"
        />
      </div>

      <KpiCards data={data} />

      <div className="grid grid-cols-1 gap-4 md:gap-5 lg:grid-cols-12">
        <div className="lg:col-span-5">
          <SpendingDoughnut pie={data.pie} total={data.totalExpense} monthLabel={data.monthLabel} />
        </div>
        <div className="lg:col-span-7">
          <MonthlyBars bar={data.bar} />
        </div>
      </div>
    </div>
  )
}
