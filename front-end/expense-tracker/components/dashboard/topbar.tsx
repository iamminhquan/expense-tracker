"use client";

import { Bell, LogOut, Moon, Search, Sparkles, Sun, WalletCards } from "lucide-react";
import { useTheme } from "next-themes";
import type { MouseEvent } from "react";

import { Button } from "@/components/ui/button";
import type { AuthUser } from "@/lib/auth";
import { cn } from "@/lib/utils";
import type { ExpenseSummary } from "./utils";
import { formatCompactCurrency } from "./utils";

export function Topbar({
  isLoading,
  onSignOut,
  summary,
  user,
}: {
  isLoading: boolean;
  onSignOut: () => void;
  summary: ExpenseSummary;
  user: AuthUser | null;
}) {
  const { resolvedTheme, setTheme } = useTheme();
  const isDark = (resolvedTheme ?? "dark") === "dark";

  function toggleTheme(e: MouseEvent<HTMLButtonElement>) {
    const nextTheme = isDark ? "light" : "dark";

    if (typeof document === "undefined") {
      setTheme(nextTheme);
      return;
    }

    // Circular reveal from button click position (View Transitions API)
    // Use a boolean to avoid TypeScript narrowing `document` to `never` in the fallback branch
    const supportsVT = "startViewTransition" in (document as object);
    if (supportsVT) {
      const { clientX: x, clientY: y } = e;
      const endRadius = Math.hypot(
        Math.max(x, window.innerWidth - x),
        Math.max(y, window.innerHeight - y),
      );
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const vt = (document as any).startViewTransition(() => setTheme(nextTheme));
      vt.ready
        .then(() => {
          document.documentElement.animate(
            {
              clipPath: [
                `circle(0px at ${x}px ${y}px)`,
                `circle(${endRadius}px at ${x}px ${y}px)`,
              ],
            },
            {
              duration: 500,
              easing: "ease-in-out",
              pseudoElement: "::view-transition-new(root)",
            },
          );
        })
        .catch(() => {});
      return;
    }

    // Fallback: smooth CSS transition via class
    const root = document.documentElement;
    root.classList.add("theme-transitioning");
    setTheme(nextTheme);
    setTimeout(() => root.classList.remove("theme-transitioning"), 400);
  }

  const points = summary.monthlyPoints;
  const thisMonth = points[points.length - 1]?.total ?? 0;
  const lastMonth = points[points.length - 2]?.total ?? 0;
  const trend = lastMonth > 0 ? ((thisMonth - lastMonth) / lastMonth) * 100 : 8.2;
  const isUp = trend > 0;

  return (
    <header className="sticky top-0 z-20 border-b border-black/[0.06] bg-[var(--dash-shell)]/90 backdrop-blur-md dark:border-white/[0.06]">
      <div className="flex min-w-0 flex-wrap items-center gap-3 px-4 py-3 sm:px-5">
        <div className="flex min-w-0 items-center gap-3 lg:hidden">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-indigo-500/15 text-indigo-600 ring-1 ring-indigo-500/25 dark:text-indigo-300 dark:ring-indigo-400/25">
            <WalletCards className="size-5" />
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-bold text-slate-900 dark:text-slate-100">Nexora</p>
            <p className="truncate text-xs text-slate-500">{user?.email}</p>
          </div>
        </div>

        <label className="order-3 flex h-9 min-w-0 flex-1 basis-full items-center gap-2 rounded-xl border border-black/[0.07] bg-black/[0.03] px-3 text-sm text-slate-500 transition-colors focus-within:border-indigo-500/40 focus-within:bg-dash-input dark:border-white/[0.07] dark:bg-white/[0.03] sm:order-none sm:basis-72 lg:max-w-sm">
          <Search className="size-4 shrink-0 text-slate-500" />
          <input
            className="min-w-0 flex-1 bg-transparent text-slate-900 outline-none placeholder:text-slate-500 dark:text-slate-100 dark:placeholder:text-slate-600"
            placeholder="Tìm khoản chi, danh mục..."
            type="search"
          />
        </label>

        <div className="ml-auto flex items-center gap-2">
          <div
            className={cn(
              "hidden h-8 items-center gap-1.5 rounded-xl border px-2.5 text-xs font-semibold sm:flex",
              isUp
                ? "border-rose-500/20 bg-rose-500/10 text-rose-600 dark:text-rose-400"
                : "border-emerald-500/20 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
            )}
          >
            <Sparkles className="size-3" />
            {isUp ? "+" : ""}{trend.toFixed(1)}%
          </div>
          <button
            aria-label={isDark ? "Chuyển sang chế độ sáng" : "Chuyển sang chế độ tối"}
            className="flex size-9 items-center justify-center rounded-xl border border-black/[0.07] bg-black/[0.02] text-slate-500 transition-colors hover:bg-black/[0.05] hover:text-slate-700 dark:border-white/[0.07] dark:bg-white/[0.02] dark:hover:bg-white/[0.05] dark:hover:text-slate-300"
            onClick={toggleTheme}
            type="button"
          >
            {isDark ? <Sun className="size-4" /> : <Moon className="size-4" />}
          </button>
          <button
            aria-label="Thông báo"
            className="flex size-9 items-center justify-center rounded-xl border border-black/[0.07] bg-black/[0.02] text-slate-500 transition-colors hover:bg-black/[0.05] hover:text-slate-700 dark:border-white/[0.07] dark:bg-white/[0.02] dark:hover:bg-white/[0.05] dark:hover:text-slate-300"
            type="button"
          >
            <Bell className="size-4" />
          </button>
          <div className="hidden rounded-xl border border-black/[0.07] bg-black/[0.02] px-3 py-1.5 text-right dark:border-white/[0.07] dark:bg-white/[0.02] sm:block">
            <p className="text-[10px] text-slate-500">Tháng này</p>
            <p className="text-sm font-bold tabular-nums text-slate-800 dark:text-slate-200">
              {formatCompactCurrency(summary.monthTotal)}
            </p>
          </div>
          <Button
            aria-label="Đăng xuất"
            className="rounded-xl border border-black/[0.07] bg-black/[0.02] text-slate-500 hover:bg-black/[0.06] hover:text-slate-700 dark:border-white/[0.07] dark:bg-white/[0.02] dark:hover:bg-white/[0.06] dark:hover:text-slate-200 lg:hidden"
            disabled={isLoading}
            onClick={onSignOut}
            size="icon"
            type="button"
            variant="outline"
          >
            <LogOut className="size-4" />
          </Button>
        </div>
      </div>
    </header>
  );
}
