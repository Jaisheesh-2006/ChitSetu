"use client";

import React, { useState, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence, useInView } from "framer-motion";
import Navbar from "@/components/Navbar";
import AnimatedButton from "@/components/ui/AnimatedButton";
import { useAuth } from "@/hooks/useAuth";
import {
  Shield, Cpu, Zap, Link2, CreditCard, ArrowRight,
  Lock, Landmark, Globe, ChevronDown, Sparkles,
  UserCheck, Building2, TrendingUp, CheckCircle2, Wifi,
} from "lucide-react";

/* ═══════════════════════════════════════════════════════════════
   CSS CREDIT CARD — adapts to light/dark
   ═══════════════════════════════════════════════════════════════ */

function CSSCreditCard({ w = 320, compact = false }: { w?: number; compact?: boolean }) {
  const h = w * 0.63;
  const fs = w / 320; // scale factor

  return (
    <div style={{
      width: w, height: h, borderRadius: 16 * fs,
      position: "relative", overflow: "hidden",
      background: "linear-gradient(135deg, rgba(249,115,22,0.12) 0%, var(--color-bg-card) 40%, rgba(249,115,22,0.06) 100%)",
      border: "1px solid rgba(249,115,22,0.15)",
      boxShadow: "0 20px 60px rgba(0,0,0,0.3), inset 0 1px 0 rgba(255,255,255,0.05)",
      fontFamily: '"Inter", sans-serif',
    }}>
      {/* Top-edge gradient line */}
      <div style={{
        position: "absolute", top: 0, left: 0, right: 0, height: 2,
        background: "linear-gradient(90deg, transparent, rgba(249,115,22,0.5), transparent)",
      }} />

      {/* Mesh gradient overlay */}
      <div style={{
        position: "absolute", inset: 0,
        background: "radial-gradient(ellipse at 80% 20%, rgba(249,115,22,0.08) 0%, transparent 60%)",
        pointerEvents: "none",
      }} />

      {/* Chip */}
      <div style={{
        position: "absolute", top: h * 0.22, left: w * 0.07,
        width: 42 * fs, height: 32 * fs, borderRadius: 6 * fs,
        background: "linear-gradient(145deg, rgba(255,200,100,0.3), rgba(255,180,60,0.15))",
        border: "1px solid rgba(255,180,60,0.2)",
        display: "flex", alignItems: "center", justifyContent: "center",
      }}>
        {/* Chip lines */}
        <div style={{ width: "60%", height: "60%", position: "relative" }}>
          <div style={{ position: "absolute", top: "50%", left: 0, right: 0, height: 1, background: "rgba(255,180,60,0.3)" }} />
          <div style={{ position: "absolute", left: "50%", top: 0, bottom: 0, width: 1, background: "rgba(255,180,60,0.3)" }} />
        </div>
      </div>

      {/* Contactless icon */}
      <div style={{
        position: "absolute", top: h * 0.24, left: w * 0.07 + 50 * fs,
        color: "rgba(255,255,255,0.15)",
      }}>
        <Wifi size={16 * fs} style={{ transform: "rotate(90deg)" }} />
      </div>

      {/* Brand logo circle */}
      <div style={{
        position: "absolute", top: h * 0.1, right: w * 0.06,
        display: "flex", alignItems: "center", gap: 4 * fs,
      }}>
        <div style={{
          width: 28 * fs, height: 28 * fs, borderRadius: "50%",
          background: "linear-gradient(135deg, rgba(249,115,22,0.6), rgba(234,88,12,0.4))",
          border: "1px solid rgba(249,115,22,0.3)",
          display: "flex", alignItems: "center", justifyContent: "center",
        }}>
          <Shield size={14 * fs} style={{ color: "rgba(255,255,255,0.8)" }} />
        </div>
        {!compact && (
          <span style={{
            fontSize: 11 * fs, fontWeight: 700, color: "var(--color-accent)",
            letterSpacing: 0.5,
          }}>
            CS
          </span>
        )}
      </div>

      {/* Card number */}
      <div style={{
        position: "absolute", top: h * 0.52, left: w * 0.07,
        display: "flex", gap: 10 * fs, alignItems: "center",
      }}>
        {["••••", "••••", "••••"].map((g, i) => (
          <span key={i} style={{
            fontSize: 13 * fs, fontWeight: 600, color: "var(--color-text-muted)",
            letterSpacing: 2 * fs, opacity: 0.4,
          }}>
            {g}
          </span>
        ))}
        <span style={{
          fontSize: 15 * fs, fontWeight: 700, color: "var(--color-text)",
          letterSpacing: 2 * fs,
        }}>
          0005
        </span>
      </div>

      {/* Bottom row: Name + Expiry */}
      <div style={{
        position: "absolute", bottom: h * 0.12, left: w * 0.07, right: w * 0.07,
        display: "flex", justifyContent: "space-between", alignItems: "flex-end",
      }}>
        <div>
          <span style={{
            fontSize: 8 * fs, color: "var(--color-text-muted)", letterSpacing: 1,
            textTransform: "uppercase", opacity: 0.5, display: "block", marginBottom: 2,
          }}>
            Card Holder
          </span>
          <span style={{
            fontSize: 12 * fs, fontWeight: 700, color: "var(--color-text)",
            letterSpacing: 1.5, textTransform: "uppercase",
          }}>
            AlgoForge
          </span>
        </div>
        <div style={{ textAlign: "right" }}>
          <span style={{
            fontSize: 8 * fs, color: "var(--color-text-muted)", letterSpacing: 1,
            textTransform: "uppercase", opacity: 0.5, display: "block", marginBottom: 2,
          }}>
            Expires
          </span>
          <span style={{
            fontSize: 12 * fs, fontWeight: 700, color: "var(--color-text)",
            letterSpacing: 1,
          }}>
            06/28
          </span>
        </div>
      </div>

      {/* Bottom right network logo circles */}
      <div style={{
        position: "absolute", bottom: h * 0.1, right: w * 0.06,
        display: "flex",
      }}>
        <div style={{
          width: 22 * fs, height: 22 * fs, borderRadius: "50%",
          background: "rgba(249,115,22,0.35)", marginRight: -8 * fs,
        }} />
        <div style={{
          width: 22 * fs, height: 22 * fs, borderRadius: "50%",
          background: "rgba(234,88,12,0.25)",
        }} />
      </div>

      {/* Subtle noise */}
      <div style={{
        position: "absolute", inset: 0,
        background: "repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(255,255,255,0.005) 2px, rgba(255,255,255,0.005) 4px)",
        pointerEvents: "none",
      }} />
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════
   CSS COIN — pure CSS, adapts to theme
   ═══════════════════════════════════════════════════════════════ */

function CSSCoin({ size = 60 }: { size?: number }) {
  return (
    <div style={{
      width: size, height: size, borderRadius: "50%",
      position: "relative",
      background: "linear-gradient(145deg, rgba(249,115,22,0.2), rgba(249,115,22,0.05))",
      border: "2px solid rgba(249,115,22,0.2)",
      boxShadow: `
        inset 0 2px 6px rgba(249,115,22,0.15),
        inset 0 -2px 6px rgba(0,0,0,0.1),
        0 4px 12px rgba(0,0,0,0.15)
      `,
      display: "flex", alignItems: "center", justifyContent: "center",
    }}>
      {/* Inner ring */}
      <div style={{
        width: size * 0.7, height: size * 0.7, borderRadius: "50%",
        border: "1px solid rgba(249,115,22,0.15)",
        display: "flex", alignItems: "center", justifyContent: "center",
      }}>
        <Shield size={size * 0.3} style={{
          color: "var(--color-accent)",
          opacity: 0.7,
        }} />
      </div>
      {/* Glass highlight */}
      <div style={{
        position: "absolute", top: "8%", left: "15%", right: "15%", height: "30%",
        borderRadius: "50%",
        background: "linear-gradient(180deg, rgba(255,255,255,0.12), transparent)",
        pointerEvents: "none",
      }} />
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════
   GLITCH CREDIT CARD LOADER
   ═══════════════════════════════════════════════════════════════ */

function GlitchLoader({ onFinish }: { onFinish: () => void }) {
  const [progress, setProgress] = useState(0);

  useEffect(() => {
    const start = Date.now();
    const duration = 2800;
    const tick = () => {
      const elapsed = Date.now() - start;
      const p = Math.min(elapsed / duration, 1);
      setProgress(Math.round(p * 100));
      if (p < 1) requestAnimationFrame(tick);
      else setTimeout(onFinish, 300);
    };
    requestAnimationFrame(tick);
  }, [onFinish]);

  return (
    <motion.div
      exit={{ opacity: 0, scale: 1.05 }}
      transition={{ duration: 0.5, ease: "easeInOut" }}
      style={{
        position: "fixed", inset: 0, zIndex: 9999,
        background: "var(--color-bg)",
        display: "flex", flexDirection: "column",
        alignItems: "center", justifyContent: "center",
        overflow: "hidden",
      }}
    >
      {/* Background grid */}
      <div style={{
        position: "absolute", inset: 0, opacity: 0.03,
        backgroundImage: `
          linear-gradient(rgba(249,115,22,0.3) 1px, transparent 1px),
          linear-gradient(90deg, rgba(249,115,22,0.3) 1px, transparent 1px)
        `,
        backgroundSize: "60px 60px",
      }} />

      {/* Floating particles */}
      {Array.from({ length: 12 }).map((_, i) => (
        <div key={i} style={{
          position: "absolute",
          width: 3 + Math.random() * 4, height: 3 + Math.random() * 4,
          borderRadius: "50%",
          background: `rgba(249,115,22,${0.1 + Math.random() * 0.2})`,
          top: `${Math.random() * 100}%`, left: `${Math.random() * 100}%`,
          animation: `float ${4 + Math.random() * 4}s ease-in-out ${Math.random() * 2}s infinite`,
          pointerEvents: "none",
        }} />
      ))}

      {/* Credit Card */}
      <div style={{
        position: "relative", marginBottom: 40,
        animation: "card-rotate-y 3s ease-in-out infinite",
        transformStyle: "preserve-3d",
      }}>
        <div style={{ position: "relative" }}>
          <CSSCreditCard w={340} />

          {/* Scan line */}
          <div style={{
            position: "absolute", left: 0, width: "100%", height: 3,
            background: "linear-gradient(90deg, transparent, rgba(249,115,22,0.4), transparent)",
            animation: "scan-line 1.5s linear infinite", pointerEvents: "none",
          }} />

          {/* Glitch layer */}
          <div style={{
            position: "absolute", inset: 0, borderRadius: 16, overflow: "hidden",
            animation: "glitch-1 3s steps(1) infinite", opacity: 0.06, pointerEvents: "none",
            background: "linear-gradient(135deg, rgba(249,115,22,0.3), transparent, rgba(249,115,22,0.2))",
          }} />
        </div>

        {/* Glow underneath */}
        <div style={{
          position: "absolute", bottom: -20, left: "10%", right: "10%", height: 40,
          background: "radial-gradient(ellipse, rgba(249,115,22,0.2), transparent)",
          filter: "blur(20px)", animation: "pulse-glow 2s ease-in-out infinite",
        }} />
      </div>

      {/* Loading text & progress */}
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.3 }}
        style={{ textAlign: "center", position: "relative", zIndex: 2 }}
      >
        <p style={{
          fontSize: 11, fontWeight: 700, letterSpacing: 3,
          color: "rgba(249,115,22,0.7)", textTransform: "uppercase", marginBottom: 12,
        }}>
          Initializing ChitSetu
        </p>
        <div style={{
          width: 200, height: 2, borderRadius: 1,
          background: "var(--color-bg-subtle)", overflow: "hidden", margin: "0 auto",
        }}>
          <motion.div
            animate={{ width: `${progress}%` }}
            style={{
              height: "100%", borderRadius: 1,
              background: "linear-gradient(90deg, #f97316, #ea580c)",
              boxShadow: "0 0 10px rgba(249,115,22,0.4)",
            }}
          />
        </div>
        <p style={{
          fontSize: 10, color: "var(--color-text-muted)", marginTop: 8, fontFamily: "monospace",
        }}>
          {progress}%
        </p>
      </motion.div>
    </motion.div>
  );
}

/* ═══════════════════════════════════════════════════════════════
   HERO VISUAL — CSS Card + Orbiting CSS Coins
   ═══════════════════════════════════════════════════════════════ */

function HeroVisual() {
  const coinPositions = [
    { size: 44, x: -80, y: -120, delay: 0, dur: 5 },
    { size: 36, x: 170, y: -100, delay: 0.4, dur: 6 },
    { size: 30, x: -90, y: 100, delay: 0.8, dur: 7 },
    { size: 38, x: 180, y: 110, delay: 1.2, dur: 5.5 },
    { size: 24, x: -40, y: -30, delay: 0.6, dur: 6.5 },
    { size: 28, x: 130, y: 10, delay: 1.0, dur: 7.5 },
  ];

  return (
    <div style={{
      position: "relative", width: "100%", height: "100%", minHeight: 440,
      display: "flex", alignItems: "center", justifyContent: "center",
      perspective: "1000px",
    }}>
      {/* Outer orbital ring */}
      <motion.div
        animate={{ rotate: 360 }}
        transition={{ repeat: Infinity, duration: 20, ease: "linear" }}
        style={{
          position: "absolute", width: 360, height: 360, borderRadius: "50%",
          border: "1px solid rgba(249,115,22,0.06)",
        }}
      >
        <div style={{
          position: "absolute", top: -4, left: "50%", transform: "translateX(-50%)",
          width: 8, height: 8, borderRadius: "50%", background: "var(--color-accent)",
          boxShadow: "0 0 12px rgba(249,115,22,0.4)",
        }} />
        <div style={{
          position: "absolute", bottom: -4, left: "50%", transform: "translateX(-50%)",
          width: 5, height: 5, borderRadius: "50%", background: "rgba(249,115,22,0.4)",
          boxShadow: "0 0 8px rgba(249,115,22,0.2)",
        }} />
      </motion.div>

      {/* Inner orbital ring */}
      <motion.div
        animate={{ rotate: -360 }}
        transition={{ repeat: Infinity, duration: 15, ease: "linear" }}
        style={{
          position: "absolute", width: 240, height: 240, borderRadius: "50%",
          border: "1px dashed rgba(249,115,22,0.04)",
        }}
      >
        <div style={{
          position: "absolute", top: "50%", right: -3, transform: "translateY(-50%)",
          width: 6, height: 6, borderRadius: "50%", background: "rgba(96,165,250,0.5)",
          boxShadow: "0 0 8px rgba(96,165,250,0.3)",
        }} />
      </motion.div>

      {/* Central card */}
      <motion.div
        animate={{
          rotateY: [0, 8, -8, 0],
          rotateX: [-3, 3, -3],
        }}
        transition={{ repeat: Infinity, duration: 8, ease: "easeInOut" }}
        style={{
          position: "relative", zIndex: 2,
          transformStyle: "preserve-3d",
        }}
      >
        <CSSCreditCard w={280} />

        {/* Glossy sweep */}
        <motion.div
          animate={{ x: ["-100%", "300%"] }}
          transition={{ repeat: Infinity, duration: 4, ease: "easeInOut", repeatDelay: 3 }}
          style={{
            position: "absolute", top: 0, left: 0, width: "30%", height: "100%",
            borderRadius: 16,
            background: "linear-gradient(90deg, transparent, rgba(255,255,255,0.06), transparent)",
            transform: "skewX(-15deg)", pointerEvents: "none",
          }}
        />
      </motion.div>

      {/* Floating CSS coins around the card */}
      {coinPositions.map((c, i) => (
        <motion.div
          key={i}
          initial={{ opacity: 0, scale: 0.3 }}
          animate={{ opacity: [0.5, 0.8, 0.5], scale: 1, y: [0, -12, 0] }}
          transition={{
            opacity: { repeat: Infinity, duration: c.dur, delay: c.delay },
            scale: { duration: 0.6, delay: 0.8 + c.delay },
            y: { repeat: Infinity, duration: c.dur, delay: c.delay, ease: "easeInOut" },
          }}
          style={{
            position: "absolute",
            top: `calc(50% + ${c.y}px)`,
            left: `calc(50% + ${c.x}px)`,
            zIndex: i % 2 === 0 ? 1 : 3,
            filter: `drop-shadow(0 4px 8px rgba(0,0,0,0.15))`,
          }}
        >
          <CSSCoin size={c.size} />
        </motion.div>
      ))}

      {/* Floating feature badges */}
      {[
        { icon: <Shield size={14} />, label: "Secured", x: -50, y: -140, delay: 0.2 },
        { icon: <Cpu size={14} />, label: "ML Powered", x: 150, y: -110, delay: 0.5 },
        { icon: <Link2 size={14} />, label: "On-chain", x: -70, y: 120, delay: 0.8 },
        { icon: <Zap size={14} />, label: "Real-time", x: 160, y: 100, delay: 1.1 },
      ].map((badge) => (
        <motion.div
          key={badge.label}
          initial={{ opacity: 0, scale: 0.5 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 1 + badge.delay, duration: 0.5, type: "spring" }}
          style={{
            position: "absolute",
            top: `calc(50% + ${badge.y}px)`,
            left: `calc(50% + ${badge.x}px)`,
            zIndex: 4,
          }}
        >
          <motion.div
            animate={{ y: [0, -6, 0] }}
            transition={{ repeat: Infinity, duration: 3 + badge.delay, ease: "easeInOut" }}
            style={{
              display: "flex", alignItems: "center", gap: 6,
              background: "var(--color-bg-card)",
              borderRadius: 8, padding: "6px 12px",
              boxShadow: "0 8px 24px rgba(0,0,0,0.2), 0 0 0 1px rgba(255,255,255,0.04)",
              border: "1px solid rgba(249,115,22,0.08)",
            }}
          >
            <span style={{ color: "var(--color-accent)", display: "flex" }}>{badge.icon}</span>
            <span style={{
              fontSize: 10, fontWeight: 600, color: "var(--color-text-secondary)",
              letterSpacing: 0.3,
            }}>
              {badge.label}
            </span>
          </motion.div>
        </motion.div>
      ))}

      {/* Background glow */}
      <div style={{
        position: "absolute", width: 200, height: 200, borderRadius: "50%",
        background: "radial-gradient(circle, rgba(249,115,22,0.06), transparent)",
        filter: "blur(40px)", pointerEvents: "none",
      }} />
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════
   SCROLL REVEAL
   ═══════════════════════════════════════════════════════════════ */

function ScrollReveal({ children, delay = 0 }: { children: React.ReactNode; delay?: number }) {
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, { once: true, margin: "-80px" });

  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 40 }}
      animate={inView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.7, delay, ease: [0.16, 1, 0.3, 1] }}
    >
      {children}
    </motion.div>
  );
}

