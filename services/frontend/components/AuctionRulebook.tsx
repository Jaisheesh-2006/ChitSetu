"use client";

import React from "react";
import { motion, AnimatePresence } from "framer-motion";
import AnimatedButton from "@/components/ui/AnimatedButton";

interface Props {
  isOpen: boolean;
  onClose: () => void;
  onEnter: () => void;
  fundName: string;
  monthlyContribution: number;
  maxMembers: number;
}

function fmt(n: number) {
  return new Intl.NumberFormat("en-IN", { style: "currency", currency: "INR", maximumFractionDigits: 0 }).format(n);
}

const rules = [
  {
    icon: "📊",
    title: "Reverse Auction",
    description: "This is a reverse (discount) auction. Members bid to offer a discount on the total pool. The member offering the highest discount wins the pot.",
  },
  {
    icon: "💸",
    title: "Discount Bidding",
    description: "Bids are placed as increments of ₹10, ₹100, or ₹200. Every accepted bid is added on top of the current discount (strict incremental flow).",
  },
  {
    icon: "🛡️",
    title: "Fair Bid Controls",
    description: "No member can place two consecutive bids, and the total discount cannot exceed 50% of the monthly pool.",
  },
  {
    icon: "⏱️",
    title: "20-Second Countdown",
    description: "After each bid, a 20-second countdown begins. If no new bid is placed within 20 seconds, the auction ends and the last bidder wins.",
  },
  {
    icon: "🏆",
    title: "Winner Takes the Pot",
    description: "The winner receives the total pool minus the accumulated discount. The discount is distributed equally among all other members as a dividend.",
  },
  {
    icon: "✅",
    title: "Contribution Required",
    description: "You must have paid your monthly contribution for the current cycle before you can place any bids.",
  },
  {
    icon: "🔒",
    title: "One Win Per Member",
    description: "Each member can win the pot only once during the fund's lifetime. Previous winners are ineligible to bid.",
  },
];

