"use client";

import { useTheme } from "next-themes";
import {
  Cell,
  Pie,
  PieChart as RechartsPieChart,
  ResponsiveContainer,
  Tooltip,
} from "recharts";

import { useIsClient } from "./hooks";
import type { ExpenseSummary } from "./utils";
import { CATEGORY_CHART_COLORS, formatCompactCurrency, formatCurrency } from "./utils";

export function SpendingOverview({ summary }: { summary: ExpenseSummary }) {
  const isClient = useIsClient();
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme !== "light";

  const categories = summary.categoryBreakdown.slice(0, 6);
  const empty = categories.length === 0;
  const chartData = empty ? [{ name: "Trống", total: 1 }] : categories;

  const tooltip = isDark
    ? {
        bg: "#18223a",
        border: "1px solid rgba(148,163,184,0.1)",
        shadow: "0 16px 40px rgba(0,0,0,0.5)",
      }
    : {
        bg: "#ffffff",
        border: "1px solid rgba(15,23,42,0.08)",
        shadow: "0 4px 20px rgba(0,0,0,0.08)",
      };

  return (
    <section className="rounded-2xl border border-black/[0.07] bg-dash-card p-5 shadow-[0_4px_16px_rgba(0,0,0,0.06)] dark:border-white/[0.07] dark:shadow-[0_4px_40px_rgba(0,0,0,0.4)]">
      <div className="mb-5 flex items-center justify-between gap-3">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-slate-500">Cơ cấu chi tiêu</h2>
        <button
          className="rounded-xl border border-black/[0.07] bg-black/[0.03] px-3 py-1.5 text-xs font-medium text-slate-500 transition-colors hover:bg-black/[0.06] hover:text-slate-700 dark:border-white/[0.07] dark:bg-white/[0.03] dark:hover:bg-white/[0.06] dark:hover:text-slate-300"
          type="button"
        >
          Báo cáo
        </button>
      </div>

      <div className="grid gap-5 md:grid-cols-[180px_minmax(0,1fr)] md:items-center">
        <div className="relative mx-auto size-44">
          {isClient ? (
            <ResponsiveContainer height="100%" minHeight={1} minWidth={1} width="100%">
              <RechartsPieChart>
                <Pie
                  cx="50%"
                  cy="50%"
                  data={chartData}
                  dataKey="total"
                  innerRadius={54}
                  outerRadius={82}
                  paddingAngle={3}
                  stroke="none"
                >
                  {chartData.map((entry, index) => (
                    <Cell
                      fill={
                        empty
                          ? isDark ? "rgba(148,163,184,0.1)" : "rgba(15,23,42,0.07)"
                          : CATEGORY_CHART_COLORS[index % CATEGORY_CHART_COLORS.length]
                      }
                      key={entry.name}
                    />
                  ))}
                </Pie>
                {!empty ? (
                  <Tooltip
                    contentStyle={{
                      background: tooltip.bg,
                      border: tooltip.border,
                      borderRadius: 12,
                      boxShadow: tooltip.shadow,
                      padding: "8px 12px",
                    }}
                    formatter={(value) => [formatCurrency(Number(value)), "Chi tiêu"]}
                    itemStyle={{ color: "#f1f5f9" }}
                  />
                ) : null}
              </RechartsPieChart>
            </ResponsiveContainer>
          ) : (
            <div className="size-full rounded-full bg-dash-chart-bg" />
          )}
          <div className="pointer-events-none absolute inset-0 grid place-items-center">
            <div className="grid size-24 place-items-center rounded-full bg-dash-card text-center ring-1 ring-black/[0.06] dark:ring-white/[0.06]">
              <div>
                <p className="text-[10px] text-slate-500">Tổng</p>
                <p className="text-sm font-bold tabular-nums text-slate-800 dark:text-slate-200">
                  {formatCompactCurrency(summary.total)}
                </p>
              </div>
            </div>
          </div>
        </div>

        <div className="grid gap-2.5">
          {empty ? (
            <p className="rounded-xl bg-black/[0.03] px-4 py-3 text-sm text-slate-500 dark:bg-white/[0.03]">
              Chưa có dữ liệu chi tiêu.
            </p>
          ) : (
            categories.map((category, index) => (
              <div
                className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-3"
                key={category.name}
              >
                <div className="flex min-w-0 items-center gap-2">
                  <span
                    className="size-2 shrink-0 rounded-full"
                    style={{
                      backgroundColor:
                        CATEGORY_CHART_COLORS[index % CATEGORY_CHART_COLORS.length],
                    }}
                  />
                  <span className="truncate text-sm text-slate-600 dark:text-slate-400">{category.name}</span>
                </div>
                <div className="text-right">
                  <p className="text-sm font-semibold tabular-nums text-slate-800 dark:text-slate-200">
                    {formatCompactCurrency(category.total)}
                  </p>
                  <p className="text-xs text-slate-500">
                    {Math.round((category.total / summary.total) * 100) || 0}%
                  </p>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </section>
  );
}
