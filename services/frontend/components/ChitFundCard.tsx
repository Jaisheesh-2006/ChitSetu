"use client";
import React, { useRef, useState } from "react";
import { motion } from "framer-motion";

interface Props {
  id: string; name: string; totalPool: number; totalMembers: number;
  monthlyContribution: number; status: string; minRiskScore: number;
  onClick?: (e?: any) => void | Promise<void>;
}

function badge(s: string) {
  switch (s) {
    case "open": return { bg: "rgba(249,115,22,0.08)", text: "#f97316", dot: "#f97316" };
    case "active": return { bg: "rgba(34,197,94,0.08)", text: "#22c55e", dot: "#22c55e" };
    case "full": return { bg: "rgba(239,68,68,0.08)", text: "#ef4444", dot: "#ef4444" };
    case "closed": case "completed": return { bg: "rgba(107,114,128,0.08)", text: "#6b7280", dot: "#6b7280" };
    default: return { bg: "var(--color-bg-subtle)", text: "var(--color-text-muted)", dot: "var(--color-text-muted)" };
  }
}
function fmt(n: number) { return new Intl.NumberFormat("en-IN", { style: "currency", currency: "INR", maximumFractionDigits: 0 }).format(n); }

export default function ChitFundCard({ name, totalPool, totalMembers, monthlyContribution, status, minRiskScore, onClick }: Props) {
  const b = badge(status);
  const ref = useRef<HTMLDivElement>(null);
  const [hovering, setHovering] = useState(false);
  const [rot, setRot] = useState({ x: 0, y: 0 });

  const onMove = (e: React.MouseEvent) => {
    if (!ref.current) return;
    const r = ref.current.getBoundingClientRect();
    setRot({ x: -((e.clientY - (r.top + r.height / 2)) / (r.height / 2)) * 3, y: ((e.clientX - (r.left + r.width / 2)) / (r.width / 2)) * 3 });
  };

  return (
    <motion.div ref={ref} whileTap={{ scale: 0.98 }}
      onMouseMove={onMove} onMouseEnter={() => setHovering(true)} onMouseLeave={() => { setRot({ x: 0, y: 0 }); setHovering(false); }}
      onClick={onClick}
      style={{
        position: "relative", overflow: "hidden",
        background: "var(--color-bg-card)", border: "none", borderRadius: 10, padding: "16px 18px",
        cursor: onClick ? "pointer" : "default",
        boxShadow: hovering ? "var(--shadow-hover), var(--shadow-glow-sm)" : "var(--shadow-card)",
        transform: hovering ? `perspective(700px) rotateX(${rot.x}deg) rotateY(${rot.y}deg) translateY(-3px)` : "perspective(700px) rotateX(0) rotateY(0) translateY(0)",
        transition: "box-shadow 0.35s, transform 0.3s", transformStyle: "preserve-3d",
      }}
    >
      {/* Glossy shine sweep */}
      <div style={{
        position: "absolute", top: 0, left: 0, width: "40%", height: "100%",
        background: "linear-gradient(90deg, transparent, rgba(255,255,255,0.04), transparent)",
        transform: hovering ? "translateX(350%) skewX(-15deg)" : "translateX(-100%) skewX(-15deg)",
        transition: "transform 0.7s ease", pointerEvents: "none", zIndex: 1,
      }} />

      {/* Top edge glow */}
      <div style={{
        position: "absolute", top: 0, left: 0, right: 0, height: hovering ? 2 : 1,
        background: hovering ? "var(--gradient-primary)" : "linear-gradient(90deg, transparent, rgba(255,255,255,0.02), transparent)",
        transition: "all 0.3s", pointerEvents: "none",
      }} />

      <div style={{ position: "relative", zIndex: 2 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 12 }}>
          <p style={{ fontSize: 14, fontWeight: 600, color: "var(--color-text)", margin: 0, lineHeight: 1.3 }}>{name}</p>
          <span style={{ display: "inline-flex", alignItems: "center", gap: 4, fontSize: 10, fontWeight: 600, textTransform: "capitalize", color: b.text, background: b.bg, borderRadius: 4, padding: "2px 8px", flexShrink: 0, marginLeft: 8 }}>
            <span style={{ width: 4, height: 4, borderRadius: "50%", background: b.dot }} />{status}
          </span>
        </div>
        <p style={{ fontSize: 22, fontWeight: 800, margin: "0 0 14px", letterSpacing: -0.5, background: "var(--gradient-primary)", WebkitBackgroundClip: "text", WebkitTextFillColor: "transparent", backgroundClip: "text" }}>{fmt(totalPool)}</p>
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 0, borderTop: "1px solid rgba(255,255,255,0.04)", paddingTop: 10 }}>
          {[{ l: "Members", v: `${totalMembers}` }, { l: "Monthly", v: fmt(monthlyContribution) }, { l: "Min Score", v: `${minRiskScore}` }].map(({ l, v }, i) => (
            <div key={l} style={{ padding: i === 1 ? "0 8px" : undefined, borderLeft: i > 0 ? "1px solid rgba(255,255,255,0.04)" : undefined }}>
              <p style={{ fontSize: 9, color: "var(--color-text-muted)", margin: 0, textTransform: "uppercase", letterSpacing: 0.5, fontWeight: 600 }}>{l}</p>
              <p style={{ fontSize: 13, fontWeight: 700, color: "var(--color-text)", margin: "3px 0 0" }}>{v}</p>
            </div>
          ))}
        </div>
      </div>
    </motion.div>
  );
}