export default function AuctionRulebook({ isOpen, onClose, onEnter, fundName, monthlyContribution, maxMembers }: Props) {
  if (!isOpen) return null;

  const totalPool = monthlyContribution * maxMembers;

  return (
    <>
      {/* Backdrop */}
      <div onClick={onClose} style={{ position: "fixed", inset: 0, backgroundColor: "rgba(0,0,0,0.65)", backdropFilter: "blur(6px)", zIndex: 1000 }} />

      {/* Modal */}
      <div style={{ position: "fixed", inset: 0, display: "flex", alignItems: "center", justifyContent: "center", zIndex: 1001, padding: 16, pointerEvents: "none" }}>
        <AnimatePresence>
          <motion.div
            initial={{ opacity: 0, scale: 0.92, y: 24 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.92, y: 24 }}
            transition={{ duration: 0.3, ease: "easeOut" }}
            style={{
              pointerEvents: "auto", width: "100%", maxWidth: 520, maxHeight: "85vh",
              borderRadius: 14, overflow: "hidden",
              background: "var(--color-bg-card)", boxShadow: "var(--shadow-elevated)",
            }}
            onClick={(e) => e.stopPropagation()}
          >
            {/* Header */}
            <div style={{ background: "var(--gradient-primary)", padding: "20px 24px 16px", display: "flex", alignItems: "flex-start", justifyContent: "space-between" }}>
              <div>
                <p style={{ fontSize: 10, fontWeight: 700, letterSpacing: 1.2, color: "rgba(255,255,255,0.7)", textTransform: "uppercase", marginBottom: 4 }}>
                  AUCTION RULES
                </p>
                <h2 style={{ fontSize: 18, fontWeight: 800, color: "#fff", margin: 0, letterSpacing: -0.3 }}>
                  {fundName}
                </h2>
              </div>
              <button
                onClick={onClose}
                style={{
                  background: "rgba(255,255,255,0.15)", border: "none", borderRadius: 8,
                  width: 30, height: 30, display: "flex", alignItems: "center", justifyContent: "center",
                  cursor: "pointer", color: "#fff", fontSize: 18, lineHeight: 1,
                }}
              >
                ×
              </button>
            </div>

            {/* Body — scrollable */}
            <div style={{ padding: "20px 24px 28px", overflowY: "auto", maxHeight: "calc(85vh - 140px)" }}>
              {/* Pool info */}
              <div style={{
                background: "var(--color-bg-subtle)", borderRadius: 10, padding: "14px 18px", marginBottom: 20,
                display: "flex", justifyContent: "space-between", alignItems: "center", gap: 12,
                boxShadow: "inset 0 1px 3px rgba(0,0,0,0.2)",
              }}>
                <div>
                  <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 1 }}>
                    Total Pool This Cycle
                  </span>
                  <p style={{ fontSize: 22, fontWeight: 800, color: "var(--color-text)", margin: "4px 0 0", letterSpacing: -0.5 }}>
                    {fmt(totalPool)}
                  </p>
                </div>
                <div style={{ textAlign: "right" }}>
                  <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 1 }}>
                    Contribution
                  </span>
                  <p style={{ fontSize: 16, fontWeight: 700, color: "var(--color-accent)", margin: "4px 0 0" }}>
                    {fmt(monthlyContribution)} × {maxMembers}
                  </p>
                </div>
              </div>

              {/* Rules list */}
              <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
                {rules.map((rule, idx) => (
                  <motion.div
                    key={rule.title}
                    initial={{ opacity: 0, x: -12 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: 0.05 * idx, duration: 0.3 }}
                    style={{
                      display: "flex", gap: 14, alignItems: "flex-start",
                      padding: "12px 14px", borderRadius: 10,
                      background: "rgba(255,255,255,0.02)",
                      borderBottom: "1px solid rgba(255,255,255,0.03)",
                    }}
                  >
                    <div style={{
                      width: 36, height: 36, borderRadius: 8, flexShrink: 0,
                      background: "var(--color-bg-subtle)",
                      display: "flex", alignItems: "center", justifyContent: "center",
                      fontSize: 18, boxShadow: "var(--shadow-card)",
                    }}>
                      {rule.icon}
                    </div>
                    <div>
                      <p style={{ fontSize: 13, fontWeight: 700, color: "var(--color-text)", margin: "0 0 4px" }}>
                        {idx + 1}. {rule.title}
                      </p>
                      <p style={{ fontSize: 12, color: "var(--color-text-secondary)", lineHeight: 1.6, margin: 0 }}>
                        {rule.description}
                      </p>
                    </div>
                  </motion.div>
                ))}
              </div>

              {/* Example */}
              <div style={{
                marginTop: 18, padding: "14px 16px", borderRadius: 10,
                background: "rgba(249,115,22,0.05)", border: "1px solid rgba(249,115,22,0.12)",
              }}>
                <p style={{ fontSize: 11, fontWeight: 700, color: "var(--color-accent)", textTransform: "uppercase", letterSpacing: 0.8, marginBottom: 6 }}>
                  💡 Example
                </p>
                <p style={{ fontSize: 12, color: "var(--color-text-secondary)", lineHeight: 1.7, margin: 0 }}>
                  Pool = {fmt(totalPool)}. If the winning discount is ₹500, the winner receives{" "}
                  <strong style={{ color: "var(--color-text)" }}>{fmt(totalPool - 500)}</strong>{" "}
                  and each of the other {maxMembers - 1} members receives a dividend of{" "}
                  <strong style={{ color: "var(--color-success)" }}>{fmt(Math.floor(500 / (maxMembers - 1)))}</strong>.
                </p>
              </div>

              {/* CTA */}
              <div style={{ marginTop: 22, display: "flex", gap: 10 }}>
                <AnimatedButton variant="ghost" size="md" onClick={onClose} fullWidth>
                  Cancel
                </AnimatedButton>
                <AnimatedButton variant="primary" size="lg" onClick={onEnter} fullWidth>
                  Enter Auction Room →
                </AnimatedButton>
              </div>
            </div>
          </motion.div>
        </AnimatePresence>
      </div>
    </>
  );
}
