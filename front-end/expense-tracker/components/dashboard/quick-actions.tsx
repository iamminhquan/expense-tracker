"use client";

import { ArrowLeftRight, CircleDollarSign, Plus, ReceiptText, Settings } from "lucide-react";
import { cn } from "@/lib/utils";

const actions = [
  { icon: Plus, label: "Thêm tiền", bg: "bg-emerald-400/10", text: "text-emerald-600 dark:text-emerald-400" },
  { icon: ArrowLeftRight, label: "Chuyển tiền", bg: "bg-indigo-400/10", text: "text-indigo-600 dark:text-indigo-400" },
  { icon: ReceiptText, label: "Trả hóa đơn", bg: "bg-rose-400/10", text: "text-rose-600 dark:text-rose-400" },
  { icon: CircleDollarSign, label: "Quy đổi", bg: "bg-amber-400/10", text: "text-amber-600 dark:text-amber-400" },
];

export function QuickActions() {
  return (
    <section className="rounded-2xl border border-black/[0.07] bg-dash-card p-5 shadow-[0_4px_16px_rgba(0,0,0,0.06)] dark:border-white/[0.07] dark:shadow-[0_4px_40px_rgba(0,0,0,0.4)]">
      <div className="mb-4 flex items-center justify-between gap-3">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-slate-500">Tác vụ nhanh</h2>
        <Settings className="size-4 text-slate-400" />
      </div>
      <div className="grid grid-cols-4 gap-2 sm:grid-cols-2 lg:grid-cols-4">
        {actions.map((action) => {
          const Icon = action.icon;
          return (
            <button
              className="grid justify-items-center gap-2.5 rounded-xl p-2.5 text-center text-xs font-medium text-slate-500 transition-colors hover:bg-black/[0.04] hover:text-slate-700 dark:hover:bg-white/[0.04] dark:hover:text-slate-300"
              key={action.label}
              type="button"
            >
              <span className={cn("grid size-10 place-items-center rounded-xl transition-colors", action.bg, action.text)}>
                <Icon className="size-4" />
              </span>
              <span className="w-full truncate">{action.label}</span>
            </button>
          );
        })}
      </div>
    </section>
  );
}