/* ═══════════════════════════════════════════════════════════════
   FEATURE CARD
   ═══════════════════════════════════════════════════════════════ */

function FeatureCard({ icon, title, desc, i }: { icon: React.ReactNode; title: string; desc: string; i: number }) {
  const ref = useRef<HTMLDivElement>(null);
  const [rot, setRot] = useState({ x: 0, y: 0 });
  const [h, setH] = useState(false);

  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 30 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.6, delay: i * 0.1, ease: [0.16, 1, 0.3, 1] }}
      onMouseMove={(e) => {
        if (!ref.current) return;
        const r = ref.current.getBoundingClientRect();
        setRot({
          x: -((e.clientY - (r.top + r.height / 2)) / (r.height / 2)) * 6,
          y: ((e.clientX - (r.left + r.width / 2)) / (r.width / 2)) * 6,
        });
      }}
      onMouseEnter={() => setH(true)}
      onMouseLeave={() => { setRot({ x: 0, y: 0 }); setH(false); }}
      style={{
        position: "relative", overflow: "hidden",
        background: "var(--color-bg-card)", borderRadius: 14, padding: "28px 24px",
        boxShadow: h ? "var(--shadow-hover), var(--shadow-glow-sm)" : "var(--shadow-card)",
        transform: h
          ? `perspective(600px) rotateX(${rot.x}deg) rotateY(${rot.y}deg) translateY(-6px)`
          : "perspective(600px) rotateX(0) rotateY(0) translateY(0)",
        transition: "box-shadow 0.35s, transform 0.3s",
        transformStyle: "preserve-3d", cursor: "default",
      }}
    >

      <div style={{
        position: "absolute", top: 0, left: 0, right: 0, height: h ? 2 : 1,
        background: h ? "var(--gradient-primary)" : "linear-gradient(90deg, transparent, rgba(255,255,255,0.02), transparent)",
        transition: "all 0.3s", pointerEvents: "none",
      }} />
      <div style={{ position: "relative", zIndex: 2 }}>
        <motion.div
          animate={h ? { scale: 1.15, rotate: 5 } : { scale: 1, rotate: 0 }}
          transition={{ type: "spring", stiffness: 400 }}
          style={{
            display: "inline-flex", alignItems: "center", justifyContent: "center",
            width: 40, height: 40, borderRadius: 10,
            background: "rgba(249,115,22,0.08)", color: "var(--color-accent)",
            marginBottom: 14,
          }}
        >
          {icon}
        </motion.div>
        <p style={{ fontSize: 16, fontWeight: 700, color: "var(--color-text)", margin: "0 0 6px" }}>{title}</p>
        <p style={{ fontSize: 13, color: "var(--color-text-muted)", margin: 0, lineHeight: 1.7 }}>{desc}</p>
      </div>
    </motion.div>
  );
}

