"use client";

import type { ComponentType } from "react";
import {
  BookOpen,
  Car,
  Gamepad2,
  Heart,
  Home,
  Layers,
  Loader2,
  Pencil,
  Receipt,
  ReceiptText,
  ShoppingBag,
  Trash2,
  UtensilsCrossed,
  Users,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import type { Expense } from "@/lib/expenses";
import { cn } from "@/lib/utils";
import { formatCurrency, formatDate, getCategoryStyle } from "./utils";

const CATEGORY_ICONS: Record<string, ComponentType<{ className?: string }>> = {
  "Ăn uống": UtensilsCrossed,
  "Đi lại": Car,
  "Nhà cửa": Home,
  "Mua sắm": ShoppingBag,
  "Sức khỏe": Heart,
  "Giải trí": Gamepad2,
  "Học tập": BookOpen,
  "Gia đình": Users,
  "Hóa đơn": Receipt,
  "Khác": Layers,
};

export function RecentActivity({
  expenses,
  isLoading,
  onDelete,
  onEdit,
}: {
  expenses: Expense[];
  isLoading: boolean;
  onDelete: (expense: Expense) => void;
  onEdit: (expense: Expense) => void;
}) {
  return (
    <section className="overflow-hidden rounded-2xl border border-black/[0.07] bg-dash-card shadow-[0_4px_16px_rgba(0,0,0,0.06)] dark:border-white/[0.07] dark:shadow-[0_4px_40px_rgba(0,0,0,0.4)]">
      <div className="flex items-center justify-between gap-3 border-b border-black/[0.06] px-5 py-4 dark:border-white/[0.06]">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-slate-500">
          Hoạt động gần đây
        </h2>
        <button className="text-xs font-semibold text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300" type="button">
          Tất cả
        </button>
      </div>

      {expenses.length === 0 ? (
        <EmptyState isLoading={isLoading} />
      ) : (
        <div className="divide-y divide-black/[0.04] dark:divide-white/[0.04]">
          {expenses.slice(0, 8).map((expense) => (
            <ExpenseRow
              expense={expense}
              key={expense.id}
              onDelete={onDelete}
              onEdit={onEdit}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function EmptyState({ isLoading }: { isLoading: boolean }) {
  return (
    <div className="grid min-h-52 place-items-center p-6 text-center">
      <div>
        <div className="mx-auto mb-3 grid size-14 place-items-center rounded-2xl bg-black/[0.04] text-slate-500 dark:bg-white/[0.04]">
          {isLoading ? (
            <Loader2 className="size-6 animate-spin" />
          ) : (
            <ReceiptText className="size-6" />
          )}
        </div>
        <p className="font-semibold text-slate-700 dark:text-slate-300">
          {isLoading ? "Đang tải khoản chi" : "Chưa có khoản chi"}
        </p>
        <p className="mt-1 text-sm text-slate-500">
          {isLoading
            ? "Dữ liệu sẽ hiển thị ngay sau đây."
            : "Thêm khoản chi đầu tiên ở khung bên cạnh."}
        </p>
      </div>
    </div>
  );
}

function ExpenseRow({
  expense,
  onDelete,
  onEdit,
}: {
  expense: Expense;
  onDelete: (expense: Expense) => void;
  onEdit: (expense: Expense) => void;
}) {
  const style = getCategoryStyle(expense.category);
  const Icon = CATEGORY_ICONS[expense.category] ?? Layers;

  return (
    <article className="group grid min-w-0 gap-3 px-5 py-4 transition-colors hover:bg-black/[0.02] dark:hover:bg-white/[0.02] sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
      <div className="flex min-w-0 items-start gap-3">
        <div
          className={cn(
            "grid size-10 shrink-0 place-items-center rounded-xl",
            style.bg,
            style.text,
          )}
        >
          <Icon className="size-4" />
        </div>
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <p className="min-w-0 break-words text-sm font-semibold text-slate-900 dark:text-slate-100">
              {expense.description}
            </p>
            <span
              className="rounded-md px-2 py-0.5 text-xs font-medium"
              style={{ background: `${style.hex}1a`, color: style.hex }}
            >
              {expense.category}
            </span>
          </div>
          <div className="mt-0.5 flex flex-wrap items-center gap-x-3 text-xs text-slate-500">
            <span>{formatDate(expense.date)}</span>
            {expense.payment_method ? <span className="truncate">{expense.payment_method}</span> : null}
          </div>
        </div>
      </div>

      <div className="flex min-w-0 items-center justify-between gap-3 sm:justify-end">
        <p className="text-sm font-bold tabular-nums text-slate-900 dark:text-slate-100">
          -{formatCurrency(expense.amount)}
        </p>
        <div className="flex gap-1 sm:opacity-0 sm:transition-opacity sm:group-hover:opacity-100">
          <Button
            aria-label="Sửa khoản chi"
            className="rounded-xl border border-black/[0.07] bg-black/[0.03] text-slate-500 hover:border-indigo-500/30 hover:bg-indigo-500/10 hover:text-indigo-600 dark:border-white/[0.07] dark:bg-white/[0.03] dark:hover:text-indigo-400"
            onClick={() => onEdit(expense)}
            size="icon-sm"
            type="button"
            variant="outline"
          >
            <Pencil className="size-3.5" />
          </Button>
          <Button
            aria-label="Xóa khoản chi"
            className="rounded-xl border border-black/[0.07] bg-black/[0.03] text-slate-500 hover:border-rose-500/30 hover:bg-rose-500/10 hover:text-rose-600 dark:border-white/[0.07] dark:bg-white/[0.03] dark:hover:text-rose-400"
            onClick={() => onDelete(expense)}
            size="icon-sm"
            type="button"
            variant="outline"
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      </div>
    </article>
  );
}
