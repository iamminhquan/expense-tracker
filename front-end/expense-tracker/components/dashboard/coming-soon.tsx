"use client";

import {
  BarChart3,
  CreditCard,
  HelpCircle,
  History,
  Settings,
  Target,
  WalletCards,
} from "lucide-react";
import { type Variants, motion } from "framer-motion";

import type { DashboardView } from "./sidebar";

const VIEW_META: Record<
  Exclude<DashboardView, "overview">,
  { label: string; desc: string; icon: React.ElementType }
> = {
  wallet: {
    label: "Ví của tôi",
    desc: "Quản lý các tài khoản và ví điện tử của bạn ở một nơi duy nhất.",
    icon: WalletCards,
  },
  payments: {
    label: "Thanh toán",
    desc: "Lên lịch và theo dõi các khoản thanh toán định kỳ.",
    icon: CreditCard,
  },
  analytics: {
    label: "Phân tích",
    desc: "Báo cáo chi tiết và biểu đồ về thói quen chi tiêu của bạn.",
    icon: BarChart3,
  },
  goals: {
    label: "Mục tiêu",
    desc: "Đặt và theo dõi mục tiêu tiết kiệm tài chính cá nhân.",
    icon: Target,
  },
  history: {
    label: "Lịch sử",
    desc: "Toàn bộ lịch sử giao dịch với bộ lọc và tìm kiếm nâng cao.",
    icon: History,
  },
  settings: {
    label: "Cài đặt",
    desc: "Tuỳ chỉnh tài khoản, thông báo và giao diện.",
    icon: Settings,
  },
  help: {
    label: "Trợ giúp",
    desc: "Hướng dẫn sử dụng, câu hỏi thường gặp và hỗ trợ khách hàng.",
    icon: HelpCircle,
  },
};

const stagger: Variants = {
  hidden: {},
  show: { transition: { staggerChildren: 0.07 } },
};

const fadeUp: Variants = {
  hidden: { opacity: 0, y: 14 },
  show: { opacity: 1, y: 0, transition: { duration: 0.28, ease: "easeOut" } },
};

export function ComingSoon({ view }: { view: Exclude<DashboardView, "overview"> }) {
  const meta = VIEW_META[view];
  const Icon = meta.icon;

  return (
    <div className="grid min-h-[60vh] place-items-center p-8">
      <motion.div
        variants={stagger}
        initial="hidden"
        animate="show"
        className="text-center"
      >
        <motion.div
          variants={{
            hidden: { opacity: 0, scale: 0.6 },
            show: {
              opacity: 1,
              scale: 1,
              transition: { type: "spring", stiffness: 380, damping: 22 },
            },
          }}
          className="mx-auto mb-6 grid size-20 place-items-center rounded-2xl bg-indigo-500/10 text-indigo-600 ring-1 ring-indigo-500/20 dark:text-indigo-300"
        >
          <Icon className="size-9" />
        </motion.div>

        <motion.h2
          variants={fadeUp}
          className="text-xl font-bold text-slate-900 dark:text-slate-100"
        >
          {meta.label}
        </motion.h2>

        <motion.p
          variants={fadeUp}
          className="mx-auto mt-2 max-w-sm text-sm leading-relaxed text-slate-500"
        >
          {meta.desc}
        </motion.p>

        <motion.div
          variants={{
            hidden: { opacity: 0, scale: 0.88 },
            show: { opacity: 1, scale: 1, transition: { duration: 0.22, ease: "easeOut" } },
          }}
          className="mt-6 inline-flex items-center gap-2 rounded-xl border border-indigo-500/20 bg-indigo-500/[0.07] px-4 py-2 text-sm font-semibold text-indigo-600 dark:text-indigo-300"
        >
          <span className="size-1.5 animate-pulse rounded-full bg-indigo-500" />
          Đang phát triển
        </motion.div>
      </motion.div>
    </div>
  );
}
