import { API_BASE_URL } from "@/lib/auth";

export type Expense = {
  id: number;
  amount: number;
  currency: string;
  description: string;
  category: string;
  merchant: string;
  date: string;
  payment_method: string;
  status: string;
  tags: string;
  is_recurring: boolean;
  recurring_period: string;
  attachment_url: string;
  note: string;
};

export type ExpenseForm = {
  amount: string;
  currency: string;
  description: string;
  category: string;
  merchant: string;
  date: string;
  paymentMethod: string;
  status: string;
  tags: string;
  isRecurring: string;
  recurringPeriod: string;
  attachmentUrl: string;
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
  merchant: "",
  date: new Date().toISOString().slice(0, 10),
  paymentMethod: "",
  status: "completed",
  tags: "",
  isRecurring: "false",
  recurringPeriod: "",
  attachmentUrl: "",
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
    merchant: expense.merchant,
    date: expense.date,
    paymentMethod: expense.payment_method,
    status: expense.status,
    tags: expense.tags,
    isRecurring: String(expense.is_recurring),
    recurringPeriod: expense.recurring_period,
    attachmentUrl: expense.attachment_url,
    note: expense.note,
  };
}

function payloadFromForm(form: ExpenseForm) {
  return {
    amount: Number(form.amount),
    currency: form.currency,
    description: form.description,
    category: form.category,
    merchant: form.merchant,
    date: form.date,
    payment_method: form.paymentMethod,
    status: form.status,
    tags: form.tags,
    is_recurring: form.isRecurring === "true",
    recurring_period: form.recurringPeriod,
    attachment_url: form.attachmentUrl,
    note: form.note,
  };
}
