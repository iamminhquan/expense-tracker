import type { Expense } from "@/lib/expenses";

export interface MonthlyPoint {
  key: string;
  label: string;
  total: number;
}

export interface ExpenseSummary {
  categoryBreakdown: { name: string; total: number }[];
  count: number;
  monthCount: number;
  monthTotal: number;
  monthlyPoints: MonthlyPoint[];
  total: number;
}

export function summarizeExpenses(expenses: Expense[]): ExpenseSummary {
  const now = new Date();
  const monthPrefix = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
  const totalsByCategory = new Map<string, number>();
  const monthlyPoints = buildMonthlyPoints();

  let total = 0;
  let monthTotal = 0;
  let monthCount = 0;

  for (const expense of expenses) {
    total += expense.amount;
    totalsByCategory.set(
      expense.category,
      (totalsByCategory.get(expense.category) ?? 0) + expense.amount,
    );
    if (expense.date.startsWith(monthPrefix)) {
      monthTotal += expense.amount;
      monthCount++;
    }
    const point = monthlyPoints.find((p) => expense.date.startsWith(p.key));
    if (point) point.total += expense.amount;
  }

  const categoryBreakdown = Array.from(totalsByCategory, ([name, amount]) => ({
    name,
    total: amount,
  })).sort((a, b) => b.total - a.total);

  return { categoryBreakdown, count: expenses.length, monthCount, monthTotal, monthlyPoints, total };
}

export function buildMonthlyPoints(): MonthlyPoint[] {
  const formatter = new Intl.DateTimeFormat("vi-VN", { month: "short" });
  const points: MonthlyPoint[] = [];
  const today = new Date();
  for (let offset = 5; offset >= 0; offset--) {
    const date = new Date(today.getFullYear(), today.getMonth() - offset, 1);
    points.push({
      key: `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`,
      label: formatter.format(date).replace(".", ""),
      total: 0,
    });
  }
  return points;
}

export function formatCurrency(amount: number): string {
  return new Intl.NumberFormat("vi-VN", {
    currency: "VND",
    maximumFractionDigits: 0,
    style: "currency",
  }).format(amount);
}

export function formatCompactCurrency(amount: number): string {
  return new Intl.NumberFormat("vi-VN", {
    currency: "VND",
    maximumFractionDigits: 1,
    notation: "compact",
    style: "currency",
  }).format(amount);
}

export function formatDate(date: string): string {
  return new Intl.DateTimeFormat("vi-VN", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  }).format(new Date(`${date}T00:00:00`));
}

export function initials(name?: string): string {
  if (!name) return "U";
  return name
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join("");
}

export const CATEGORY_CHART_COLORS = [
  "#818cf8",
  "#a78bfa",
  "#f472b6",
  "#34d399",
  "#fbbf24",
  "#fb7185",
];

export interface CategoryStyle {
  text: string;
  bg: string;
  hex: string;
}

export const CATEGORY_STYLE_MAP: Record<string, CategoryStyle> = {
  "Ăn uống": { text: "text-rose-400", bg: "bg-rose-400/10", hex: "#fb7185" },
  "Đi lại": { text: "text-sky-400", bg: "bg-sky-400/10", hex: "#38bdf8" },
  "Nhà cửa": { text: "text-amber-400", bg: "bg-amber-400/10", hex: "#fbbf24" },
  "Mua sắm": { text: "text-violet-400", bg: "bg-violet-400/10", hex: "#a78bfa" },
  "Sức khỏe": { text: "text-emerald-400", bg: "bg-emerald-400/10", hex: "#34d399" },
  "Giải trí": { text: "text-pink-400", bg: "bg-pink-400/10", hex: "#f472b6" },
  "Học tập": { text: "text-cyan-400", bg: "bg-cyan-400/10", hex: "#22d3ee" },
  "Gia đình": { text: "text-orange-400", bg: "bg-orange-400/10", hex: "#fb923c" },
  "Hóa đơn": { text: "text-red-400", bg: "bg-red-400/10", hex: "#f87171" },
  "Khác": { text: "text-slate-400", bg: "bg-slate-400/10", hex: "#94a3b8" },
};

export function getCategoryStyle(category: string): CategoryStyle {
  return CATEGORY_STYLE_MAP[category] ?? { text: "text-slate-400", bg: "bg-slate-400/10", hex: "#94a3b8" };
}
