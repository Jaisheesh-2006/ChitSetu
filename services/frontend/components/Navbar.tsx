"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { useThemeMode } from "@/components/ThemeProvider";
import { useAuth } from "@/hooks/useAuth";
import { getWalletInfo } from "@/services/api";
import GoogleTranslate from "@/components/GoogleTranslate";
import {
  Sun,
  Moon,
  Menu,
  X,
  LayoutDashboard,
  Layers,
} from "lucide-react";

export default function Navbar() {
  const { mode, toggleMode } = useThemeMode();
  const { isAuthenticated, logout } = useAuth();
  const pathname = usePathname();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [walletBalance, setWalletBalance] = useState<number | null>(null);

  useEffect(() => {
    if (!isAuthenticated) {
      return;
    }

    let disposed = false;
    const loadWalletBalance = async () => {
      try {
        const wallet = await getWalletInfo();
        if (!disposed) {
          setWalletBalance(wallet.balance);
        }
      } catch {
        if (!disposed) {
          setWalletBalance(null);
        }
      }
    };

    void loadWalletBalance();
    const interval = setInterval(() => {
      void loadWalletBalance();
    }, 15000);

    return () => {
      disposed = true;
      clearInterval(interval);
    };
  }, [isAuthenticated]);

  const formattedWalletBalance = walletBalance === null
    ? "--"
    : walletBalance.toLocaleString("en-IN", { maximumFractionDigits: 2 });

  useEffect(() => {
    const check = () => setIsMobile(window.innerWidth < 768);
    check();
    window.addEventListener("resize", check);
    return () => window.removeEventListener("resize", check);
  }, []);

  const links = [
    { label: "Dashboard", href: "/dashboard", icon: <LayoutDashboard size={14} /> },
    { label: "Funds", href: "/funds", icon: <Layers size={14} /> },
  ];

  return (
    <nav style={{
      position: "sticky", top: 0, zIndex: 50,
      background: "var(--color-bg)",
      boxShadow: "0 1px 4px rgba(0,0,0,0.3)",
    }}>
      <div style={{
        display: "flex", alignItems: "center", justifyContent: "space-between",
        height: 52, maxWidth: 1200, margin: "0 auto", padding: "0 20px",
      }}>
        {/* Logo */}
        <Link href="/" style={{ display: "flex", alignItems: "center", gap: 10, textDecoration: "none" }}>
          <motion.div
            whileHover={{ rotateY: 15, scale: 1.05 }}
            transition={{ type: "spring", stiffness: 300, damping: 20 }}
            style={{
              width: 30, height: 30, borderRadius: 6,
              background: "var(--gradient-primary)",
              display: "flex", alignItems: "center", justifyContent: "center",
              boxShadow: "0 2px 8px rgba(249,115,22,0.3)",
              transformStyle: "preserve-3d",
            }}
          >
            <span style={{ color: "#fff", fontWeight: 800, fontSize: 12 }}>CS</span>
          </motion.div>
          <span style={{ fontSize: 17, fontWeight: 700, color: "var(--color-text)", letterSpacing: "-0.01em" }}>
            Chit<span className="gradient-text">Setu</span>
          </span>
        </Link>

        {/* Desktop Nav */}
        {!isMobile && (
          <div style={{ display: "flex", alignItems: "center", gap: 4 }}>
            {isAuthenticated && links.map((l) => {
              const active = pathname.startsWith(l.href);
              return (
                <Link key={l.href} href={l.href} style={{
                  position: "relative",
                  fontSize: 13, fontWeight: 500,
                  color: active ? "var(--color-text)" : "var(--color-text-muted)",
                  padding: "8px 14px", borderRadius: 6,
                  textDecoration: "none",
                  background: active ? "var(--color-bg-subtle)" : "transparent",
                  transition: "all 0.2s",
                }}>
                  {active && <motion.div layoutId="nav-indicator" style={{ position: "absolute", bottom: -1, left: 14, right: 14, height: 2, background: "var(--color-accent)", borderRadius: 1 }} transition={{ type: "spring", stiffness: 400, damping: 30 }} />}
                  <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                    {l.icon}
                    {l.label}
                  </span>
                </Link>
              );
            })}

            {/* Wallet balance display - disabled for now */}
            {isAuthenticated && (
              <div
                style={{
                  marginLeft: 8,
                  marginRight: 4,
                  padding: "6px 10px",
                  borderRadius: 6,
                  background: "rgba(34,197,94,0.08)",
                  border: "1px solid rgba(34,197,94,0.2)",
                  color: "#22c55e",
                  fontSize: 11,
                  fontWeight: 700,
                  letterSpacing: 0.3,
                  whiteSpace: "nowrap",
                }}
              >
                {formattedWalletBalance} CHIT
              </div>
            )}

            <div style={{ width: 1, height: 20, background: "rgba(255,255,255,0.04)", margin: "0 8px" }} />

            <motion.button
              onClick={toggleMode}
              whileHover={{ scale: 1.08, rotateZ: mode === "dark" ? 15 : -15 }}
              whileTap={{ scale: 0.92 }}
              style={{
                width: 32, height: 32, borderRadius: 6,
                border: "none",
                background: "var(--color-bg-subtle)",
                boxShadow: "var(--shadow-card)",
                color: mode === "dark" ? "#f59e0b" : "#6366f1",
                display: "flex", alignItems: "center", justifyContent: "center",
                cursor: "pointer", fontSize: 15, transition: "all 0.2s",
              }}
            >
              {mode === "dark" ? <Sun size={18} /> : <Moon size={18} />}
            </motion.button>

            {isAuthenticated && (
              <motion.button
                onClick={logout}
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
                style={{
                  marginLeft: 4, padding: "6px 14px", borderRadius: 6,
                  border: "none",
                  background: "var(--color-bg-subtle)",
                  boxShadow: "var(--shadow-card)",
                  color: "var(--color-text-muted)",
                  fontSize: 12, fontWeight: 500, cursor: "pointer",
                  fontFamily: '"Inter",sans-serif',
                  transition: "all 0.2s",
                }}
              >
                Log out
              </motion.button>
            )}

            <div style={{ width: 1, height: 20, background: "rgba(255,255,255,0.04)", margin: "0 4px" }} />
            
            <GoogleTranslate />
          </div>
        )}

        {/* Mobile Toggle */}
        {isMobile && (
          <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <motion.button
              onClick={toggleMode}
              whileTap={{ scale: 0.9 }}
              style={{
                width: 32, height: 32, borderRadius: 6,
                border: "none",
                background: "var(--color-bg-subtle)",
                boxShadow: "var(--shadow-card)",
                color: mode === "dark" ? "#f59e0b" : "#6366f1",
                display: "flex", alignItems: "center", justifyContent: "center",
                cursor: "pointer", fontSize: 15,
              }}
            >
              {mode === "dark" ? <Sun size={18} /> : <Moon size={18} />}
            </motion.button>
            <motion.button
              onClick={() => setMobileOpen(!mobileOpen)}
              whileTap={{ scale: 0.9 }}
              style={{
                width: 32, height: 32, borderRadius: 6,
                border: "none",
                background: "var(--color-bg-subtle)",
                boxShadow: "var(--shadow-card)",
                color: "var(--color-text-secondary)",
                display: "flex", alignItems: "center", justifyContent: "center",
                cursor: "pointer", fontSize: 16,
              }}
            >
              {mobileOpen ? <X size={18} /> : <Menu size={18} />}
            </motion.button>
          </div>
        )}
      </div>

      {/* Mobile Dropdown */}
      <AnimatePresence>
        {isMobile && mobileOpen && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            style={{
              background: "var(--color-bg-card)",
              borderTop: "1px solid var(--color-border)",
              padding: "8px 20px 12px",
              overflow: "hidden",
            }}
          >
            {isAuthenticated && links.map((l) => (
              <Link key={l.href} href={l.href} onClick={() => setMobileOpen(false)}
                style={{ display: "block", fontSize: 13, fontWeight: 500, color: pathname.startsWith(l.href) ? "var(--color-accent)" : "var(--color-text-secondary)", padding: "10px 0", textDecoration: "none", borderBottom: "1px solid var(--color-border)" }}>
                <span style={{ display: "flex", alignItems: "center", gap: 8 }}>
                  {l.icon}
                  {l.label}
                </span>
              </Link>
            ))}
            
            {/* Wallet balance display - disabled for now */}
            {isAuthenticated && (
              <div
                style={{
                  marginTop: 8,
                  marginBottom: 8,
                  fontSize: 11,
                  fontWeight: 700,
                  color: "#22c55e",
                  background: "rgba(34,197,94,0.08)",
                  border: "1px solid rgba(34,197,94,0.2)",
                  borderRadius: 6,
                  padding: "6px 10px",
                }}
              >
                Wallet Balance: {formattedWalletBalance} CHIT
              </div>
            )}
            
            {isAuthenticated && (
              <button onClick={() => { logout(); setMobileOpen(false); }}
                style={{ display: "block", marginTop: 6, fontSize: 12, color: "var(--color-text-muted)", background: "none", border: "none", cursor: "pointer", padding: "8px 0", fontFamily: '"Inter",sans-serif' }}>
                Log out
              </button>
            )}

            <div style={{ marginTop: 16 }}>
              <GoogleTranslate />
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </nav>
  );
}
