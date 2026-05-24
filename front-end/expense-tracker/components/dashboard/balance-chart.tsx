"use client";

import { TrendingDown, TrendingUp } from "lucide-react";
import { useTheme } from "next-themes";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import { cn } from "@/lib/utils";
import { useIsClient } from "./hooks";
import type { ExpenseSummary } from "./utils";
import { formatCompactCurrency, formatCurrency } from "./utils";

export function BalanceChart({ summary }: { summary: ExpenseSummary }) {
  const isClient = useIsClient();
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme !== "light";

  const points = summary.monthlyPoints;
  const thisMonth = points[points.length - 1]?.total ?? 0;
  const lastMonth = points[points.length - 2]?.total ?? 0;
  const trend = lastMonth > 0 ? ((thisMonth - lastMonth) / lastMonth) * 100 : 0;
  const isUp = trend > 0;

  const chart = isDark
    ? {
        grid: "rgba(148,163,184,0.05)",
        tick: "#475569",
        tooltipBg: "#18223a",
        tooltipBorder: "rgba(148,163,184,0.1)",
        tooltipShadow: "0 16px 40px rgba(0,0,0,0.5)",
        labelColor: "#f1f5f9",
        activeDotFill: "#0f1624",
      }
    : {
        grid: "rgba(15,23,42,0.06)",
        tick: "#94a3b8",
        tooltipBg: "#ffffff",
        tooltipBorder: "rgba(15,23,42,0.08)",
        tooltipShadow: "0 4px 20px rgba(0,0,0,0.08)",
        labelColor: "#0f172a",
        activeDotFill: "#ffffff",
      };

  return (
    <section className="overflow-hidden rounded-2xl border border-black/[0.07] bg-dash-card shadow-[0_4px_16px_rgba(0,0,0,0.06)] dark:border-white/[0.07] dark:shadow-[0_4px_40px_rgba(0,0,0,0.5)]">
      <div className="px-6 pb-2 pt-6">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-xs font-medium uppercase tracking-wider text-slate-500">
              Tổng chi tiêu
            </p>
            <h1
              className="mt-2 text-4xl font-extrabold tracking-tight text-slate-900 sm:text-5xl dark:text-slate-50"
              style={{ textShadow: isDark ? "0 0 60px rgba(129, 140, 248, 0.2)" : "none" }}
            >
              {formatCurrency(summary.total)}
            </h1>
            <div className="mt-3 flex flex-wrap items-center gap-2">
              {trend !== 0 ? (
                <span
                  className={cn(
                    "inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-semibold",
                    isUp
                      ? "bg-rose-500/15 text-rose-600 dark:text-rose-400"
                      : "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400",
                  )}
                >
                  {isUp ? (
                    <TrendingUp className="size-3.5" />
                  ) : (
                    <TrendingDown className="size-3.5" />
                  )}
                  {isUp ? "+" : ""}
                  {trend.toFixed(1)}% so với tháng trước
                </span>
              ) : null}
              <span className="text-xs text-slate-500">
                {summary.monthCount} giao dịch tháng này
              </span>
            </div>
          </div>
          <span className="rounded-xl border border-black/[0.07] bg-black/[0.03] px-3 py-1.5 text-xs font-medium text-slate-500 dark:border-white/[0.07] dark:bg-white/[0.03]">
            6 tháng gần nhất
          </span>
        </div>
      </div>

      <div className="h-52 px-2 pb-3 sm:h-60">
        {isClient ? (
          <ResponsiveContainer height="100%" width="100%">
            <AreaChart
              data={points}
              margin={{ bottom: 8, left: 0, right: 8, top: 8 }}
            >
              <defs>
                <linearGradient id="areaFill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#818cf8" stopOpacity={isDark ? 0.35 : 0.2} />
                  <stop offset="100%" stopColor="#818cf8" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid
                stroke={chart.grid}
                strokeDasharray="4 4"
                vertical={false}
              />
              <XAxis
                axisLine={false}
                dataKey="label"
                interval={0}
                tick={{ fill: chart.tick, fontSize: 11 }}
                tickLine={false}
              />
              <YAxis
                axisLine={false}
                tick={{ fill: chart.tick, fontSize: 10 }}
                tickFormatter={(v) => formatCompactCurrency(Number(v))}
                tickLine={false}
                width={52}
              />
              <Tooltip
                contentStyle={{
                  background: chart.tooltipBg,
                  border: `1px solid ${chart.tooltipBorder}`,
                  borderRadius: 12,
                  boxShadow: chart.tooltipShadow,
                  padding: "8px 12px",
                }}
                cursor={{ stroke: chart.grid, strokeWidth: 1 }}
                formatter={(value) => [formatCurrency(Number(value)), "Chi tiêu"]}
                labelStyle={{ color: chart.labelColor, fontWeight: 700, marginBottom: 2 }}
                itemStyle={{ color: "#818cf8" }}
              />
              <Area
                activeDot={{
                  fill: chart.activeDotFill,
                  r: 5,
                  stroke: "#818cf8",
                  strokeWidth: 3,
                }}
                dataKey="total"
                dot={false}
                fill="url(#areaFill)"
                stroke="#818cf8"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2.5}
                type="monotone"
              />
            </AreaChart>
          </ResponsiveContainer>
        ) : (
          <div className="h-full rounded-xl bg-dash-chart-bg" />
        )}
      </div>
    </section>
  );
}
