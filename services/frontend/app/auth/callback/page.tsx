"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import CircularProgress from "@mui/material/CircularProgress";
import { getSupabaseClient } from "@/lib/supabase";
import { useAuth, type AuthUser } from "@/hooks/useAuth";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export default function AuthCallbackPage() {
  const router = useRouter();
  const { setOAuthSession } = useAuth();

  useEffect(() => {
    let active = true;

    async function handleCallback() {
      try {
        const supabase = getSupabaseClient();

        // Supabase client reads OAuth callback params and restores session.
        const { data, error } = await supabase.auth.getSession();
        if (error || !data.session) {
          router.replace("/login");
          return;
        }

        const accessToken = data.session.access_token;

        // Verify token with backend and trigger local user upsert.
        const verifyRes = await fetch(`${API_BASE}/auth/verify`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ access_token: accessToken }),
        });
        if (!verifyRes.ok) {
          router.replace("/login");
          return;
        }

        const tokenData = await verifyRes.json();

        const user = data.session.user;
        const metadata = user.user_metadata || {};
        const appUser: AuthUser = {
          id: user.id,
          email: user.email || "",
          full_name: metadata.full_name || metadata.name || undefined,
          avatar_url: metadata.avatar_url || metadata.picture || undefined,
        };

        if (!active) {
          return;
        }

        setOAuthSession(tokenData.access_token, tokenData.refresh_token, appUser);
        router.replace("/dashboard");
      } catch {
        router.replace("/login");
      }
    }

    handleCallback();
    return () => {
      active = false;
    };
  }, [router, setOAuthSession]);

  return (
    <div
      className="flex min-h-screen items-center justify-center"
      style={{ backgroundColor: "var(--color-bg)" }}
    >
      <CircularProgress sx={{ color: "var(--color-accent)" }} />
    </div>
  );
}
