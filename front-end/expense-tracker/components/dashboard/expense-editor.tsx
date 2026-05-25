"use client";

import type { ChangeEvent, FormEvent, ReactNode } from "react";
import { Check, Loader2, Plus, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import type { AuthUser } from "@/lib/auth";
import type { ExpenseForm } from "@/lib/expenses";
import { cn } from "@/lib/utils";

const categoryOptions = [
  "Ăn uống","Đi lại","Nhà cửa","Mua sắm","Sức khỏe",
  "Giải trí","Học tập","Gia đình","Hóa đơn","Khác",
];

const paymentMethodOptions = [
  "Tiền mặt","Chuyển khoản","Thẻ ngân hàng","Ví điện tử","Thẻ tín dụng","Khác",
];

const currencyOptions = ["VND", "USD", "EUR", "GBP", "JPY", "SGD"];

const recurringPeriodOptions = [
  { value: "daily", label: "Hàng ngày" },
  { value: "weekly", label: "Hàng tuần" },
  { value: "monthly", label: "Hàng tháng" },
  { value: "yearly", label: "Hàng năm" },
];

type FieldChangeHandler = (
  field: keyof ExpenseForm,
) => (event: ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => void;

export function ExpenseEditor({
  disabled,
  editingExpenseId,
  form,
  isSaving,
  message,
  onCancel,
  onFieldChange,
  onSubmit,
  user,
}: {
  disabled: boolean;
  editingExpenseId: number | null;
  form: ExpenseForm;
  isSaving: boolean;
  message: string;
  onCancel: () => void;
  onFieldChange: FieldChangeHandler;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  user: AuthUser | null;
}) {
  const isEditing = editingExpenseId !== null;

  return (
    <section className="rounded-2xl border border-black/[0.07] bg-dash-card p-5 shadow-[0_4px_16px_rgba(0,0,0,0.06)] dark:border-white/[0.07] dark:shadow-[0_4px_40px_rgba(0,0,0,0.4)] sm:col-span-2 lg:col-span-1">
      <div className="mb-5">
        <span className="inline-flex items-center gap-1.5 rounded-lg bg-indigo-500/10 px-2.5 py-1 text-xs font-semibold text-indigo-600 ring-1 ring-indigo-500/20 dark:text-indigo-400">
          <Plus className="size-3" />
          Nhập nhanh
        </span>
        <h2 className="mt-3 text-base font-semibold text-slate-900 dark:text-slate-100">
          {isEditing ? "Sửa khoản chi" : "Thêm khoản chi"}
        </h2>
        {user ? (
          <p className="mt-0.5 truncate text-xs text-slate-500">{user.name}</p>
        ) : null}
      </div>

      <form className="grid gap-4" onSubmit={onSubmit}>
        <div className="grid grid-cols-[1fr_96px] gap-3">
          <Field label="Số tiền">
            <input
              className={inputCls}
              disabled={disabled}
              inputMode="numeric"
              onChange={(e) => {
                const raw = e.target.value.replace(/[^\d]/g, "");
                onFieldChange("amount")({ target: { value: raw } } as ChangeEvent<HTMLInputElement>);
              }}
              placeholder="120.000"
              required
              type="text"
              value={form.amount ? Number(form.amount).toLocaleString("vi-VN") : ""}
            />
          </Field>
          <Field label="Tiền tệ">
            <select
              className={inputCls}
              disabled={disabled}
              onChange={onFieldChange("currency")}
              value={form.currency}
            >
              {currencyOptions.map((c) => (
                <option key={c} value={c}>{c}</option>
              ))}
            </select>
          </Field>
        </div>

        <Field label="Mô tả">
          <input
            className={inputCls}
            disabled={disabled}
            onChange={onFieldChange("description")}
            placeholder="Ăn trưa"
            required
            type="text"
            value={form.description}
          />
        </Field>

        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
          <Field label="Danh mục">
            <select
              className={inputCls}
              disabled={disabled}
              onChange={onFieldChange("category")}
              required
              value={form.category}
            >
              <option value="">Chọn danh mục</option>
              {categoryOptions.map((cat) => (
                <option key={cat} value={cat}>{cat}</option>
              ))}
            </select>
          </Field>

          <Field label="Ngày">
            <input
              className={inputCls}
              disabled={disabled}
              onChange={onFieldChange("date")}
              required
              type="date"
              value={form.date}
            />
          </Field>
        </div>

        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
          <Field label="Phương thức">
            <select
              className={inputCls}
              disabled={disabled}
              onChange={onFieldChange("paymentMethod")}
              value={form.paymentMethod}
            >
              <option value="">Chọn phương thức</option>
              {paymentMethodOptions.map((method) => (
                <option key={method} value={method}>{method}</option>
              ))}
            </select>
          </Field>
        </div>

        <label className="flex cursor-pointer items-center gap-3 rounded-xl border border-black/[0.07] bg-dash-input px-3 py-2.5 dark:border-white/[0.07]">
          <input
            checked={form.isRecurring === "true"}
            className="size-4 rounded accent-indigo-500"
            disabled={disabled}
            onChange={(e) => {
              onFieldChange("isRecurring")({
                target: { value: String(e.target.checked) },
              } as ChangeEvent<HTMLInputElement>);
            }}
            type="checkbox"
          />
          <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">
            Chi phí định kỳ
          </span>
        </label>

        {form.isRecurring === "true" && (
          <Field label="Chu kỳ">
            <select
              className={inputCls}
              disabled={disabled}
              onChange={onFieldChange("recurringPeriod")}
              value={form.recurringPeriod}
            >
              <option value="">Chọn chu kỳ</option>
              {recurringPeriodOptions.map((p) => (
                <option key={p.value} value={p.value}>{p.label}</option>
              ))}
            </select>
          </Field>
        )}

        <Field label="Ghi chú">
          <textarea
            className={cn(inputCls, "min-h-[72px] resize-none py-2.5")}
            disabled={disabled}
            onChange={onFieldChange("note")}
            placeholder="Thông tin bổ sung"
            value={form.note}
          />
        </Field>

        {message ? (
          <p className="rounded-xl border border-rose-500/20 bg-rose-500/10 px-3.5 py-2.5 text-sm text-rose-600 dark:text-rose-400">
            {message}
          </p>
        ) : null}

        <div className="flex flex-col gap-2 sm:flex-row lg:flex-col xl:flex-row">
          <button
            className="flex h-10 flex-1 items-center justify-center gap-2 rounded-xl bg-indigo-500 text-sm font-semibold text-white shadow-lg shadow-indigo-500/20 transition-all hover:bg-indigo-400 disabled:cursor-not-allowed disabled:opacity-40"
            disabled={disabled}
            type="submit"
          >
            {isSaving ? (
              <Loader2 className="size-4 animate-spin" />
            ) : isEditing ? (
              <Check className="size-4" />
            ) : (
              <Plus className="size-4" />
            )}
            {isEditing ? "Lưu thay đổi" : "Thêm khoản chi"}
          </button>
          {isEditing ? (
            <Button
              className="h-10 rounded-xl border border-black/[0.07] bg-black/[0.03] text-slate-500 hover:bg-black/[0.06] hover:text-slate-700 dark:border-white/[0.07] dark:bg-white/[0.03] dark:hover:bg-white/[0.06] dark:hover:text-slate-200"
              disabled={disabled}
              onClick={onCancel}
              type="button"
              variant="outline"
            >
              <X className="size-4" />
              Hủy
            </Button>
          ) : null}
        </div>
      </form>
    </section>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block min-w-0 space-y-1.5 text-xs font-semibold uppercase tracking-wider text-slate-500">
      <span>{label}</span>
      {children}
    </label>
  );
}

const inputCls =
  "h-10 w-full min-w-0 rounded-xl border border-black/[0.07] bg-dash-input px-3 text-sm font-medium text-slate-900 outline-none transition-all placeholder:text-slate-400 focus:border-indigo-500/40 focus:bg-dash-input-focus focus:ring-2 focus:ring-indigo-500/15 disabled:cursor-not-allowed disabled:opacity-40 dark:border-white/[0.07] dark:text-slate-100 dark:placeholder:text-slate-600";
