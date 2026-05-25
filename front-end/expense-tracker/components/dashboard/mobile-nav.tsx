"use client";

import { Home, LogOut, PieChart, Plus, Target } from "lucide-react";
import { LayoutGroup, motion } from "framer-motion";

import { cn } from "@/lib/utils";
import type { DashboardView } from "./sidebar";

const items: { icon: React.ElementType; label: string; view?: DashboardView; center?: true }[] = [
  { icon: Home, label: "Tổng quan", view: "overview" },
  { icon: PieChart, label: "Báo cáo", view: "analytics" },
  { icon: Plus, label: "Thêm", center: true },
  { icon: Target, label: "Mục tiêu", view: "goals" },
];

const springTransition = { type: "spring" as const, stiffness: 500, damping: 38 };

export function MobileNav({
  currentView,
  onNavigate,
  onSignOut,
}: {
  currentView: DashboardView;
  onNavigate: (view: DashboardView) => void;
  onSignOut: () => void;
}) {
  return (
    <nav className="fixed inset-x-3 bottom-3 z-30 flex rounded-2xl border border-black/[0.07] bg-white/95 p-1 shadow-2xl shadow-black/10 backdrop-blur-md dark:border-white/[0.07] dark:bg-[var(--dash-card)]/95 dark:shadow-black/60 lg:hidden">
      <LayoutGroup id="mobile-nav">
        {items.map((item) => {
          const Icon = item.icon;
          const isCenter = item.center === true;
          const isActive = !isCenter && item.view === currentView;
          return (
            <motion.button
              key={item.label}
              whileTap={{ scale: 0.93 }}
              className={cn(
                "relative grid flex-1 justify-items-center gap-1 rounded-xl py-2 text-[11px] font-medium transition-colors",
                isCenter
                  ? "text-slate-500"
                  : isActive
                    ? "text-indigo-600 dark:text-indigo-300"
                    : "text-slate-500 hover:text-slate-700 dark:hover:text-slate-400",
              )}
              onClick={() => !isCenter && item.view && onNavigate(item.view)}
              type="button"
            >
              {isActive && (
                <motion.span
                  layoutId="mobile-nav-active"
                  className="absolute inset-0 rounded-xl bg-indigo-500/10"
                  transition={springTransition}
                />
              )}
              {isCenter ? (
                <span className="relative z-10 flex size-8 items-center justify-center rounded-xl bg-indigo-500 text-white shadow-lg shadow-indigo-500/25">
                  <Icon className="size-4" />
                </span>
              ) : (
                <Icon className="relative z-10 size-4" />
              )}
              <span className="relative z-10 w-full truncate text-center">{item.label}</span>
            </motion.button>
          );
        })}
      </LayoutGroup>
      <motion.button
        whileTap={{ scale: 0.93 }}
        className="grid flex-1 justify-items-center gap-1 rounded-xl py-2 text-[11px] font-medium text-slate-500 transition-colors hover:text-slate-700 dark:hover:text-slate-400"
        onClick={onSignOut}
        type="button"
      >
        <LogOut className="size-4" />
        <span className="w-full truncate text-center">Thoát</span>
      </motion.button>
    </nav>
  );
}
