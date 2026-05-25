"use client";

import type { ExpenseSummary } from "./utils";
import { formatCompactCurrency } from "./utils";

export function SavingsGoals({ summary }: { summary: ExpenseSummary }) {
  const goals = [
    {
      label: "Quỹ dự phòng",
      saved: 12000000,
      target: 30000000,
      bar: "bg-gradient-to-r from-indigo-500 to-violet-500",
      track: "bg-black/[0.05] dark:bg-white/[0.05]",
    },
    {
      label: "Du lịch",
      saved: Math.max(5000000 - summary.monthTotal * 0.1, 0),
      target: 15000000,
      bar: "bg-gradient-to-r from-violet-500 to-pink-500",
      track: "bg-black/[0.05] dark:bg-white/[0.05]",
    },
  ];

  return (
    <section className="rounded-2xl border border-black/[0.07] bg-dash-card p-5 shadow-[0_4px_16px_rgba(0,0,0,0.06)] dark:border-white/[0.07] dark:shadow-[0_4px_40px_rgba(0,0,0,0.4)]">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-slate-500">Mục tiêu tiết kiệm</h2>
        <button className="text-xs font-semibold text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300" type="button">
          Xem tất cả
        </button>
      </div>
      <div className="grid gap-4">
        {goals.map((goal) => {
          const progress = Math.min((goal.saved / goal.target) * 100, 100);
          return (
            <div
              className="rounded-xl bg-black/[0.02] p-4 transition-colors hover:bg-black/[0.035] dark:bg-white/[0.02] dark:hover:bg-white/[0.035]"
              key={goal.label}
            >
              <div className="mb-3 flex items-center justify-between gap-3">
                <p className="truncate text-sm font-semibold text-slate-800 dark:text-slate-200">{goal.label}</p>
                <p className="shrink-0 text-xs font-bold tabular-nums text-slate-500">
                  {Math.round(progress)}%
                </p>
              </div>
              <div className={`h-1.5 overflow-hidden rounded-full ${goal.track}`}>
                <div
                  className={`h-full rounded-full transition-all duration-1000 ${goal.bar}`}
                  style={{ width: `${progress}%` }}
                />
              </div>
              <div className="mt-2.5 flex items-center justify-between text-xs text-slate-500">
                <span>{formatCompactCurrency(goal.saved)}</span>
                <span>{formatCompactCurrency(goal.target)}</span>
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}
