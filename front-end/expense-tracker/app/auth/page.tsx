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

export default function AuthPage() {
  const router = useRouter();
  const [mode, setMode] = useState<AuthMode>("login");
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
          router.replace("/");
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
    if (nextMode === mode || isLoading) {
      return;
    }

    setMode(nextMode);
    setMessage("");
    setMessageTone("neutral");
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
      const response = await fetch(`${API_BASE_URL}${endpointFor(mode)}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payloadFor(mode, form)),
      });
      const data = (await response.json()) as AuthResponse;

      if (!response.ok) {
        throw new Error(data.error ?? "Authentication failed");
      }
      if (!data.token || !data.user) {
        throw new Error("Authentication response is missing session data");
      }

      saveToken(data.token);
      setForm(initialAuthForm);
      router.replace("/");
    } catch (error) {
      setMessage(
        error instanceof Error ? error.message : "Authentication failed",
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
      mode={mode}
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
