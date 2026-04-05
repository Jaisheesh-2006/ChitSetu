"use client";
import React from "react";
import { getSupabaseClient } from "@/lib/supabase";

type Variant = "signin" | "signup";
export default function GoogleAuthBtn({ variant }: { variant?: Variant }) {
  // auto-detect variant from URL when not provided
  let resolved: Variant = variant ?? "signin";
  try {
    const p = typeof window !== "undefined" ? window.location.pathname.toLowerCase() : "";
    const s = typeof window !== "undefined" ? new URLSearchParams(window.location.search) : null;
    const tab = s?.get("tab")?.toLowerCase();
    if (!variant) {
      if (p.includes("register") || p.includes("signup") || tab === "register" || tab === "signup") resolved = "signup";
      else resolved = "signin";
    }
  } catch {}
  const go = async () => {
    const url = process.env.NEXT_PUBLIC_AUTH_CALLBACK_URL
      ?? (typeof window !== "undefined" ? `${window.location.origin}/auth/callback` : "");
    if (!url) throw new Error("Google auth callback URL unavailable");
    const { error } = await getSupabaseClient().auth.signInWithOAuth({ provider: "google", options: { redirectTo: url } });
    if (error) throw error;
  };
  return (
    <button type="button" onClick={() => go().catch((e: unknown) => alert(e instanceof Error ? e.message : "Failed"))}
      style={{
        width: "100%", display: "flex", alignItems: "center", justifyContent: "center", gap: 8,
        padding: "10px 16px", borderRadius: 6, border: "none",
        background: "var(--color-bg-subtle)", color: "var(--color-text-secondary)",
        fontSize: 13, fontWeight: 500, cursor: "pointer", fontFamily: '"Inter",sans-serif',
        boxShadow: "var(--shadow-card)", transition: "box-shadow 0.2s, transform 0.2s",
      }}
      onMouseEnter={(e) => { e.currentTarget.style.boxShadow = "var(--shadow-elevated)"; e.currentTarget.style.transform = "translateY(-1px)"; }}
      onMouseLeave={(e) => { e.currentTarget.style.boxShadow = "var(--shadow-card)"; e.currentTarget.style.transform = "translateY(0)"; }}
    >
      <svg width="16" height="16" viewBox="0 0 24 24"><path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4" /><path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853" /><path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05" /><path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335" /></svg>
      {resolved === "signin" ? "Sign in with Google" : "Sign up with Google"}
    </button>
  );
}
