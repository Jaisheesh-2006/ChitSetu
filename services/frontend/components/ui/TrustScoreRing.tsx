"use client";
import React, { useEffect, useState } from "react";
import { motion } from "framer-motion";

interface Props { score: number; maxScore?: number; riskBand?: string; size?: number; strokeWidth?: number; showLabel?: boolean; animated?: boolean; }
const COLORS: Record<string, string> = { Excellent: "#22c55e", Good: "#34d399", Average: "#f59e0b", Risky: "#f97316", "High Risk": "#ef4444", low: "#22c55e", medium: "#f59e0b", high: "#ef4444" };
function grade(s: number, m: number) { const p = s / m; return p >= 0.8 ? "Excellent" : p >= 0.65 ? "Good" : p >= 0.5 ? "Average" : p >= 0.35 ? "Risky" : "High Risk"; }

export default function TrustScoreRing({ score, maxScore = 1000, riskBand, size = 140, strokeWidth = 8, showLabel = true, animated = true }: Props) {
  const [d, setD] = useState(animated ? 0 : score);
  const g = riskBand || grade(score, maxScore);
  const c = COLORS[g] || "#f97316";
  const r = (size - strokeWidth) / 2;
  const circ = 2 * Math.PI * r;
  const off = circ * (1 - (animated ? d / maxScore : Math.min(score / maxScore, 1)));

  useEffect(() => { if (!animated) { setD(score); return; } setD(0); const s = Date.now(); const t = setInterval(() => { const p = Math.min((Date.now() - s) / 1200, 1); setD(Math.round((1 - Math.pow(1 - p, 3)) * score)); if (p >= 1) clearInterval(t); }, 16); return () => clearInterval(t); }, [score, animated]);

  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 8 }}>
      <div style={{ position: "relative", width: size, height: size }}>
        <svg width={size} height={size} style={{ transform: "rotate(-90deg)" }}>
          <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="var(--color-border)" strokeWidth={strokeWidth} />
          <motion.circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke={c} strokeWidth={strokeWidth} strokeLinecap="round" strokeDasharray={circ} initial={{ strokeDashoffset: circ }} animate={{ strokeDashoffset: off }} transition={{ duration: 1.2, ease: "easeOut" }} />
        </svg>
        <div style={{ position: "absolute", inset: 0, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center" }}>
          <span style={{ fontSize: size * 0.22, fontWeight: 800, color: c, lineHeight: 1, letterSpacing: -1 }}>{d}</span>
          <span style={{ fontSize: size * 0.07, color: "var(--color-text-muted)", marginTop: 2 }}>/ {maxScore}</span>
        </div>
      </div>
      {showLabel && <span style={{ display: "inline-flex", alignItems: "center", gap: 4, fontSize: 11, fontWeight: 600, color: c, background: `${c}12`, border: `1px solid ${c}25`, borderRadius: 4, padding: "2px 8px" }}><span style={{ width: 5, height: 5, borderRadius: "50%", background: c }} />{g}</span>}
    </div>
  );
}
