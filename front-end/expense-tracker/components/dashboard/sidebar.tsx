"use client";

import {
  CreditCard,
  HelpCircle,
  History,
  LayoutDashboard,
  LogOut,
  PieChart,
  Settings,
  ShieldCheck,
  Target,
  WalletCards,
} from "lucide-react";
import { LayoutGroup, motion } from "framer-motion";

import { Button } from "@/components/ui/button";
import type { AuthUser } from "@/lib/auth";
import { cn } from "@/lib/utils";
import { initials } from "./utils";

export type DashboardView =
  | "overview"
  | "wallet"
  | "payments"
  | "analytics"
  | "goals"
  | "history"
  | "settings"
  | "help";

const navItems: { icon: React.ElementType; label: string; view: DashboardView }[] = [
  { icon: LayoutDashboard, label: "Tổng quan", view: "overview" },
  { icon: WalletCards, label: "Ví của tôi", view: "wallet" },
  { icon: CreditCard, label: "Thanh toán", view: "payments" },
  { icon: PieChart, label: "Phân tích", view: "analytics" },
  { icon: Target, label: "Mục tiêu", view: "goals" },
  { icon: History, label: "Lịch sử", view: "history" },
  { icon: Settings, label: "Cài đặt", view: "settings" },
  { icon: HelpCircle, label: "Trợ giúp", view: "help" },
];

const springTransition = { type: "spring" as const, stiffness: 500, damping: 38 };

export function Sidebar({
  currentView,
  isLoading,
  onNavigate,
  onSignOut,
  user,
}: {
  currentView: DashboardView;
  isLoading: boolean;
  onNavigate: (view: DashboardView) => void;
  onSignOut: () => void;
  user: AuthUser | null;
}) {
  return (
    <aside className="hidden min-h-full flex-col border-r border-black/[0.06] bg-dash-shell dark:border-white/[0.06] lg:flex">
      <div className="flex items-center gap-3 px-5 py-6">
        <div className="flex size-10 items-center justify-center rounded-xl bg-indigo-500/15 text-indigo-600 ring-1 ring-indigo-500/25 dark:text-indigo-300 dark:ring-indigo-400/25">
          <WalletCards className="size-5" />
        </div>
        <div className="min-w-0">
          <p className="truncate text-sm font-bold tracking-wide text-slate-900 dark:text-slate-100">Nexora</p>
          <p className="truncate text-xs text-slate-500">Expense manager</p>
        </div>
      </div>

      <LayoutGroup id="sidebar-nav">
        <nav className="grid gap-0.5 px-3">
          {navItems.map((item, i) => {
            const Icon = item.icon;
            const isActive = currentView === item.view;
            return (
              <motion.button
                key={item.view}
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.2, ease: "easeOut", delay: i * 0.04 }}
                whileTap={{ scale: 0.97 }}
                className={cn(
                  "relative flex h-10 items-center gap-3 rounded-xl px-3 text-left text-sm font-medium transition-colors duration-150",
                  isActive
                    ? "text-indigo-600 dark:text-indigo-300"
                    : "text-slate-500 hover:text-slate-700 dark:hover:text-slate-300",
                )}
                onClick={() => onNavigate(item.view)}
                type="button"
              >
                {isActive && (
                  <>
                    <motion.span
                      layoutId="sidebar-active-bg"
                      className="absolute inset-0 rounded-xl bg-indigo-500/10"
                      transition={springTransition}
                    />
                    <motion.span
                      layoutId="sidebar-active-bar"
                      className="absolute left-0 top-1/2 h-5 w-[3px] -translate-y-1/2 rounded-r-full bg-indigo-500 dark:bg-indigo-400"
                      transition={springTransition}
                    />
                  </>
                )}
                <Icon className="relative z-10 size-4 shrink-0" />
                <span className="relative z-10 truncate">{item.label}</span>
              </motion.button>
            );
          })}
        </nav>
      </LayoutGroup>

      <div className="mt-auto grid gap-3 p-4">
        <div className="overflow-hidden rounded-xl border border-indigo-500/20 bg-indigo-500/[0.07] p-4">
          <div className="mb-3 flex size-9 items-center justify-center rounded-lg bg-indigo-500/20 text-indigo-600 dark:text-indigo-300">
            <ShieldCheck className="size-5" />
          </div>
          <p className="text-sm font-semibold text-slate-800 dark:text-slate-200">Kế hoạch thông minh</p>
          <p className="mt-1 text-xs leading-5 text-slate-500">
            Theo dõi ngân sách và mục tiêu mỗi tháng.
          </p>
          <button
            className="mt-3 flex h-8 w-full items-center justify-center rounded-lg bg-indigo-500/20 text-xs font-semibold text-indigo-600 transition-colors hover:bg-indigo-500/30 dark:text-indigo-300"
            onClick={() => onNavigate("goals")}
            type="button"
          >
            Bắt đầu
          </button>
        </div>

        <div className="flex min-w-0 items-center gap-3 rounded-xl border border-black/[0.07] bg-black/[0.02] p-3 dark:border-white/[0.07] dark:bg-white/[0.02]">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-indigo-500/30 to-violet-500/30 text-sm font-bold text-indigo-600 ring-1 ring-indigo-500/20 dark:text-indigo-200">
            {initials(user?.name)}
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold text-slate-900 dark:text-slate-100">
              {user?.name ?? "Đang tải"}
            </p>
            <p className="truncate text-xs text-slate-500">{user?.email}</p>
          </div>
          <Button
            aria-label="Đăng xuất"
            className="ml-auto shrink-0 rounded-xl text-slate-500 hover:bg-black/[0.06] hover:text-slate-700 dark:hover:bg-white/[0.06] dark:hover:text-slate-300"
            disabled={isLoading}
            onClick={onSignOut}
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            <LogOut className="size-4" />
          </Button>
        </div>
      </div>
    </aside>
  );
}
