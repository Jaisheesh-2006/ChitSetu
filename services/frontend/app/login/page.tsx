"use client";

import React, { useState } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import Navbar from "@/components/Navbar";
import GoogleAuthBtn from "@/components/GoogleAuthBtn";
import AnimatedButton from "@/components/ui/AnimatedButton";
import { useAuth } from "@/hooks/useAuth";

type AuthTab = "login" | "register";

export default function LoginPage() {
  const [tab, setTab] = useState<AuthTab>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const { login, register, isAuthenticated } = useAuth();
  const router = useRouter();

  React.useEffect(() => {
    if (isAuthenticated) router.push("/dashboard");
  }, [isAuthenticated, router]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (!email.trim() || !password.trim()) {
      setError("Fill in all fields.");
      return;
    }
    if (tab === "register") {
      if (password.length < 8) {
        setError("Min 8 characters.");
        return;
      }
      if (password !== confirm) {
        setError("Passwords don't match.");
        return;
      }
    }
    setLoading(true);
    try {
      if (tab === "login") {
        await login(email, password);
      } else {
        await register(email, password);
      }
      router.push("/dashboard");
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Error");
    } finally {
      setLoading(false);
    }
  };

  const sw = (t: AuthTab) => {
    setTab(t);
    setError("");
    setPassword("");
    setConfirm("");
  };

  return (
    <div style={{ background: "var(--color-bg)", minHeight: "100vh", overflow: "hidden" }}>
      <Navbar />

      {/* Ambient background orbs */}
      <div style={{ position: "fixed", inset: 0, pointerEvents: "none", zIndex: 0 }}>
        <motion.div
          animate={{ x: [0, 30, 0], y: [0, -20, 0] }}
          transition={{ repeat: Infinity, duration: 12, ease: "easeInOut" }}
          style={{
            position: "absolute", top: "10%", left: "15%", width: 300, height: 300,
            borderRadius: "50%",
            background: "radial-gradient(circle, rgba(249,115,22,0.04), transparent)",
            filter: "blur(60px)",
          }}
        />
        <motion.div
          animate={{ x: [0, -20, 0], y: [0, 25, 0] }}
          transition={{ repeat: Infinity, duration: 15, ease: "easeInOut" }}
          style={{
            position: "absolute", bottom: "20%", right: "10%", width: 250, height: 250,
            borderRadius: "50%",
            background: "radial-gradient(circle, rgba(96,165,250,0.03), transparent)",
            filter: "blur(50px)",
          }}
        />
      </div>

      <main style={{
        position: "relative", zIndex: 1,
        maxWidth: 440, margin: "0 auto",
        padding: "80px 20px 40px",
        display: "flex", flexDirection: "column",
        alignItems: "center", justifyContent: "center",
        minHeight: "calc(100vh - 52px)",
      }}>
        {/* Auth Card */}
        <motion.div
          initial={{ opacity: 0, y: 30, scale: 0.96 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          transition={{ duration: 0.6, delay: 0.15, ease: [0.16, 1, 0.3, 1] }}
          style={{ width: "100%", maxWidth: 400 }}
        >
          <div style={{
            position: "relative", overflow: "hidden",
            background: "var(--color-bg-card)", borderRadius: 12, border: "none",
            boxShadow: "0 16px 48px rgba(0,0,0,0.5), 0 4px 12px rgba(0,0,0,0.3), 0 0 0 1px rgba(255,255,255,0.02)",
          }}>
            {/* Accent bar */}
            <div style={{ height: 2, background: "var(--gradient-primary)" }} />
            {/* Top shine */}
            <div style={{
              position: "absolute", top: 2, left: 0, right: 0, height: 1,
              background: "linear-gradient(90deg, transparent, rgba(255,255,255,0.04), transparent)",
              pointerEvents: "none",
            }} />

            <div style={{ padding: "24px 26px 26px" }}>
              {/* Tabs */}
              <div style={{
                display: "flex", marginBottom: 22,
                background: "var(--color-bg-subtle)", borderRadius: 8,
                padding: 3, boxShadow: "inset 0 1px 3px rgba(0,0,0,0.3)",
              }}>
                {(["login", "register"] as AuthTab[]).map((t) => (
                  <button key={t} onClick={() => sw(t)} style={{
                    flex: 1, padding: "9px 0", borderRadius: 6,
                    fontSize: 12, fontWeight: 600, border: "none", cursor: "pointer",
                    fontFamily: '"Inter",sans-serif',
                    background: tab === t ? "var(--gradient-primary)" : "transparent",
                    color: tab === t ? "#fff" : "var(--color-text-muted)",
                    boxShadow: tab === t ? "0 2px 8px rgba(249,115,22,0.25)" : "none",
                    transition: "all 0.25s",
                  }}>
                    {t === "login" ? "Log in" : "Sign up"}
                  </button>
                ))}
              </div>

              <motion.h2
                key={tab + "-title"}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3 }}
                style={{ fontSize: 20, fontWeight: 700, color: "var(--color-text)", margin: "0 0 4px" }}
              >
                {tab === "login" ? "Welcome back" : "Create account"}
              </motion.h2>
              <p style={{ fontSize: 12, color: "var(--color-text-muted)", margin: "0 0 18px" }}>
                {tab === "login" ? "Enter credentials to continue." : "Sign up to get started."}
              </p>

              <AnimatePresence mode="wait">
                <motion.form
                  key={tab}
                  initial={{ opacity: 0, x: 16 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: -16 }}
                  transition={{ duration: 0.25 }}
                  onSubmit={handleSubmit}
                  style={{ display: "flex", flexDirection: "column", gap: 12 }}
                >
                  <div>
                    <label style={{
                      display: "block", fontSize: 11, fontWeight: 600,
                      color: "var(--color-text-secondary)", marginBottom: 5, letterSpacing: 0.5,
                    }}>EMAIL</label>
                    <input
                      type="email" value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      placeholder="you@example.com" required
                      className="glow-input"
                    />
                  </div>
                  <div>
                    <div style={{
                      display: "flex", justifyContent: "space-between",
                      alignItems: "center", marginBottom: 5,
                    }}>
                      <label style={{
                        fontSize: 11, fontWeight: 600,
                        color: "var(--color-text-secondary)", letterSpacing: 0.5,
                      }}>PASSWORD</label>
                      {tab === "login" && (
                        <button
                          type="button"
                          onClick={() => router.push("/forgot-password")}
                          style={{
                            background: "none", border: "none",
                            color: "var(--color-accent)", fontSize: 10,
                            fontWeight: 600, cursor: "pointer", padding: 0,
                          }}
                        >
                          Forgot Password?
                        </button>
                      )}
                    </div>
                    <input
                      type="password" value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      placeholder="••••••••" required
                      className="glow-input"
                    />
                  </div>
                  {tab === "register" && (
                    <motion.div
                      initial={{ opacity: 0, height: 0 }}
                      animate={{ opacity: 1, height: "auto" }}
                      exit={{ opacity: 0, height: 0 }}
                    >
                      <label style={{
                        display: "block", fontSize: 11, fontWeight: 600,
                        color: "var(--color-text-secondary)",
                        marginBottom: 5, letterSpacing: 0.5,
                      }}>CONFIRM PASSWORD</label>
                      <input
                        type="password" value={confirm}
                        onChange={(e) => setConfirm(e.target.value)}
                        placeholder="••••••••" required
                        className="glow-input"
                      />
                    </motion.div>
                  )}
                  {error && (
                    <motion.div
                      initial={{ opacity: 0, y: -6, scale: 0.95 }}
                      animate={{ opacity: 1, y: 0, scale: 1 }}
                      style={{
                        fontSize: 12, color: "var(--color-danger)",
                        background: "var(--color-danger-light)", borderRadius: 6,
                        padding: "8px 12px",
                        boxShadow: "0 2px 8px rgba(239,68,68,0.1)",
                      }}
                    >
                      {error}
                    </motion.div>
                  )}
                  <AnimatedButton type="submit" variant="primary" fullWidth disabled={loading} size="md">
                    {loading ? "…" : tab === "login" ? "Log in →" : "Create account →"}
                  </AnimatedButton>
                  <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                    <div style={{ flex: 1, height: 1, background: "rgba(255,255,255,0.04)" }} />
                    <span style={{
                      fontSize: 10, color: "var(--color-text-muted)", fontWeight: 600,
                    }}>OR</span>
                    <div style={{ flex: 1, height: 1, background: "rgba(255,255,255,0.04)" }} />
                  </div>
                  <GoogleAuthBtn variant={tab === "login" ? "signin" : "signup"} />
                </motion.form>
              </AnimatePresence>

              <p style={{
                textAlign: "center", fontSize: 11,
                color: "var(--color-text-muted)", marginTop: 14,
              }}>
                {tab === "login" ? "No account? " : "Have an account? "}
                <button
                  onClick={() => sw(tab === "login" ? "register" : "login")}
                  style={{
                    background: "none", border: "none",
                    color: "var(--color-accent)", fontWeight: 600,
                    cursor: "pointer", textDecoration: "underline", fontSize: 11,
                  }}
                >
                  {tab === "login" ? "Sign up" : "Log in"}
                </button>
              </p>
            </div>
          </div>

          {/* Trust badges */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.6 }}
            style={{ display: "flex", gap: 6, justifyContent: "center", marginTop: 14 }}
          >
            {["🔒 256-bit SSL", "🏛 RBI Compliant", "🌐 GDPR Ready"].map((t) => (
              <span key={t} style={{
                fontSize: 9, color: "var(--color-text-muted)", fontWeight: 500,
                background: "var(--color-bg-card)", borderRadius: 4, padding: "3px 8px",
                boxShadow: "var(--shadow-card)",
              }}>{t}</span>
            ))}
          </motion.div>
        </motion.div>
      </main>
    </div>
  );
}
