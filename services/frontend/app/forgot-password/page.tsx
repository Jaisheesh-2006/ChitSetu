"use client";

import React, { useState } from "react";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import Navbar from "@/components/Navbar";
import AnimatedButton from "@/components/ui/AnimatedButton";
import { forgotPassword } from "@/services/api";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await forgotPassword(email);
      if (res.token) {
        router.push(`/reset-password?token=${res.token}`);
      } else {
        setSuccess(true);
      }
    } catch (err: any) {
      setError(err.message || "Something went wrong. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ background: "var(--color-bg)", minHeight: "100vh", overflow: "hidden" }}>
      <Navbar />

      <main style={{ position: "relative", zIndex: 1, maxWidth: 400, margin: "80px auto", padding: "0 20px" }}>
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
          style={{
            background: "var(--color-bg-card)",
            borderRadius: 12,
            padding: "32px",
            boxShadow: "var(--shadow-card), 0 0 0 1px rgba(255,255,255,0.02)",
          }}
        >
          <div style={{ marginBottom: "24px" }}>
            <h1 style={{ fontSize: 24, fontWeight: 700, color: "var(--color-text)", margin: "0 0 8px" }}>
              Forgot Password
            </h1>
            <p style={{ fontSize: 14, color: "var(--color-text-muted)" }}>
              Enter your email address and we'll send you a link to reset your password.
            </p>
          </div>

          {!success ? (
            <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
              <div>
                <label style={{ display: "block", fontSize: 11, fontWeight: 600, color: "var(--color-text-secondary)", marginBottom: 5, letterSpacing: 0.5 }}>EMAIL ADDRESS</label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@example.com"
                  required
                  className="glow-input"
                />
              </div>

              {error && (
                <div style={{ fontSize: 12, color: "var(--color-danger)", background: "var(--color-danger-light)", borderRadius: 6, padding: "8px 12px" }}>
                  {error}
                </div>
              )}

              <AnimatedButton type="submit" variant="primary" fullWidth disabled={loading}>
                {loading ? "Sending..." : "Send Reset Link →"}
              </AnimatedButton>

              <button
                type="button"
                onClick={() => router.push("/")}
                style={{ background: "none", border: "none", color: "var(--color-text-muted)", fontSize: 12, cursor: "pointer", marginTop: "8px" }}
              >
                Back to Login
              </button>
            </form>
          ) : (
            <div style={{ textAlign: "center" }}>
              <div style={{ fontSize: "48px", marginBottom: "16px" }}>📧</div>
              <p style={{ fontSize: 14, color: "var(--color-text)", marginBottom: "24px" }}>
                If an account exists for {email}, you will receive a password reset link shortly.
              </p>
              <AnimatedButton
                onClick={() => router.push("/")}
                variant="outline"
                fullWidth
                className="mt-4"
              >
                Back to Login
              </AnimatedButton>
            </div>
          )}
        </motion.div>
      </main>
    </div>
  );
}
