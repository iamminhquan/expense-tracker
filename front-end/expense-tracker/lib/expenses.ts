import { API_BASE_URL } from "@/lib/auth";

export type Expense = {
  id: number;
  amount: number;
  currency: string;
  description: string;
  category: string;
  date: string;
  payment_method: string;
  is_recurring: boolean;
  recurring_period: string;
  note: string;
};

export type ExpenseForm = {
  amount: string;
  currency: string;
  description: string;
  category: string;
  date: string;
  paymentMethod: string;
  isRecurring: string;
  recurringPeriod: string;
  note: string;
};

export type ExpenseResponse = {
  expense?: Expense;
  error?: string;
};

export type ExpenseListResponse = {
  expenses?: Expense[];
  error?: string;
};

export const initialExpenseForm: ExpenseForm = {
  amount: "",
  currency: "VND",
  description: "",
  category: "",
  date: new Date().toISOString().slice(0, 10),
  paymentMethod: "",
  isRecurring: "false",
  recurringPeriod: "",
  note: "",
};

export async function fetchExpenses(token: string) {
  const response = await fetch(`${API_BASE_URL}/expenses`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const data = (await response.json()) as ExpenseListResponse;

  return { data, response };
}

export async function createExpense(token: string, form: ExpenseForm) {
  const response = await fetch(`${API_BASE_URL}/expenses`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payloadFromForm(form)),
  });
  const data = (await response.json()) as ExpenseResponse;

  return { data, response };
}

export async function updateExpense(
  token: string,
  id: number,
  form: ExpenseForm,
) {
  const response = await fetch(`${API_BASE_URL}/expenses/${id}`, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payloadFromForm(form)),
  });
  const data = (await response.json()) as ExpenseResponse;

  return { data, response };
}

export async function deleteExpense(token: string, id: number) {
  return fetch(`${API_BASE_URL}/expenses/${id}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
}

export function formFromExpense(expense: Expense): ExpenseForm {
  return {
    amount: String(expense.amount),
    currency: expense.currency,
    description: expense.description,
    category: expense.category,
    date: expense.date,
    paymentMethod: expense.payment_method,
    isRecurring: String(expense.is_recurring),
    recurringPeriod: expense.recurring_period,
    note: expense.note,
  };
}

function payloadFromForm(form: ExpenseForm) {
  return {
    amount: Number(form.amount),
    currency: form.currency,
    description: form.description,
    category: form.category,
    date: form.date,
    payment_method: form.paymentMethod,
    is_recurring: form.isRecurring === "true",
    recurring_period: form.recurringPeriod,
    note: form.note,
  };
}
