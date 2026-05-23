"use client";

import {
  ArrowDownRight,
  ArrowUpRight,
  CalendarDays,
  LogOut,
  PiggyBank,
  Plus,
  ReceiptText,
  ShieldCheck,
  WalletCards,
} from "lucide-react";

import type { AuthUser } from "@/lib/auth";
import { Button } from "@/components/ui/button";

const summaryItems = [
  {
    label: "Available balance",
    value: "$0.00",
    detail: "No account data yet",
    icon: WalletCards,
  },
  {
    label: "This month spent",
    value: "$0.00",
    detail: "0 transactions recorded",
    icon: ArrowDownRight,
  },
  {
    label: "Budget remaining",
    value: "$0.00",
    detail: "Create a monthly budget",
    icon: PiggyBank,
  },
];

const nextSteps = [
  "Create your first budget category",
  "Add an expense transaction",
  "Review monthly spending trends",
];

export function DashboardShell({
  isLoading = false,
  user,
  onSignOut,
}: {
  isLoading?: boolean;
  user: AuthUser | null;
  onSignOut: () => void;
}) {
  return (
    <main className="min-h-screen bg-background text-foreground">
      <DashboardHeader
        isLoading={isLoading}
        onSignOut={onSignOut}
        user={user}
      />

      <div className="mx-auto w-full max-w-6xl px-4 py-6 sm:px-6">
        <DashboardHero isLoading={isLoading} user={user} />
        <SummaryGrid isLoading={isLoading} />

        <section className="mt-6 grid gap-6 lg:grid-cols-[1fr_320px]">
          <RecentActivity isLoading={isLoading} />
          <DashboardSidebar user={user} />
        </section>
      </div>
    </main>
  );
}

function DashboardHeader({
  isLoading,
  onSignOut,
  user,
}: {
  isLoading: boolean;
  onSignOut: () => void;
  user: AuthUser | null;
}) {
  return (
    <header className="border-b border-border bg-card">
      <div className="mx-auto flex w-full max-w-6xl items-center justify-between gap-4 px-4 py-4 sm:px-6">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-md border border-border bg-background">
            <WalletCards className="size-5" />
          </div>
          <div className="min-w-0">
            <p className="text-sm font-medium">Budget Expense Tracker</p>
            <p className="truncate text-xs text-muted-foreground">
              {user ? user.email : "Checking session"}
            </p>
          </div>
        </div>

        <Button
          className="h-9"
          disabled={isLoading}
          onClick={onSignOut}
          type="button"
          variant="outline"
        >
          <LogOut className="size-4" />
          Sign out
        </Button>
      </div>
    </header>
  );
}

function DashboardHero({
  isLoading,
  user,
}: {
  isLoading: boolean;
  user: AuthUser | null;
}) {
  return (
    <section className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div className="space-y-2">
        <div className="inline-flex items-center gap-2 rounded-md border border-border px-3 py-1 text-xs text-muted-foreground">
          <ShieldCheck className="size-3.5 text-emerald-600" />
          Secure session
        </div>
        <div>
          <h1 className="text-3xl font-semibold tracking-normal">
            {user ? `Welcome back, ${user.name}` : "Loading your workspace"}
          </h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
            Track budgets, expenses, and monthly cash flow from one focused
            workspace.
          </p>
        </div>
      </div>

      <Button className="h-10 w-full sm:w-auto" disabled={isLoading}>
        <Plus className="size-4" />
        Add expense
      </Button>
    </section>
  );
}

function SummaryGrid({ isLoading }: { isLoading: boolean }) {
  return (
    <section className="grid gap-3 md:grid-cols-3">
      {summaryItems.map((item) => (
        <SummaryCard key={item.label} item={item} isLoading={isLoading} />
      ))}
    </section>
  );
}

function SummaryCard({
  isLoading,
  item,
}: {
  isLoading: boolean;
  item: (typeof summaryItems)[number];
}) {
  const Icon = item.icon;

  return (
    <div className="rounded-md border border-border bg-card p-4">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm text-muted-foreground">{item.label}</p>
          <p className="mt-2 text-2xl font-semibold tracking-normal tabular-nums">
            {isLoading ? "--" : item.value}
          </p>
        </div>
        <div className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-background">
          <Icon className="size-4" />
        </div>
      </div>
      <p className="mt-4 text-xs text-muted-foreground">
        {isLoading ? "Loading account data" : item.detail}
      </p>
    </div>
  );
}

function RecentActivity({ isLoading }: { isLoading: boolean }) {
  return (
    <div className="rounded-md border border-border bg-card">
      <div className="flex items-center justify-between gap-4 border-b border-border px-4 py-3">
        <div>
          <h2 className="text-sm font-medium">Recent activity</h2>
          <p className="text-xs text-muted-foreground">
            Transactions will appear here after you add expenses.
          </p>
        </div>
        <Button className="h-8" disabled={isLoading} variant="outline">
          <CalendarDays className="size-4" />
          Month
        </Button>
      </div>

      <div className="flex min-h-72 flex-col items-center justify-center px-4 py-10 text-center">
        <div className="mb-4 flex size-12 items-center justify-center rounded-md border border-border bg-background">
          <ReceiptText className="size-6 text-muted-foreground" />
        </div>
        <h3 className="text-base font-medium">No expenses yet</h3>
        <p className="mt-2 max-w-sm text-sm leading-6 text-muted-foreground">
          Start by adding an expense so the dashboard can show spending, budget
          usage, and recent activity.
        </p>
        <Button className="mt-5 h-10" disabled={isLoading}>
          <Plus className="size-4" />
          Add first expense
        </Button>
      </div>
    </div>
  );
}

function DashboardSidebar({ user }: { user: AuthUser | null }) {
  return (
    <aside className="space-y-4">
      <AccountPanel user={user} />
      <NextStepsPanel />
    </aside>
  );
}

function AccountPanel({ user }: { user: AuthUser | null }) {
  return (
    <div className="rounded-md border border-border bg-card p-4">
      <h2 className="text-sm font-medium">Account</h2>
      <div className="mt-4 space-y-3 text-sm">
        <div>
          <p className="text-xs text-muted-foreground">Name</p>
          <p className="font-medium">{user?.name ?? "Loading"}</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Email</p>
          <p className="break-all font-medium">
            {user?.email ?? "Checking session"}
          </p>
        </div>
      </div>
    </div>
  );
}

function NextStepsPanel() {
  return (
    <div className="rounded-md border border-border bg-card p-4">
      <h2 className="text-sm font-medium">Next steps</h2>
      <ul className="mt-4 space-y-3">
        {nextSteps.map((step) => (
          <li key={step} className="flex gap-3 text-sm">
            <ArrowUpRight className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
            <span>{step}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
