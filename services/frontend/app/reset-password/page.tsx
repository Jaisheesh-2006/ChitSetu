"use client";

import React, { useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { motion } from "framer-motion";
import Navbar from "@/components/Navbar";
import AnimatedButton from "@/components/ui/AnimatedButton";
import { resetPassword } from "@/services/api";

function ResetPasswordForm() {
  const searchParams = useSearchParams();
  const token = searchParams.get("token");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);
  const router = useRouter();

  if (!token) {
    return (
      <div style={{ textAlign: "center" }}>
        <p style={{ color: "var(--color-danger)", marginBottom: "20px" }}>Invalid or missing reset token.</p>
        <AnimatedButton
          onClick={() => router.push("/")}
          variant="outline"
          fullWidth
          className="mt-4"
        >
          Go Back to Login
        </AnimatedButton>
      </div>
    );
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (password !== confirm) {
      setError("Passwords do not match.");
      return;
    }

    if (password.length < 8) {
      setError("Min 8 characters.");
      return;
    }

    setLoading(true);
    try {
      await resetPassword(token, password);
      setSuccess(true);
    } catch (err: any) {
      setError(err.message || "Failed to reset password.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      {!success ? (
        <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
          <div>
            <label style={{ display: "block", fontSize: 11, fontWeight: 600, color: "var(--color-text-secondary)", marginBottom: 5, letterSpacing: 0.5 }}>NEW PASSWORD</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
              className="glow-input"
            />
          </div>

          <div>
            <label style={{ display: "block", fontSize: 11, fontWeight: 600, color: "var(--color-text-secondary)", marginBottom: 5, letterSpacing: 0.5 }}>CONFIRM PASSWORD</label>
            <input
              type="password"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              placeholder="••••••••"
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
            {loading ? "Resetting..." : "Update Password →"}
          </AnimatedButton>
        </form>
      ) : (
        <div style={{ textAlign: "center" }}>
          <div style={{ fontSize: "48px", marginBottom: "16px" }}>✅</div>
          <p style={{ fontSize: 14, color: "var(--color-text)", marginBottom: "24px" }}>
            Your password has been successfully reset. Redirecting to login...
          </p>
          <AnimatedButton onClick={() => router.push("/")} variant="primary" fullWidth>
            Go to Login
          </AnimatedButton>
        </div>
      )}
    </div>
  );
}

export default function ResetPasswordPage() {
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
              Reset Password
            </h1>
            <p style={{ fontSize: 14, color: "var(--color-text-muted)" }}>
              Set a strong new password for your account.
            </p>
          </div>

          <Suspense fallback={<div>Loading...</div>}>
            <ResetPasswordForm />
          </Suspense>
        </motion.div>
      </main>
    </div>
  );
}
