"use client";
import React, { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import Navbar from "@/components/Navbar";
import ChitFundCard from "@/components/ChitFundCard";
import AnimatedButton from "@/components/ui/AnimatedButton";
import { listFunds, type FundDetails } from "@/services/api";

export default function FundsPage() {
  const router = useRouter();
  const [funds, setFunds] = useState<FundDetails[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const loadFunds = async () => {
      if (cancelled) return;
      setLoading(true);
      setError(null);
      try {
        const data = await listFunds();
        if (!cancelled) setFunds(data || []);
      } catch (e: unknown) {
        if (!cancelled) {
          const message = e instanceof Error ? e.message : "Failed to fetch funds";
          setError(message);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    void loadFunds();

    const onPageShow = () => {
      void loadFunds();
    };

    window.addEventListener("pageshow", onPageShow);
    return () => {
      cancelled = true;
      window.removeEventListener("pageshow", onPageShow);
    };
  }, []);

  return (
    <div style={{ background: "var(--color-bg)", minHeight: "100vh" }}>
      <Navbar />
      <main style={{ maxWidth: 1200, margin: "0 auto", padding: "24px 20px" }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
          <div>
            <h1 style={{ fontSize: 22, fontWeight: 700, color: "var(--color-text)", margin: 0 }}>Funds</h1>
            <p style={{ fontSize: 12, color: "var(--color-text-muted)", margin: "2px 0 0" }}>Browse and join chit funds.</p>
          </div>
          <span style={{ fontSize: 10, fontWeight: 600, color: "var(--color-text-muted)", background: "var(--color-bg-card)", border: "1px solid var(--color-border)", borderRadius: 3, padding: "3px 8px" }}>{funds.length} Active</span>
        </div>

        {loading ? (
          <div style={{ display: "flex", justifyContent: "center", padding: 48 }}><div style={{ width: 28, height: 28, borderRadius: "50%", border: "2px solid var(--color-border)", borderTopColor: "var(--color-accent)", animation: "spin 0.7s linear infinite" }} /></div>
        ) : error ? (
          <div style={{ fontSize: 13, color: "var(--color-danger)", background: "var(--color-danger-light)", border: "1px solid rgba(239,68,68,0.15)", borderRadius: 6, padding: "10px 14px" }}>{error}</div>
        ) : funds.length === 0 ? (
          <div style={{ textAlign: "center", padding: "40px 0", background: "var(--color-bg-card)", border: "1px dashed var(--color-border)", borderRadius: 8 }}>
            <span style={{ fontSize: 28 }}>💰</span>
            <p style={{ fontSize: 14, fontWeight: 600, color: "var(--color-text)", margin: "8px 0 4px" }}>No Funds</p>
            <p style={{ fontSize: 12, color: "var(--color-text-muted)", marginBottom: 14 }}>Be the first to create one</p>
            <AnimatedButton variant="primary" size="sm" onClick={() => { router.push("/dashboard?tab=create"); }}>Create Fund →</AnimatedButton>
          </div>
        ) : (
          <motion.div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(260px, 1fr))", gap: 10 }} initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
            {funds.map((f) => {
              const isFull = (f.current_member_count ?? 0) >= (f.max_members ?? 0);
              const displayStatus = isFull ? "full" : f.status;
              return <ChitFundCard key={f._id} id={f._id} name={f.name} totalPool={f.total_amount || 0} totalMembers={f.max_members || 0} monthlyContribution={f.monthly_contribution || 0} minRiskScore={0} status={displayStatus} onClick={() => { router.push(`/fund/${f._id}`); }} />;
            })}
          </motion.div>
        )}
      </main>
    </div>
  );
}
