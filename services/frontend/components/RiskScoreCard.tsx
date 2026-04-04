"use client";

import React from "react";
import { motion } from "framer-motion";
import TrustScoreRing from "@/components/ui/TrustScoreRing";

interface RiskScoreCardProps {
  score: number;
  band: string;
}

export default function RiskScoreCard({ score, band }: RiskScoreCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.5, ease: "easeOut" }}
      whileHover={{ y: -4, scale: 1.02, transition: { duration: 0.25 } }}
      className="relative overflow-hidden rounded-2xl p-6"
      style={{
        background:
          "linear-gradient(145deg, rgba(16,185,129,0.08) 0%, rgba(14,165,233,0.04) 100%)",
        backdropFilter: "blur(16px)",
        WebkitBackdropFilter: "blur(16px)",
        border: "1px solid rgba(16,185,129,0.12)",
        boxShadow: "var(--shadow-glow-sm), var(--shadow-card)",
      }}
    >
      {/* Subtle gradient overlay */}
      <div
        className="pointer-events-none absolute inset-0 opacity-30"
        style={{
          background:
            "radial-gradient(ellipse at 50% 0%, rgba(16,185,129,0.15) 0%, transparent 60%)",
        }}
      />

      <div className="relative z-10 flex flex-col items-center text-center">
        <p
          style={{
            fontSize: 11,
            fontWeight: 700,
            color: "rgba(255,255,255,0.4)",
            letterSpacing: "0.1em",
            textTransform: "uppercase",
            marginBottom: 16,
          }}
        >
          RISK SCORE
        </p>

        <TrustScoreRing
          score={score}
          riskBand={band}
          size={140}
          strokeWidth={8}
        />
      </div>
    </motion.div>
  );
}
