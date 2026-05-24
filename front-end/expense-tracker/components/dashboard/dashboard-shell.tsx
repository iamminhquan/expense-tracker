"use client";

import type { ChangeEvent, FormEvent } from "react";
import { useEffect, useMemo, useState } from "react";

import type { AuthUser } from "@/lib/auth";
import {
  createExpense,
  deleteExpense,
  fetchExpenses,
  formFromExpense,
  initialExpenseForm,
  updateExpense,
  type Expense,
  type ExpenseForm,
} from "@/lib/expenses";

import { AnimatePresence, motion } from "framer-motion";

import { BalanceChart } from "./balance-chart";
import { ComingSoon } from "./coming-soon";
import { ExpenseEditor } from "./expense-editor";
import { MobileNav } from "./mobile-nav";
import { MonthlySnapshot } from "./monthly-snapshot";
import { QuickActions } from "./quick-actions";
import { RecentActivity } from "./recent-activity";
import { SavingsGoals } from "./savings-goals";
import { type DashboardView, Sidebar } from "./sidebar";
import { SpendingOverview } from "./spending-overview";
import { Topbar } from "./topbar";
import { summarizeExpenses } from "./utils";

export function DashboardShell({
  isLoading = false,
  token = "",
  user,
  onSignOut,
}: {
  isLoading?: boolean;
  token?: string;
  user: AuthUser | null;
  onSignOut: () => void;
}) {
  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [form, setForm] = useState<ExpenseForm>(initialExpenseForm);
  const [editingExpenseId, setEditingExpenseId] = useState<number | null>(null);
  const [message, setMessage] = useState("");
  const [isExpenseLoading, setIsExpenseLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [currentView, setCurrentView] = useState<DashboardView>("overview");

  useEffect(() => {
    if (!token) return;
    let isCurrent = true;

    async function load() {
      setIsExpenseLoading(true);
      setMessage("");
      try {
        const { data, response } = await fetchExpenses(token);
        if (!isCurrent) return;
        if (!response.ok) throw new Error(data.error ?? "Không thể tải danh sách chi tiêu");
        setExpenses(data.expenses ?? []);
      } catch (error) {
        if (isCurrent) {
          setMessage(
            error instanceof Error ? error.message : "Không thể tải danh sách chi tiêu",
          );
        }
      } finally {
        if (isCurrent) setIsExpenseLoading(false);
      }
    }

    load();
    return () => {
      isCurrent = false;
    };
  }, [token]);

  const summary = useMemo(() => summarizeExpenses(expenses), [expenses]);
  const disabled = isLoading || isSaving || !token;

  function updateField(field: keyof ExpenseForm) {
    return (
      event: ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>,
    ): void => {
      setForm((f) => ({ ...f, [field]: event.target.value }));
    };
  }

  function editExpense(expense: Expense) {
    setEditingExpenseId(expense.id);
    setForm(formFromExpense(expense));
    setMessage("");
    setCurrentView("overview");
  }

  function resetForm() {
    setEditingExpenseId(null);
    setForm(initialExpenseForm);
    setMessage("");
  }

  async function submitExpense(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) return;

    setIsSaving(true);
    setMessage("");

    try {
      const result =
        editingExpenseId === null
          ? await createExpense(token, form)
          : await updateExpense(token, editingExpenseId, form);

      if (!result.response.ok || !result.data.expense) {
        throw new Error(result.data.error ?? "Không thể lưu khoản chi");
      }

      setExpenses((current) =>
        editingExpenseId === null
          ? [result.data.expense as Expense, ...current]
          : current.map((e) =>
              e.id === editingExpenseId ? (result.data.expense as Expense) : e,
            ),
      );
      resetForm();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Không thể lưu khoản chi");
    } finally {
      setIsSaving(false);
    }
  }

  async function removeExpense(expense: Expense) {
    if (!token || !confirm(`Xóa khoản chi "${expense.description}"?`)) return;
    setMessage("");

    try {
      const response = await deleteExpense(token, expense.id);
      if (!response.ok) throw new Error("Không thể xóa khoản chi");
      setExpenses((current) => current.filter((e) => e.id !== expense.id));
      if (editingExpenseId === expense.id) resetForm();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Không thể xóa khoản chi");
    }
  }

  return (
    <main
      className="min-h-screen p-2 text-slate-900 sm:p-4 dark:text-slate-100"
      style={{
        backgroundColor: "var(--dash-bg)",
        backgroundImage: `
          radial-gradient(ellipse at 15% 25%, var(--dash-glow-1) 0%, transparent 50%),
          radial-gradient(ellipse at 85% 75%, var(--dash-glow-2) 0%, transparent 50%),
          radial-gradient(var(--dash-dot) 1px, transparent 1px)
        `,
        backgroundSize: "100% 100%, 100% 100%, 24px 24px",
      }}
    >
      <div className="mx-auto grid min-h-[calc(100vh-16px)] max-w-[1440px] overflow-hidden rounded-2xl border border-black/[0.06] bg-dash-shell shadow-[0_24px_80px_rgba(0,0,0,0.08)] dark:border-white/[0.06] dark:shadow-[0_24px_80px_rgba(0,0,0,0.6)] lg:grid-cols-[220px_minmax(0,1fr)]">
        <Sidebar
          currentView={currentView}
          isLoading={isLoading}
          onNavigate={setCurrentView}
          onSignOut={onSignOut}
          user={user}
        />

        <div
          className="flex min-w-0 flex-col"
          style={{
            backgroundColor: "var(--dash-bg)",
            backgroundImage: `
              radial-gradient(ellipse at 20% 10%, var(--dash-glow-1) 0%, transparent 40%),
              radial-gradient(var(--dash-dot) 1px, transparent 1px)
            `,
            backgroundSize: "100% 100%, 24px 24px",
          }}
        >
          <Topbar isLoading={isLoading} onSignOut={onSignOut} summary={summary} user={user} />

          <div className="min-w-0 flex-1 p-3 sm:p-5">
            <AnimatePresence mode="wait">
              <motion.div
                key={currentView}
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -6 }}
                transition={{ duration: 0.16, ease: [0.25, 0.46, 0.45, 0.94] }}
              >
                {currentView === "overview" ? (
                  <div className="grid gap-4 pb-20 sm:gap-5 lg:grid-cols-[minmax(0,1fr)_340px] lg:pb-5 xl:grid-cols-[minmax(0,1fr)_360px]">
                    <motion.section
                      className="grid min-w-0 content-start gap-4 sm:gap-5"
                      initial={{ opacity: 0, y: 18 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ duration: 0.28, ease: "easeOut" }}
                    >
                      <BalanceChart summary={summary} />
                      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_280px]">
                        <SpendingOverview summary={summary} />
                        <MonthlySnapshot summary={summary} />
                      </div>
                      <RecentActivity expenses={expenses} isLoading={isExpenseLoading} onDelete={removeExpense} onEdit={editExpense} />
                    </motion.section>

                    <motion.aside
                      className="grid min-w-0 content-start gap-4 sm:grid-cols-2 lg:grid-cols-1"
                      initial={{ opacity: 0, y: 18 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ duration: 0.28, ease: "easeOut", delay: 0.07 }}
                    >
                      <QuickActions />
                      <SavingsGoals summary={summary} />
                      <ExpenseEditor
                        disabled={disabled}
                        editingExpenseId={editingExpenseId}
                        form={form}
                        isSaving={isSaving}
                        message={message}
                        onCancel={resetForm}
                        onFieldChange={updateField}
                        onSubmit={submitExpense}
                        user={user}
                      />
                    </motion.aside>
                  </div>
                ) : (
                  <ComingSoon view={currentView} />
                )}
              </motion.div>
            </AnimatePresence>
          </div>
        </div>
      </div>
      <MobileNav currentView={currentView} onNavigate={setCurrentView} onSignOut={onSignOut} />
    </main>
  );
}
