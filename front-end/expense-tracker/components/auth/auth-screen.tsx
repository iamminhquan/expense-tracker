"use client";

import type { ChangeEvent, FormEvent } from "react";
import type { LucideIcon } from "lucide-react";
import {
  ArrowRight,
  BadgeCheck,
  LockKeyhole,
  LogIn,
  Mail,
  ShieldCheck,
  User,
  UserPlus,
  WalletCards,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export type AuthMode = "login" | "register";

export type AuthForm = {
  name: string;
  email: string;
  password: string;
};

export const initialAuthForm: AuthForm = {
  name: "",
  email: "",
  password: "",
};

const modes: Array<{ label: string; value: AuthMode }> = [
  { label: "Login", value: "login" },
  { label: "Register", value: "register" },
];

const inputClassName =
  "h-11 w-full rounded-md border border-input bg-background px-10 text-sm outline-none transition-shadow focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-60";

export function AuthScreen({
  form,
  isLoading,
  message,
  messageTone,
  mode,
  onFieldChange,
  onModeChange,
  onSubmit,
}: {
  form: AuthForm;
  isLoading: boolean;
  message: string;
  messageTone: "neutral" | "error";
  mode: AuthMode;
  onFieldChange: (
    field: keyof AuthForm,
  ) => (event: ChangeEvent<HTMLInputElement>) => void;
  onModeChange: (mode: AuthMode) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <main className="min-h-screen bg-background px-4 py-6 text-foreground sm:px-6">
      <div className="mx-auto grid min-h-[calc(100vh-3rem)] w-full max-w-6xl items-center gap-8 lg:grid-cols-[1fr_420px]">
        <AuthIntro />
        <AuthPanel
          form={form}
          isLoading={isLoading}
          message={message}
          messageTone={messageTone}
          mode={mode}
          onFieldChange={onFieldChange}
          onModeChange={onModeChange}
          onSubmit={onSubmit}
        />
      </div>
    </main>
  );
}

function AuthPanel({
  form,
  isLoading,
  message,
  messageTone,
  mode,
  onFieldChange,
  onModeChange,
  onSubmit,
}: {
  form: AuthForm;
  isLoading: boolean;
  message: string;
  messageTone: "neutral" | "error";
  mode: AuthMode;
  onFieldChange: (
    field: keyof AuthForm,
  ) => (event: ChangeEvent<HTMLInputElement>) => void;
  onModeChange: (mode: AuthMode) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <section className="rounded-md border border-border bg-card shadow-sm">
      <AuthPanelHeader mode={mode} />

      <div className="p-5">
        <ModeSwitch
          disabled={isLoading}
          mode={mode}
          onModeChange={onModeChange}
        />

        <AuthFormFields
          form={form}
          isLoading={isLoading}
          mode={mode}
          onFieldChange={onFieldChange}
          onSubmit={onSubmit}
        />

        {message ? (
          <StatusMessage message={message} tone={messageTone} />
        ) : null}

        <ModePrompt
          disabled={isLoading}
          mode={mode}
          onModeChange={onModeChange}
        />
      </div>
    </section>
  );
}

function AuthPanelHeader({ mode }: { mode: AuthMode }) {
  return (
    <div className="border-b border-border px-5 py-5">
      <div className="flex items-center gap-3">
        <div className="flex size-10 items-center justify-center rounded-md border border-border bg-background">
          <WalletCards className="size-5" />
        </div>
        <div>
          <h1 className="text-lg font-semibold tracking-normal">
            {mode === "login" ? "Sign in" : "Create account"}
          </h1>
          <p className="text-sm text-muted-foreground">
            {mode === "login"
              ? "Continue to your expense workspace."
              : "Start tracking budgets with a secure account."}
          </p>
        </div>
      </div>
    </div>
  );
}

function AuthFormFields({
  form,
  isLoading,
  mode,
  onFieldChange,
  onSubmit,
}: {
  form: AuthForm;
  isLoading: boolean;
  mode: AuthMode;
  onFieldChange: (
    field: keyof AuthForm,
  ) => (event: ChangeEvent<HTMLInputElement>) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <form className="space-y-4" onSubmit={onSubmit}>
      <NameField
        disabled={isLoading}
        isVisible={mode === "register"}
        onChange={onFieldChange("name")}
        value={form.name}
      />

      <TextField
        autoComplete="email"
        disabled={isLoading}
        icon={Mail}
        label="Email"
        onChange={onFieldChange("email")}
        placeholder="you@example.com"
        type="email"
        value={form.email}
      />

      <TextField
        autoComplete={mode === "login" ? "current-password" : "new-password"}
        disabled={isLoading}
        hint={mode === "register" ? "Use at least 8 characters." : undefined}
        icon={LockKeyhole}
        label="Password"
        onChange={onFieldChange("password")}
        placeholder="Enter your password"
        type="password"
        value={form.password}
      />

      <Button className="h-11 w-full" disabled={isLoading} type="submit">
        {mode === "login" ? (
          <LogIn className="size-4" />
        ) : (
          <UserPlus className="size-4" />
        )}
        {submitLabel(mode, isLoading)}
      </Button>
    </form>
  );
}

function AuthIntro() {
  return (
    <section className="space-y-6">
      <div className="inline-flex items-center gap-2 rounded-md border border-border px-3 py-1 text-sm text-muted-foreground">
        <ShieldCheck className="size-4 text-emerald-600" />
        Budget Expense Tracker
      </div>

      <div className="max-w-2xl space-y-4">
        <h2 className="text-4xl font-semibold tracking-normal sm:text-5xl">
          Your expenses, budgets, and cash flow in one place.
        </h2>
        <p className="max-w-xl text-base leading-7 text-muted-foreground">
          Sign in to review spending, create budgets, and keep personal finance
          activity organized without extra noise.
        </p>
      </div>

      <div className="grid max-w-2xl gap-3 sm:grid-cols-3">
        <IntroItem label="Secure login" />
        <IntroItem label="Budget overview" />
        <IntroItem label="Expense history" />
      </div>
    </section>
  );
}

function IntroItem({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-2 rounded-md border border-border bg-card px-3 py-3 text-sm">
      <BadgeCheck className="size-4 shrink-0 text-emerald-600" />
      <span>{label}</span>
    </div>
  );
}

function ModeSwitch({
  disabled,
  mode,
  onModeChange,
}: {
  disabled: boolean;
  mode: AuthMode;
  onModeChange: (mode: AuthMode) => void;
}) {
  return (
    <div className="relative mb-5 grid grid-cols-2 overflow-hidden rounded-md bg-muted p-1">
      <span
        className={cn(
          "absolute inset-y-1 left-1 w-[calc(50%-0.25rem)] rounded-md bg-background shadow-sm transition-transform duration-300 ease-out",
          mode === "register" && "translate-x-full",
        )}
      />
      {modes.map((item) => (
        <button
          key={item.value}
          aria-pressed={mode === item.value}
          className={cn(
            "relative z-10 h-9 rounded-md px-3 text-sm font-medium transition-colors",
            mode === item.value ? "text-foreground" : "text-muted-foreground",
          )}
          disabled={disabled}
          onClick={() => onModeChange(item.value)}
          type="button"
        >
          {item.label}
        </button>
      ))}
    </div>
  );
}

function NameField({
  disabled,
  isVisible,
  onChange,
  value,
}: {
  disabled: boolean;
  isVisible: boolean;
  onChange: (event: ChangeEvent<HTMLInputElement>) => void;
  value: string;
}) {
  return (
    <div
      aria-hidden={!isVisible}
      className={cn(
        "grid transition-[grid-template-rows,opacity,transform] duration-300 ease-out",
        isVisible
          ? "grid-rows-[1fr] opacity-100 translate-y-0"
          : "grid-rows-[0fr] opacity-0 -translate-y-1",
      )}
    >
      <TextField
        autoComplete="name"
        disabled={disabled || !isVisible}
        icon={User}
        label="Name"
        onChange={onChange}
        placeholder="Your full name"
        tabIndex={isVisible ? 0 : -1}
        value={value}
        wrapperClassName="min-h-0 overflow-hidden"
      />
    </div>
  );
}

function TextField({
  autoComplete,
  disabled,
  hint,
  icon: Icon,
  label,
  onChange,
  placeholder,
  tabIndex,
  type = "text",
  value,
  wrapperClassName,
}: {
  autoComplete: string;
  disabled: boolean;
  hint?: string;
  icon: LucideIcon;
  label: string;
  onChange: (event: ChangeEvent<HTMLInputElement>) => void;
  placeholder: string;
  tabIndex?: number;
  type?: string;
  value: string;
  wrapperClassName?: string;
}) {
  return (
    <label
      className={cn("block space-y-2 text-sm font-medium", wrapperClassName)}
    >
      <span>{label}</span>
      <span className="relative block">
        <Icon className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <input
          autoComplete={autoComplete}
          className={inputClassName}
          disabled={disabled}
          onChange={onChange}
          placeholder={placeholder}
          tabIndex={tabIndex}
          type={type}
          value={value}
        />
      </span>
      {hint ? (
        <span className="block text-xs text-muted-foreground">{hint}</span>
      ) : null}
    </label>
  );
}

function StatusMessage({
  message,
  tone,
}: {
  message: string;
  tone: "neutral" | "error";
}) {
  return (
    <p
      className={cn(
        "mt-4 rounded-md border px-3 py-2 text-sm",
        tone === "error"
          ? "border-destructive/30 bg-destructive/10 text-destructive"
          : "border-border bg-muted text-muted-foreground",
      )}
    >
      {message}
    </p>
  );
}

function ModePrompt({
  disabled,
  mode,
  onModeChange,
}: {
  disabled: boolean;
  mode: AuthMode;
  onModeChange: (mode: AuthMode) => void;
}) {
  return (
    <p className="mt-5 text-center text-sm text-muted-foreground">
      {mode === "login" ? "New to the app?" : "Already have an account?"}{" "}
      <button
        className="font-medium text-foreground underline-offset-4 hover:underline"
        disabled={disabled}
        onClick={() => onModeChange(mode === "login" ? "register" : "login")}
        type="button"
      >
        {mode === "login" ? "Create an account" : "Sign in instead"}
      </button>
    </p>
  );
}

function submitLabel(mode: AuthMode, isLoading: boolean) {
  if (isLoading) {
    return "Please wait";
  }

  return (
    <>
      {mode === "login" ? "Sign in" : "Create account"}
      <ArrowRight className="size-4" />
    </>
  );
}
