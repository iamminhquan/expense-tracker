"use client";

import { Landmark, ReceiptText, WalletCards } from "lucide-react";

import { cn } from "@/lib/utils";
import type { ExpenseSummary } from "./utils";
import { formatCurrency } from "./utils";

export function MonthlySnapshot({ summary }: { summary: ExpenseSummary }) {
  const income = Math.max(summary.monthTotal * 1.8, 15000000);
  const savings = Math.max(income - summary.monthTotal, 0);

  const items = [
    {
      icon: Landmark,
      label: "Thu nhập ước tính",
      value: formatCurrency(income),
      iconClass: "bg-emerald-400/10 text-emerald-600 dark:text-emerald-400",
      valueClass: "text-emerald-600 dark:text-emerald-400",
    },
    {
      icon: ReceiptText,
      label: "Chi tiêu",
      value: formatCurrency(summary.monthTotal),
      iconClass: "bg-rose-400/10 text-rose-600 dark:text-rose-400",
      valueClass: "text-rose-600 dark:text-rose-400",
    },
    {
      icon: WalletCards,
      label: "Còn lại",
      value: formatCurrency(savings),
      iconClass: "bg-indigo-400/10 text-indigo-600 dark:text-indigo-400",
      valueClass: "text-indigo-600 dark:text-indigo-300",
    },
  ];

  return (
    <section className="rounded-2xl border border-black/[0.07] bg-dash-card p-5 shadow-[0_4px_16px_rgba(0,0,0,0.06)] dark:border-white/[0.07] dark:shadow-[0_4px_40px_rgba(0,0,0,0.4)]">
      <div className="mb-4 flex items-center justify-between gap-3">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-slate-500">Tháng này</h2>
        <span className="rounded-xl border border-black/[0.07] bg-black/[0.02] px-2.5 py-1 text-[10px] text-slate-500 dark:border-white/[0.07] dark:bg-white/[0.02]">
          Hiện tại
        </span>
      </div>
      <div className="grid gap-2">
        {items.map((item) => {
          const Icon = item.icon;
          return (
            <div
              className="grid grid-cols-[36px_minmax(0,1fr)_auto] items-center gap-3 rounded-xl bg-black/[0.02] px-3 py-2.5 dark:bg-white/[0.02]"
              key={item.label}
            >
              <div className={cn("grid size-8 place-items-center rounded-lg text-sm", item.iconClass)}>
                <Icon className="size-4" />
              </div>
              <p className="truncate text-sm text-slate-500">{item.label}</p>
              <p className={cn("text-sm font-bold tabular-nums", item.valueClass)}>
                {item.value}
              </p>
            </div>
          );
        })}
      </div>
    </section>
  );
}
