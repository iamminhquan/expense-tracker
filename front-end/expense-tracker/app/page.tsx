"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { DashboardShell } from "@/components/dashboard/dashboard-shell";
import {
  clearToken,
  fetchProfile,
  getStoredToken,
  type AuthUser,
} from "@/lib/auth";

export default function HomePage() {
  const router = useRouter();
  const [user, setUser] = useState<AuthUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const redirectToAuth = useCallback(() => {
    clearToken();
    router.replace("/auth");
  }, [router]);

  const signOut = useCallback(() => {
    redirectToAuth();
  }, [redirectToAuth]);

  useEffect(() => {
    const token = getStoredToken();
    if (!token) {
      redirectToAuth();
      return;
    }

    const storedToken = token;
    let isCurrent = true;

    async function loadProfile() {
      try {
        const { data, response } = await fetchProfile(storedToken);

        if (!isCurrent) {
          return;
        }
        if (!response.ok || !data.user) {
          redirectToAuth();
          return;
        }

        setUser(data.user);
      } catch {
        if (isCurrent) {
          redirectToAuth();
        }
      } finally {
        if (isCurrent) {
          setIsLoading(false);
        }
      }
    }

    loadProfile();

    return () => {
      isCurrent = false;
    };
  }, [redirectToAuth]);

  if (isLoading) {
    return <DashboardShell user={null} onSignOut={signOut} isLoading />;
  }

  if (!user) {
    return null;
  }

  return <DashboardShell user={user} onSignOut={signOut} />;
}