/* ═══════════════════════════════════════════════════════════════
   MAIN LANDING PAGE
   ═══════════════════════════════════════════════════════════════ */

export default function LandingPage() {
  const [loaderDone, setLoaderDone] = useState(false);
  const { isAuthenticated } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (isAuthenticated) router.push("/dashboard");
  }, [isAuthenticated, router]);

  const features = [
    { icon: <Cpu size={20} />, title: "ML Risk Scoring", desc: "AI-powered trust assessment using credit history, bank data, and ML models to calculate default probability." },
    { icon: <Zap size={20} />, title: "Real-time Auctions", desc: "Live reverse auctions with 20-second countdown bidding. WebSocket-powered instant updates for all participants." },
    { icon: <CreditCard size={20} />, title: "Secure Payments", desc: "Razorpay-integrated UPI & card payments. Every transaction recorded and verified on-chain." },
    { icon: <Link2 size={20} />, title: "Blockchain Records", desc: "All contributions and payouts are recorded on zkSync blockchain. Immutable, transparent, and verifiable." },
  ];

  const howItWorks = [
    { step: "01", title: "Create Profile & KYC", desc: "Sign up, complete your profile, and verify your identity through our automated PAN & credit verification.", icon: <UserCheck size={22} /> },
    { step: "02", title: "Join or Create Funds", desc: "Browse available chit funds or create your own. Set pool size, monthly contribution, and member limits.", icon: <Building2 size={22} /> },
    { step: "03", title: "Participate & Earn", desc: "Pay monthly contributions, bid in reverse auctions, and earn dividends. Track everything on your dashboard.", icon: <TrendingUp size={22} /> },
  ];

  return (
    <div style={{ background: "var(--color-bg)", minHeight: "100vh", overflow: "hidden" }}>
      <AnimatePresence>
        {!loaderDone && <GlitchLoader onFinish={() => setLoaderDone(true)} />}
      </AnimatePresence>

      {loaderDone && (
        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.6 }}>
          <Navbar />

          {/* Ambient Background */}
          <div style={{ position: "fixed", inset: 0, pointerEvents: "none", zIndex: 0 }}>
            <motion.div
              animate={{ x: [0, 30, 0], y: [0, -20, 0] }}
              transition={{ repeat: Infinity, duration: 12, ease: "easeInOut" }}
              style={{
                position: "absolute", top: "10%", left: "15%", width: 400, height: 400, borderRadius: "50%",
                background: "radial-gradient(circle, rgba(249,115,22,0.04), transparent)", filter: "blur(60px)",
              }}
            />
            <motion.div
              animate={{ x: [0, -20, 0], y: [0, 25, 0] }}
              transition={{ repeat: Infinity, duration: 15, ease: "easeInOut" }}
              style={{
                position: "absolute", bottom: "20%", right: "10%", width: 300, height: 300, borderRadius: "50%",
                background: "radial-gradient(circle, rgba(96,165,250,0.03), transparent)", filter: "blur(50px)",
              }}
            />
          </div>

          {/* ── HERO ── */}
          <section style={{
            position: "relative", zIndex: 1, maxWidth: 1200, margin: "0 auto",
            padding: "60px 24px 40px",
            display: "flex", flexWrap: "wrap", gap: 40,
            alignItems: "center", justifyContent: "center",
            minHeight: "calc(100vh - 52px)",
          }}>
            <div style={{ flex: "1 1 480px", maxWidth: 540 }}>
              <motion.div
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.5, delay: 0.1 }}
                style={{
                  display: "inline-flex", alignItems: "center", gap: 8,
                  background: "var(--color-bg-card)", borderRadius: 8,
                  padding: "6px 14px", marginBottom: 24, boxShadow: "var(--shadow-card)",
                }}
              >
                <motion.span
                  animate={{ scale: [1, 1.4, 1] }}
                  transition={{ repeat: Infinity, duration: 2 }}
                  style={{
                    width: 6, height: 6, borderRadius: "50%",
                    background: "var(--color-accent)",
                    boxShadow: "0 0 8px rgba(249,115,22,0.4)", display: "inline-block",
                  }}
                />
                <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-accent)", letterSpacing: 1.5, textTransform: "uppercase" }}>
                  Live Platform
                </span>
              </motion.div>

              <motion.h1
                initial={{ opacity: 0, y: 24, filter: "blur(8px)" }}
                animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
                transition={{ duration: 0.8, delay: 0.2, ease: [0.16, 1, 0.3, 1] }}
                style={{ fontSize: 52, fontWeight: 800, color: "var(--color-text)", lineHeight: 1.08, letterSpacing: "-0.03em", margin: "0 0 8px" }}
              >
                Chit funds,
              </motion.h1>
              <motion.h1
                initial={{ opacity: 0, y: 24, filter: "blur(8px)" }}
                animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
                transition={{ duration: 0.8, delay: 0.35, ease: [0.16, 1, 0.3, 1] }}
                style={{ fontSize: 52, fontWeight: 800, lineHeight: 1.08, letterSpacing: "-0.03em", margin: "0 0 20px" }}
              >
                <span className="gradient-text">reimagined.</span>
              </motion.h1>

              <motion.p
                initial={{ opacity: 0, y: 12 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.5 }}
                style={{ fontSize: 16, color: "var(--color-text-secondary)", lineHeight: 1.8, maxWidth: 440, margin: "0 0 32px" }}
              >
                A transparent, digital chit fund platform powered by machine learning risk scoring,
                real-time blockchain auctions, and bank-grade secure payments.
              </motion.p>

              <motion.div
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.6, duration: 0.5 }}
                style={{ display: "flex", gap: 12, marginBottom: 28, flexWrap: "wrap" }}
              >
                <AnimatedButton variant="primary" size="lg" onClick={() => router.push("/login")}>
                  <span style={{ display: "flex", alignItems: "center", gap: 8 }}>
                    Get Started <ArrowRight size={16} />
                  </span>
                </AnimatedButton>
                <AnimatedButton variant="ghost" size="lg" onClick={() => { document.getElementById("features")?.scrollIntoView({ behavior: "smooth" }); }}>
                  Learn More
                </AnimatedButton>
              </motion.div>

              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ delay: 0.8 }}
                style={{ display: "flex", gap: 8, flexWrap: "wrap" }}
              >
                {[
                  { icon: <Lock size={10} />, text: "256-bit SSL" },
                  { icon: <Landmark size={10} />, text: "RBI Compliant" },
                  { icon: <Globe size={10} />, text: "GDPR Ready" },
                  { icon: <Link2 size={10} />, text: "zkSync Secured" },
                ].map((t) => (
                  <span key={t.text} style={{
                    display: "inline-flex", alignItems: "center", gap: 5,
                    fontSize: 10, color: "var(--color-text-muted)", fontWeight: 500,
                    background: "var(--color-bg-card)", borderRadius: 6, padding: "5px 10px",
                    boxShadow: "var(--shadow-card)",
                  }}>
                    <span style={{ color: "var(--color-accent)", display: "flex" }}>{t.icon}</span>
                    {t.text}
                  </span>
                ))}
              </motion.div>
            </div>

            <motion.div
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ duration: 0.8, delay: 0.4 }}
              style={{ flex: "1 1 400px", maxWidth: 500, minHeight: 440 }}
            >
              <HeroVisual />
            </motion.div>

            <motion.div
              animate={{ y: [0, 8, 0], opacity: [0.3, 0.7, 0.3] }}
              transition={{ repeat: Infinity, duration: 2, ease: "easeInOut" }}
              style={{
                position: "absolute", bottom: 24, left: "50%", transform: "translateX(-50%)",
                display: "flex", flexDirection: "column", alignItems: "center", gap: 4,
              }}
            >
              <span style={{ fontSize: 9, color: "var(--color-text-muted)", fontWeight: 500, letterSpacing: 1, textTransform: "uppercase" }}>Scroll</span>
              <ChevronDown size={16} style={{ color: "var(--color-text-muted)" }} />
            </motion.div>
          </section>

          {/* ── FEATURES ── */}
          <section id="features" style={{ position: "relative", zIndex: 1, maxWidth: 1100, margin: "0 auto", padding: "60px 24px 80px" }}>
            <ScrollReveal>
              <div style={{ textAlign: "center", marginBottom: 48 }}>
                <span style={{
                  display: "inline-flex", alignItems: "center", gap: 6,
                  fontSize: 10, fontWeight: 700, color: "var(--color-accent)",
                  letterSpacing: 2, textTransform: "uppercase",
                  background: "rgba(249,115,22,0.08)", borderRadius: 6, padding: "5px 14px",
                }}>
                  <Sparkles size={12} /> Features
                </span>
                <h2 style={{ fontSize: 36, fontWeight: 800, color: "var(--color-text)", letterSpacing: "-0.02em", margin: "16px 0 8px" }}>
                  Everything you need to <span className="gradient-text">manage chit funds</span>
                </h2>
                <p style={{ fontSize: 14, color: "var(--color-text-muted)", maxWidth: 500, margin: "0 auto" }}>
                  Built with cutting-edge technology for maximum transparency and security.
                </p>
              </div>
            </ScrollReveal>
            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(240px, 1fr))", gap: 14 }}>
              {features.map((f, i) => (
                <FeatureCard key={f.title} icon={f.icon} title={f.title} desc={f.desc} i={i} />
              ))}
            </div>
          </section>

          {/* ── HOW IT WORKS ── */}
          <section style={{ position: "relative", zIndex: 1, maxWidth: 1100, margin: "0 auto", padding: "40px 24px 80px" }}>
            <ScrollReveal>
              <div style={{ textAlign: "center", marginBottom: 48 }}>
                <span style={{
                  display: "inline-flex", alignItems: "center", gap: 6,
                  fontSize: 10, fontWeight: 700, color: "var(--color-accent)",
                  letterSpacing: 2, textTransform: "uppercase",
                  background: "rgba(249,115,22,0.08)", borderRadius: 6, padding: "5px 14px",
                }}>
                  <CheckCircle2 size={12} /> How It Works
                </span>
                <h2 style={{ fontSize: 36, fontWeight: 800, color: "var(--color-text)", letterSpacing: "-0.02em", margin: "16px 0 8px" }}>
                  Get started in <span className="gradient-text">3 simple steps</span>
                </h2>
              </div>
            </ScrollReveal>
            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))", gap: 16 }}>
              {howItWorks.map((step, i) => (
                <ScrollReveal key={step.step} delay={i * 0.12}>
                  <div style={{
                    position: "relative", background: "var(--color-bg-card)",
                    borderRadius: 14, padding: "28px 24px",
                    boxShadow: "var(--shadow-card)", overflow: "hidden",
                  }}>
                    <span style={{
                      position: "absolute", top: -8, right: 12,
                      fontSize: 80, fontWeight: 900, color: "rgba(255,255,255,0.02)",
                      lineHeight: 1, pointerEvents: "none",
                    }}>{step.step}</span>
                    <div style={{
                      position: "absolute", top: 0, left: 0, right: 0, height: 2,
                      background: i === 0 ? "linear-gradient(90deg, #f97316, transparent)" : i === 1 ? "linear-gradient(90deg, transparent, #f97316, transparent)" : "linear-gradient(90deg, transparent, #f97316)",
                    }} />
                    <div style={{ position: "relative", zIndex: 2 }}>
                      <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 14 }}>
                        <div style={{
                          width: 40, height: 40, borderRadius: 10,
                          background: "rgba(249,115,22,0.08)",
                          display: "flex", alignItems: "center", justifyContent: "center",
                          color: "var(--color-accent)",
                        }}>{step.icon}</div>
                        <span style={{ fontSize: 11, fontWeight: 700, color: "var(--color-accent)", letterSpacing: 1, textTransform: "uppercase" }}>
                          Step {step.step}
                        </span>
                      </div>
                      <p style={{ fontSize: 17, fontWeight: 700, color: "var(--color-text)", margin: "0 0 6px" }}>{step.title}</p>
                      <p style={{ fontSize: 13, color: "var(--color-text-muted)", margin: 0, lineHeight: 1.7 }}>{step.desc}</p>
                    </div>
                    {i < howItWorks.length - 1 && (
                      <div style={{ position: "absolute", top: "50%", right: -8, width: 16, height: 2, background: "rgba(249,115,22,0.15)" }} />
                    )}
                  </div>
                </ScrollReveal>
              ))}
            </div>
          </section>

          {/* ── CTA ── */}
          <section style={{ position: "relative", zIndex: 1, maxWidth: 800, margin: "0 auto", padding: "40px 24px 100px" }}>
            <ScrollReveal>
              <div style={{
                position: "relative", overflow: "hidden",
                background: "var(--color-bg-card)", borderRadius: 20,
                padding: "60px 40px", textAlign: "center",
                boxShadow: "0 20px 60px rgba(0,0,0,0.4), 0 0 0 1px rgba(255,255,255,0.02)",
              }}>
                <div style={{
                  position: "absolute", top: -60, left: "50%", transform: "translateX(-50%)",
                  width: 400, height: 400, borderRadius: "50%",
                  background: "radial-gradient(circle, rgba(249,115,22,0.06), transparent)", pointerEvents: "none",
                }} />
                <div style={{ position: "absolute", top: 0, left: 0, right: 0, height: 2, background: "var(--gradient-primary)" }} />
                <div style={{ position: "relative", zIndex: 2 }}>
                  <motion.div
                    animate={{ y: [0, -6, 0] }}
                    transition={{ repeat: Infinity, duration: 3, ease: "easeInOut" }}
                    style={{
                      display: "inline-flex", alignItems: "center", justifyContent: "center",
                      width: 56, height: 56, borderRadius: 14, background: "rgba(249,115,22,0.08)",
                      color: "var(--color-accent)", marginBottom: 16,
                    }}
                  >
                    <Sparkles size={28} />
                  </motion.div>
                  <h2 style={{ fontSize: 32, fontWeight: 800, color: "var(--color-text)", letterSpacing: "-0.02em", margin: "0 0 10px" }}>
                    Ready to get started?
                  </h2>
                  <p style={{ fontSize: 15, color: "var(--color-text-muted)", maxWidth: 420, margin: "0 auto 28px", lineHeight: 1.7 }}>
                    Join thousands of members who are already using ChitSetu to manage their chit funds transparently and securely.
                  </p>
                  <div style={{ display: "flex", gap: 12, justifyContent: "center", flexWrap: "wrap" }}>
                    <AnimatedButton variant="primary" size="lg" onClick={() => router.push("/login")}>
                      <span style={{ display: "flex", alignItems: "center", gap: 8 }}>Get Started <ArrowRight size={16} /></span>
                    </AnimatedButton>
                    <AnimatedButton variant="ghost" size="lg" onClick={() => { document.getElementById("features")?.scrollIntoView({ behavior: "smooth" }); }}>
                      Learn More
                    </AnimatedButton>
                  </div>
                </div>
              </div>
            </ScrollReveal>
          </section>

          {/* ── FOOTER ── */}
          <footer style={{ position: "relative", zIndex: 1, borderTop: "1px solid rgba(255,255,255,0.04)", padding: "24px", textAlign: "center" }}>
            <p style={{ fontSize: 11, color: "var(--color-text-muted)" }}>© 2026 ChitSetu. Built for transparent chit fund management.</p>
          </footer>
        </motion.div>
      )}
    </div>
  );
}
