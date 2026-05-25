"use client";

import type { ChangeEvent, FormEvent } from "react";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import {
  AuthScreen,
  initialAuthForm,
  type AuthForm,
  type AuthMode,
} from "@/components/auth/auth-screen";
import {
  API_BASE_URL,
  clearToken,
  fetchProfile,
  getStoredToken,
  saveToken,
  type AuthResponse,
} from "@/lib/auth";

export function AuthPageClient({ initialMode }: { initialMode: AuthMode }) {
  const router = useRouter();
  const [form, setForm] = useState<AuthForm>(initialAuthForm);
  const [message, setMessage] = useState("");
  const [messageTone, setMessageTone] = useState<"neutral" | "error">(
    "neutral",
  );
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    const token = getStoredToken();
    if (!token) {
      return;
    }

    const storedToken = token;
    let isCurrent = true;

    async function verifySession() {
      try {
        const { data, response } = await fetchProfile(storedToken);

        if (!isCurrent) {
          return;
        }
        if (response.ok && data.user) {
          router.replace("/expenses");
          return;
        }

        clearToken();
      } catch {
        if (isCurrent) {
          clearToken();
        }
      }
    }

    verifySession();

    return () => {
      isCurrent = false;
    };
  }, [router]);

  function switchMode(nextMode: AuthMode) {
    if (nextMode === initialMode || isLoading) {
      return;
    }

    router.replace(nextMode === "login" ? "/login" : "/register");
  }

  function updateField(field: keyof AuthForm) {
    return (event: ChangeEvent<HTMLInputElement>) => {
      setForm((currentForm) => ({
        ...currentForm,
        [field]: event.target.value,
      }));
    };
  }

  async function submitAuth(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsLoading(true);
    setMessage("");
    setMessageTone("neutral");

    try {
      const response = await fetch(`${API_BASE_URL}${endpointFor(initialMode)}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payloadFor(initialMode, form)),
      });
      const data = (await response.json()) as AuthResponse;

      if (!response.ok) {
        throw new Error(data.error ?? "Không thể xác thực tài khoản");
      }
      if (!data.token || !data.user) {
        throw new Error("Phản hồi đăng nhập thiếu dữ liệu phiên");
      }

      saveToken(data.token);
      setForm(initialAuthForm);
      router.replace("/expenses");
    } catch (error) {
      setMessage(
        error instanceof Error ? error.message : "Không thể xác thực tài khoản",
      );
      setMessageTone("error");
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <AuthScreen
      form={form}
      isLoading={isLoading}
      message={message}
      messageTone={messageTone}
      mode={initialMode}
      onFieldChange={updateField}
      onModeChange={switchMode}
      onSubmit={submitAuth}
    />
  );
}

function endpointFor(mode: AuthMode) {
  return mode === "login" ? "/auth/login" : "/auth/register";
}

function payloadFor(mode: AuthMode, form: AuthForm) {
  return mode === "login"
    ? { email: form.email, password: form.password }
    : form;
}
