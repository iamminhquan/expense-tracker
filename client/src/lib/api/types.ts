// Hand-written to match server/internal/api's DTOs; update both together.

export interface User {
  id: number
  name: string
  email: string
  username: string
  theme: Theme
  emailVerified: boolean
}

export interface AuthResponse {
  accessToken: string
  expiresAt: string
  user: User
}

export interface Category {
  id: number
  name: string
  type: 'expense' | 'income'
  color: string
  transactionCount: number
  isDefault: boolean
}

export interface CategoriesResponse {
  expenseCategories: Category[]
  incomeCategories: Category[]
  hasCustomCategories: boolean
}

export interface MonthOption {
  value: string
  label: string
}

export interface Transaction {
  id: number
  categoryId: number
  categoryName: string
  categoryColor: string
  description: string
  amount: number
  type: 'expense' | 'income'
  occurredOn: string
  isDuplicate: boolean
}

export interface TransactionsResponse {
  transactions: Transaction[]
  totalCount: number
  page: number
  totalPages: number
  hasPrev: boolean
  hasNext: boolean
  monthValue: string
  monthLabel: string
  allMonths: boolean
  currentMonthValue: string
  availableMonths: MonthOption[]
}

export interface TransactionFilters {
  month?: string
  page?: number
  q?: string
  type?: 'expense' | 'income'
  category?: number
  min?: number
  max?: number
  sort?: 'amount_desc' | 'amount_asc'
}

export interface TransactionWrite {
  categoryId: number
  amount: number
  occurredOn: string
  description: string
  type?: 'expense' | 'income'
}

export interface Balance {
  remaining: number
  spentPct: number
  hasIncome: boolean
  empty: boolean
}

export interface PieLegendEntry {
  name: string
  color: string
  percent: number
  amount: number
}

export interface PieData {
  labels: string[]
  values: number[]
  colors: string[]
  legend: PieLegendEntry[]
}

export interface BarData {
  labels: string[]
  expense: number[]
  income: number[]
}

export interface DashboardResponse {
  monthValue: string
  monthLabel: string
  currentMonthValue: string
  availableMonths: MonthOption[]
  totalExpense: number
  totalIncome: number
  previousTotalExpense: number
  previousTotalIncome: number
  hasPreviousMonthData: boolean
  currentMonthEmpty: boolean
  headerBalance: Balance
  pie: PieData
  bar: BarData
}

export interface Session {
  id: string
  device: string
  createdAt: string
  isCurrent: boolean
}

export interface SettingsResponse {
  name: string
  username: string
  email: string
  pendingEmail?: string
  sessions: Session[]
}

export type Theme = 'auto' | 'light' | 'dark'
