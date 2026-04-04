"use client";
import React, { useRef, useState } from "react";
import { motion } from "framer-motion";

interface Props {
  label: string; value: number | string; suffix?: string; icon?: React.ReactNode;
  delay?: number; glow?: boolean; gradient?: boolean; animate?: boolean; accent?: string;
}

export default function StatCard({ label, value, suffix, icon, delay = 0, animate = true, accent }: Props) {
  const isNum = typeof value === "number";
  const ref = useRef<HTMLDivElement>(null);
  const [hovering, setHovering] = useState(false);
  const [rot, setRot] = useState({ x: 0, y: 0 });
  const displayValue = isNum ? Math.round(value) : value;
  const valueTransition = animate ? "transform 0.25s ease" : "none";

  const onMove = (e: React.MouseEvent) => {
    if (!ref.current) return;
    const r = ref.current.getBoundingClientRect();
    setRot({ x: -((e.clientY - (r.top + r.height / 2)) / (r.height / 2)) * 4, y: ((e.clientX - (r.left + r.width / 2)) / (r.width / 2)) * 4 });
  };

  const accentColor = accent || "var(--color-accent)";

  return (
    <motion.div ref={ref} initial={{ opacity: 0, y: 14 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4, delay }}
      onMouseMove={onMove} onMouseEnter={() => setHovering(true)} onMouseLeave={() => { setRot({ x: 0, y: 0 }); setHovering(false); }}
      style={{
        position: "relative", overflow: "hidden",
        background: "var(--color-bg-card)", border: "none", borderRadius: 10, padding: "18px 20px",
        boxShadow: hovering ? "var(--shadow-hover), var(--shadow-glow-sm)" : "var(--shadow-card)",
        transform: hovering ? `perspective(600px) rotateX(${rot.x}deg) rotateY(${rot.y}deg) translateY(-3px)` : "perspective(600px) rotateX(0) rotateY(0) translateY(0)",
        transition: "box-shadow 0.35s, transform 0.3s", transformStyle: "preserve-3d",
      }}
    >


      {/* Top edge glow */}
      <div style={{
        position: "absolute", top: 0, left: 0, right: 0, height: 1,
        background: hovering ? "linear-gradient(90deg, transparent, rgba(255,255,255,0.06), transparent)" : "linear-gradient(90deg, transparent, rgba(255,255,255,0.02), transparent)",
        transition: "background 0.3s", pointerEvents: "none",
      }} />

      <div style={{ position: "relative", zIndex: 2 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 10 }}>
          {icon && <span style={{ fontSize: 16, filter: hovering ? "none" : "grayscale(30%)", transition: "filter 0.3s" }}>{icon}</span>}
          <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 1 }}>{label}</span>
        </div>
        <div style={{ display: "flex", alignItems: "baseline", gap: 5 }}>
          <span style={{ fontSize: 28, fontWeight: 800, color: "var(--color-text)", letterSpacing: -1.5, lineHeight: 1, transition: valueTransition }}>{displayValue}</span>
          {suffix && <span style={{ fontSize: 12, color: "var(--color-text-muted)", fontWeight: 500 }}>{suffix}</span>}
        </div>
      </div>

      {/* Bottom accent bar */}
      <motion.div initial={{ scaleX: 0 }} animate={{ scaleX: hovering ? 1 : 0 }} transition={{ duration: 0.3 }}
        style={{ position: "absolute", bottom: 0, left: 0, right: 0, height: 2, background: `linear-gradient(90deg, ${accentColor}, transparent)`, transformOrigin: "left" }} />
    </motion.div>
  );
}
