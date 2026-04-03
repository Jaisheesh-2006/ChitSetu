"use client";

import React, { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import Alert from "@mui/material/Alert";
import { motion, AnimatePresence } from "framer-motion";
import Navbar from "@/components/Navbar";
import ChitFundCard from "@/components/ChitFundCard";
import KYCWizardModal from "@/components/KYCWizardModal";
import GlassCard from "@/components/ui/GlassCard";
import AnimatedButton from "@/components/ui/AnimatedButton";
import StatCard from "@/components/ui/StatCard";
import TrustScoreRing from "@/components/ui/TrustScoreRing";
import { useAuth } from "@/hooks/useAuth";
import { 
  upsertProfile, getProfile, createFund, getMyFunds, getRiskScore, getMyContributions, 
  getWalletInfo, type ProfileInput, type CreateFundInput, type WalletInfo 
} from "@/services/api";

type Tab = "overview" | "profile" | "create";

/* ── Custom Input ── */
function Input({ label, icon, value, onChange, type = "text", disabled = false, required = false, rows, placeholder, span2 }: {
  label: string; icon?: string; value: string | number; onChange: (v: string) => void;
  type?: string; disabled?: boolean; required?: boolean; rows?: number; placeholder?: string; span2?: boolean;
}) {
  const [focused, setFocused] = useState(false);

  const openDatePicker = (el: HTMLInputElement) => {
    if (type !== "date") return;
    const maybeInput = el as HTMLInputElement & { showPicker?: () => void };
    if (!maybeInput.showPicker) return;
    try {
      maybeInput.showPicker();
    } catch {
      // Ignore browsers that block showPicker without a direct user gesture.
    }
  };

  return (
    <div style={{ gridColumn: span2 ? "1 / -1" : undefined }}>
      <label style={{ display: "flex", alignItems: "center", gap: 5, fontSize: 11, fontWeight: 600, color: focused ? "var(--color-accent)" : "var(--color-text-muted)", marginBottom: 6, letterSpacing: 0.5, textTransform: "uppercase", transition: "color 0.2s" }}>
        {icon && <span style={{ fontSize: 13 }}>{icon}</span>}
        {label}
        {required && <span style={{ color: "var(--color-accent)", fontSize: 10 }}>*</span>}
      </label>
      {rows ? (
        <textarea
          value={value} onChange={(e) => onChange(e.target.value)}
          disabled={disabled} required={required} rows={rows} placeholder={placeholder}
          onFocus={() => setFocused(true)} onBlur={() => setFocused(false)}
          style={{
            width: "100%", resize: "none", fontFamily: '"Inter",sans-serif',
            background: disabled ? "rgba(255,255,255,0.01)" : "var(--color-bg-subtle)",
            border: "none", borderRadius: 8, padding: "10px 14px",
            color: disabled ? "var(--color-text-secondary)" : "var(--color-text)",
            fontSize: 13, outline: "none",
            boxShadow: focused ? "inset 0 2px 4px rgba(0,0,0,0.3), 0 0 0 2px rgba(249,115,22,0.15)" : "inset 0 2px 4px rgba(0,0,0,0.2)",
            transition: "box-shadow 0.2s",
            opacity: disabled ? 0.6 : 1,
          }}
        />
      ) : (
        <input
          type={type} value={value} onChange={(e) => onChange(e.target.value)}
          disabled={disabled} required={required} placeholder={placeholder}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          onClick={(e) => {
            openDatePicker(e.currentTarget);
          }}
          onKeyDown={(e) => {
            if (type === "date") e.preventDefault();
          }}
          style={{
            width: "100%", fontFamily: '"Inter",sans-serif',
            background: disabled ? "rgba(255,255,255,0.01)" : "var(--color-bg-subtle)",
            border: "none", borderRadius: 8, padding: "10px 14px",
            color: disabled ? "var(--color-text-secondary)" : "var(--color-text)",
            fontSize: 13, outline: "none",
            boxShadow: focused ? "inset 0 2px 4px rgba(0,0,0,0.3), 0 0 0 2px rgba(249,115,22,0.15)" : "inset 0 2px 4px rgba(0,0,0,0.2)",
            transition: "box-shadow 0.2s",
            opacity: disabled ? 0.6 : 1,
            cursor: type === "date" ? "pointer" : "text",
          }}
        />
      )}
    </div>
  );
}

function CountUp({ value, prefix = "" }: { value: number; prefix?: string }) {
  const [d, setD] = useState(0);
  useEffect(() => { const s = Date.now(); const t = setInterval(() => { const p = Math.min((Date.now() - s) / 1500, 1); setD(Math.round((1 - Math.pow(1 - p, 3)) * value)); if (p >= 1) clearInterval(t); }, 16); return () => clearInterval(t); }, [value]);
  return <span>{prefix}{d.toLocaleString("en-IN")}</span>;
}

/* ───────────────────────────────────────── */
/* ── Overview ──                            */
/* ───────────────────────────────────────── */
function Overview() {
  const [loading, setLoading] = useState(true);
  const [funds, setFunds] = useState<any[]>([]);
  const [risk, setRisk] = useState<any>(null);
  const [contribs, setContribs] = useState<any[]>([]);

  const [wallet, setWallet] = useState<WalletInfo | null>(null);

  useEffect(() => {
    (async () => {
      const [f, r, c, w] = await Promise.allSettled([
        getMyFunds(), 
        getRiskScore(), 
        getMyContributions(),
        getWalletInfo()
      ]);
      if (f.status === "fulfilled") setFunds(f.value || []);
      if (r.status === "fulfilled") setRisk(r.value);
      if (c.status === "fulfilled") setContribs(c.value || []);
      if (w.status === "fulfilled") setWallet(w.value);
      setLoading(false);
    })();

    const interval = setInterval(async () => {
      try {
        const [c,w] = await Promise.allSettled([getMyContributions(), getWalletInfo()]);
        if (c.status === "fulfilled") setContribs(c.value || []);
        if (w.status === "fulfilled") setWallet(w.value);
      } catch {
        // Ignore errors in periodic refresh
      }
    }, 5000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, []);

  if (loading) return <div style={{ display: "flex", justifyContent: "center", padding: 64 }}><motion.div animate={{ rotate: 360 }} transition={{ repeat: Infinity, duration: 1, ease: "linear" }} style={{ width: 28, height: 28, borderRadius: "50%", border: "2px solid rgba(255,255,255,0.04)", borderTopColor: "var(--color-accent)" }} /></div>;

  const total = contribs.reduce((s: number, c: any) => s + (c.amount_due || 0), 0);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      <GlassCard hover={true} depth={true} delay={0} style={{ position: "relative", overflow: "hidden" }}>
        <div style={{ position: "absolute", top: -40, right: -40, width: 160, height: 160, borderRadius: "50%", background: "radial-gradient(circle, rgba(249,115,22,0.06) 0%, transparent 70%)", pointerEvents: "none" }} />
        <div style={{ display: "flex", flexWrap: "wrap", justifyContent: "space-between", alignItems: "center", gap: 12, position: "relative" }}>
          <div>
            <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 6 }}>
              <div style={{ width: 8, height: 8, borderRadius: 2, background: "var(--color-accent)", boxShadow: "0 0 6px rgba(249,115,22,0.4)" }} />
              <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 1.2 }}>Portfolio Value</span>
            </div>
            <p style={{ fontSize: 36, fontWeight: 800, color: "var(--color-text)", margin: "0 0 4px", letterSpacing: -1.5, lineHeight: 1 }}><CountUp value={total} prefix="₹" /></p>
            <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>{funds.length} fund{funds.length !== 1 ? "s" : ""} · {contribs.length} payment{contribs.length !== 1 ? "s" : ""}</span>
          </div>
          <div style={{ display: "flex", alignItems: "center", gap: 5, background: "rgba(34,197,94,0.06)", borderRadius: 6, padding: "5px 12px", boxShadow: "var(--shadow-card)" }}>
            <motion.div animate={{ scale: [1, 1.3, 1] }} transition={{ repeat: Infinity, duration: 2 }} style={{ width: 6, height: 6, borderRadius: "50%", background: "#22c55e" }} />
            <span style={{ fontSize: 11, fontWeight: 600, color: "#22c55e" }}>Active</span>
          </div>
        </div>
      </GlassCard>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(160px, 1fr))", gap: 10 }}>
        <StatCard label="Active Funds" value={funds.length} delay={0.05} icon={<span>📊</span>} accent="#60a5fa" />
        <StatCard label="Contributions" value={contribs.length} suffix="paid" delay={0.1} icon={<span>💰</span>} accent="#f59e0b" />
        {risk ? (
          <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.15 }}
            style={{ position: "relative", overflow: "hidden", background: "var(--color-bg-card)", borderRadius: 10, padding: "18px 20px", display: "flex", alignItems: "center", gap: 14, boxShadow: "var(--shadow-card)" }}>
            <TrustScoreRing score={risk.score} riskBand={risk.risk_category} size={60} strokeWidth={5} showLabel={false} />
            <div>
              <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 1 }}>Trust Score</span>
              <p style={{ fontSize: 24, fontWeight: 800, color: "var(--color-text)", margin: "2px 0 0", letterSpacing: -1 }}>{risk.score}</p>
            </div>
            <div style={{ position: "absolute", bottom: 0, left: 0, right: 0, height: 2, background: "linear-gradient(90deg, #22c55e, transparent)", opacity: 0.4 }} />
          </motion.div>
        ) : <StatCard label="Trust Score" value="—" delay={0.15} icon={<span>🛡️</span>} accent="#22c55e" />}
      </div>

      {/* Wallet & Balance Card */}
      <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.2 }}>
        <GlassCard hover={false} depth={true}>
          <div style={{ display: "flex", flexWrap: "wrap", justifyContent: "space-between", alignItems: "center", gap: 16 }}>
            <div style={{ flex: 1, minWidth: 200 }}>
              <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 8 }}>
                <span style={{ fontSize: 14 }}>👛</span>
                <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 1 }}>Your Custodial Wallet</span>
              </div>
              <div style={{ display: "flex", alignItems: "center", gap: 8, background: "var(--color-bg-subtle)", padding: "8px 12px", borderRadius: 8, boxShadow: "inset 0 1px 3px rgba(0,0,0,0.2)" }}>
                <code style={{ fontSize: 12, color: "var(--color-text-secondary)", fontFamily: "monospace", overflow: "hidden", textOverflow: "ellipsis" }}>
                  {wallet?.address || "Loading..."}
                </code>
                {wallet?.address && (
                  <button 
                    onClick={() => {
                      navigator.clipboard.writeText(wallet.address);
                      alert("Address copied to clipboard!");
                    }}
                    style={{ background: "transparent", border: "none", cursor: "pointer", fontSize: 14, opacity: 0.6, padding: 4 }}
                    title="Copy Address"
                  >
                    📋
                  </button>
                )}
              </div>
            </div>

            <div style={{ textAlign: "right" }}>
              <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 1 }}>Token Balance</span>
              <p style={{ fontSize: 28, fontWeight: 800, margin: "4px 0 0", color: "var(--color-text)", letterSpacing: -1 }}>
                {wallet ? wallet.balance.toLocaleString() : "—"} 
                <span style={{ fontSize: 14, color: "var(--color-accent)", marginLeft: 6, fontWeight: 700 }}>CHIT</span>
              </p>
            </div>
          </div>
        </GlassCard>
      </motion.div>

      <div>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12 }}>
          <h3 style={{ fontSize: 15, fontWeight: 700, color: "var(--color-text)", margin: 0 }}>Your Funds</h3>
          <span style={{ fontSize: 10, fontWeight: 600, color: "var(--color-text-muted)", background: "var(--color-bg-subtle)", borderRadius: 4, padding: "3px 8px", boxShadow: "var(--shadow-card)" }}>{funds.length}</span>
        </div>
        {funds.length === 0 ? (
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }}
            style={{ textAlign: "center", padding: "40px 20px", background: "var(--color-bg-card)", borderRadius: 10, boxShadow: "var(--shadow-card)" }}>
            <motion.span animate={{ y: [0, -6, 0] }} transition={{ repeat: Infinity, duration: 3, ease: "easeInOut" }} style={{ fontSize: 32, display: "inline-block" }}>🏦</motion.span>
            <p style={{ fontSize: 14, fontWeight: 600, color: "var(--color-text-secondary)", margin: "10px 0 4px" }}>No funds yet</p>
            <p style={{ fontSize: 12, color: "var(--color-text-muted)", marginBottom: 14 }}>Create or join a chit fund to get started</p>
            <AnimatedButton variant="primary" size="sm" onClick={() => { window.location.href = "/funds" }}>Browse Funds →</AnimatedButton>
          </motion.div>
        ) : (
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(260px, 1fr))", gap: 10 }}>
            {funds.map((f) => <ChitFundCard key={f._id} id={f._id} name={f.name} totalPool={f.total_amount || 0} totalMembers={f.current_member_count || 0} monthlyContribution={f.monthly_contribution || 0} minRiskScore={0} status={f.status} onClick={() => { window.location.href = `/fund/${f._id}` }} />)}
          </div>
        )}
      </div>

      <div>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12, marginTop: 10 }}>
          <h3 style={{ fontSize: 15, fontWeight: 700, color: "var(--color-text)", margin: 0 }}>Recent Contributions</h3>
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {contribs.length === 0 ? (
             <p style={{ fontSize: 12, color: "var(--color-text-muted)" }}>No payments recorded yet.</p>
          ) : (
            contribs.slice(0, 5).map((c) => {
              const status = (c.status === "paid" && c.blockchain_status && c.blockchain_status !== "confirmed") ? c.blockchain_status : c.status;
              const style = status === "paid" || status === "confirmed" ? { color: "#22c55e", bg: "rgba(34,197,94,0.08)" } 
                          : status === "pending" ? { color: "#f59e0b", bg: "rgba(245,158,11,0.08)" }
                          : status === "minting tokens" ? { color: "#60a5fa", bg: "rgba(96,165,250,0.08)" }
                          : status === "depositing to pool" ? { color: "#f472b6", bg: "rgba(244,114,182,0.08)" }
                          : { color: "var(--color-text-secondary)", bg: "var(--color-bg-subtle)" };
              
              return (
                <GlassCard key={`${c.fund_id}-${c.cycle_number}`} padding="p-3" hover={true}>
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                    <div>
                      <p style={{ fontSize: 13, fontWeight: 700, margin: 0 }}>{c.fund_name || "Fund Payment"}</p>
                      <p style={{ fontSize: 11, color: "var(--color-text-muted)", margin: "2px 0 0" }}>Cycle {c.cycle_number} · ₹{c.amount_due}</p>
                    </div>
                    <div style={{ display: "flex", flexDirection: "column", alignItems: "flex-end", gap: 4 }}>
                      <span style={{ fontSize: 10, fontWeight: 700, padding: "2px 8px", borderRadius: 4, color: style.color, background: style.bg, textTransform: "uppercase" }}>
                        {status}
                      </span>
                      {c.status === "paid" && c.blockchain_status && c.blockchain_status !== "confirmed" && (
                        <motion.span animate={{ opacity: [1, 0.5, 1] }} transition={{ repeat: Infinity, duration: 1.5 }} style={{ fontSize: 9, color: "var(--color-text-secondary)" }}>
                          Syncing to Blockchain...
                        </motion.span>
                      )}
                    </div>
                  </div>
                </GlassCard>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
}

/* ───────────────────────────────────────── */
/* ── Profile ──                             */
/* ───────────────────────────────────────── */
function Profile({onUpdate}: { onUpdate: () => void }) {
  const [form, setForm] = useState<ProfileInput>({ full_name: "", age: 0, phone_number: "", pan_number: "", monthly_income: 0, employment_years: 0 });
  const [loading, setLoading] = useState(false);
  const [init, setInit] = useState(true);
  const [edit, setEdit] = useState(false);
  const [msg, setMsg] = useState<{ t: "success" | "error"; m: string } | null>(null);
  const [kycOpen, setKycOpen] = useState(false);
  const [kycDone, setKycDone] = useState(false);
  const [trust, setTrust] = useState<{ score: number; riskBand: string } | null>(null);

  useEffect(() => {
    getProfile().then((d: any) => {
      if (d?.full_name) {
        setForm({ full_name: d.full_name, age: d.age || 0, phone_number: d.phone_number || "", pan_number: d.profile?.pan || d.pan_number || "", monthly_income: d.profile?.monthly_income || d.monthly_income || 0, employment_years: d.employment_years || 0 });
        setEdit(false);
        if (d.credit?.score > 0) { setTrust({ score: d.credit.score, riskBand: d.credit.risk_category }); setKycDone(true); }
      } else setEdit(true);
    }).catch(() => setEdit(true)).finally(() => setInit(false));
  }, []);

  const up = (f: keyof ProfileInput) => (v: string) =>
    setForm((p) => ({ ...p, [f]: ["age", "monthly_income", "employment_years"].includes(f) ? Number(v) || 0 : v }));

  const sub = async (e: React.FormEvent) => {
    e.preventDefault(); setMsg(null); setLoading(true);
    try {
      await upsertProfile(form);
      setMsg({ t: "success", m: "Saved. Launching KYC…" });
      setEdit(false);
      onUpdate?.();
      setTimeout(() => { setMsg(null); setKycOpen(true); }, 800);
    }
    catch (e: unknown) { setMsg({ t: "error", m: e instanceof Error ? e.message : "Failed" }); }
    finally { setLoading(false); }
  };

  if (init) return <div style={{ display: "flex", justifyContent: "center", padding: 48 }}><motion.div animate={{ rotate: 360 }} transition={{ repeat: Infinity, duration: 1, ease: "linear" }} style={{ width: 28, height: 28, borderRadius: "50%", border: "2px solid rgba(255,255,255,0.04)", borderTopColor: "var(--color-accent)" }} /></div>;

  return (
    <>
      <KYCWizardModal isOpen={kycOpen} onClose={() => {
        setKycOpen(false);
        if (!kycDone) {
          setEdit(true);
          setMsg({ t: "error", m: "KYC incomplete or PAN verification failed. Please check your details and save again." });
        }
      }} onComplete={(r) => { setTrust({ score: r.score, riskBand: r.riskBand }); setKycDone(true); onUpdate?.()}} />

      <div style={{ display: "grid", gridTemplateColumns: kycDone && trust ? "280px 1fr" : "1fr", gap: 16, alignItems: "start" }}>

        {/* Trust Score Card */}
        {kycDone && trust && (
          <GlassCard hover={true} depth={true} delay={0.05} style={{ position: "relative", overflow: "hidden" }}>
            {/* Background glow */}
            <div style={{ position: "absolute", top: -30, left: "50%", transform: "translateX(-50%)", width: 200, height: 200, borderRadius: "50%", background: "radial-gradient(circle, rgba(34,197,94,0.04), transparent)", pointerEvents: "none" }} />

            <div style={{ position: "relative", textAlign: "center" }}>
              <div style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: 5, marginBottom: 14 }}>
                <div style={{ width: 6, height: 6, borderRadius: "50%", background: "#22c55e", boxShadow: "0 0 6px rgba(34,197,94,0.4)" }} />
                <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", letterSpacing: 1.2, textTransform: "uppercase" }}>Trust Score</span>
              </div>

              <div style={{ margin: "0 auto 14px" }}>
                <TrustScoreRing score={trust.score} riskBand={trust.riskBand} size={140} strokeWidth={8} />
              </div>

              {/* Mini stats */}
              <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8, marginBottom: 12 }}>
                <div style={{ background: "var(--color-bg-subtle)", borderRadius: 6, padding: "8px 10px", boxShadow: "inset 0 1px 3px rgba(0,0,0,0.2)" }}>
                  <p style={{ fontSize: 9, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 0.5, margin: 0 }}>Band</p>
                  <p style={{ fontSize: 12, fontWeight: 700, color: "#22c55e", margin: "2px 0 0" }}>{trust.riskBand}</p>
                </div>
                <div style={{ background: "var(--color-bg-subtle)", borderRadius: 6, padding: "8px 10px", boxShadow: "inset 0 1px 3px rgba(0,0,0,0.2)" }}>
                  <p style={{ fontSize: 9, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 0.5, margin: 0 }}>Max</p>
                  <p style={{ fontSize: 12, fontWeight: 700, color: "var(--color-text)", margin: "2px 0 0" }}>1000</p>
                </div>
              </div>

              <AnimatedButton variant="ghost" size="sm" onClick={() => setKycOpen(true)} fullWidth>
                🔄 Recalculate Score
              </AnimatedButton>
            </div>
          </GlassCard>
        )}

        {/* Profile Form */}
        <GlassCard hover={false} depth={false} delay={0.1}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 18 }}>
            <div>
              <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 4 }}>
                <div style={{ width: 6, height: 6, borderRadius: 2, background: "var(--color-accent)", boxShadow: "0 0 6px rgba(249,115,22,0.4)" }} />
                <h2 style={{ fontSize: 17, fontWeight: 700, color: "var(--color-text)", margin: 0 }}>
                  Personal Information
                </h2>
              </div>
              <p style={{ fontSize: 12, color: "var(--color-text-muted)", margin: "4px 0 0 12px" }}>
                {edit ? "Fill in your details for KYC verification." : "Your verified profile information."}
              </p>
            </div>
            <div style={{ display: "flex", gap: 6, flexShrink: 0 }}>
              {!edit && !kycDone && (
                <span style={{ fontSize: 10, fontWeight: 600, color: "#f59e0b", background: "rgba(245,158,11,0.08)", borderRadius: 4, padding: "4px 10px", boxShadow: "0 0 8px rgba(245,158,11,0.06)" }}>⚠ KYC Pending</span>
              )}
              {kycDone && (
                <span style={{ fontSize: 10, fontWeight: 600, color: "#22c55e", background: "rgba(34,197,94,0.08)", borderRadius: 4, padding: "4px 10px" }}>✓ Verified</span>
              )}
            </div>
          </div>

          <form onSubmit={sub}>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
              <Input icon="👤" label="Full Name" value={form.full_name} onChange={up("full_name")} required disabled={!edit} span2 placeholder="Enter your full name" />
              <Input icon="🎂" label="Age" value={form.age || ""} onChange={up("age")} type="number" required disabled={!edit} placeholder="25" />
              <Input icon="📱" label="Phone Number" value={form.phone_number} onChange={up("phone_number")} required disabled={!edit} placeholder="+91 9876543210" />
              <Input icon="🪪" label="PAN Number" value={form.pan_number} onChange={up("pan_number")} required disabled={!edit} placeholder="ABCDE1234F" />
              <Input icon="💰" label="Monthly Income" value={form.monthly_income || ""} onChange={up("monthly_income")} type="number" required disabled={!edit} placeholder="50000" />
              <Input icon="💼" label="Experience (yrs)" value={form.employment_years || ""} onChange={up("employment_years")} type="number" required disabled={!edit} span2={false} placeholder="3" />
            </div>

            {msg && <Alert severity={msg.t} sx={{ mt: 2, borderRadius: "8px", fontSize: "0.8rem", border: "none", boxShadow: "var(--shadow-card)" }}>{msg.m}</Alert>}

            <div style={{ display: "flex", gap: 8, marginTop: 16, paddingTop: 14, borderTop: "1px solid rgba(255,255,255,0.04)" }}>
              {!edit ? (
                <>
                  <AnimatedButton variant="outline" size="sm" onClick={() => setEdit(true)}>✏️ Edit Profile</AnimatedButton>
                  {!kycDone && <AnimatedButton variant="primary" size="sm" onClick={() => setKycOpen(true)}>🛡️ Get Trust Score</AnimatedButton>}
                </>
              ) : (
                <>
                  <AnimatedButton type="submit" variant="primary" size="md" disabled={loading}>{loading ? "Saving…" : "💾 Save & Continue"}</AnimatedButton>
                  {form.full_name && <AnimatedButton variant="ghost" size="sm" onClick={() => setEdit(false)}>Cancel</AnimatedButton>}
                </>
              )}
            </div>
          </form>
        </GlassCard>
      </div>

      {/* Responsive: stack on mobile */}
      <style>{`
        @media (max-width: 700px) {
          .profile-grid { grid-template-columns: 1fr !important; }
        }
      `}</style>
    </>
  );
}

/* ───────────────────────────────────────── */
/* ── Create Fund ──                         */
/* ───────────────────────────────────────── */
function CreateFund() {
  const router = useRouter();
  const [form, setForm] = useState<CreateFundInput>({ name: "", description: "", max_members: 5, duration_months: 12, total_amount: 12000, monthly_contribution: 1000, start_date: "" });
  const [loading, setLoading] = useState(false);
  const [step, setStep] = useState(1);
  const [msg, setMsg] = useState<{ t: "success" | "error"; m: string } | null>(null);

  const up = (f: keyof CreateFundInput) => (v: string) =>
    setForm((p) => ({ ...p, [f]: ["max_members", "monthly_contribution", "total_amount", "duration_months"].includes(f) ? Number(v) || 0 : v }));

  const pool = form.max_members * form.monthly_contribution;
  const tomorrow = new Date(); tomorrow.setDate(tomorrow.getDate() + 1);

  const sub = async (e: React.FormEvent) => {
    e.preventDefault(); setMsg(null); setLoading(true);
    try {
      const r = await createFund({ ...form, total_amount: pool });
      setMsg({ t: "success", m: `Fund created successfully! Redirecting...` });
      setTimeout(() => {
        router.push(`/fund/${r._id}`);
      }, 1500);
    } catch (e: unknown) { setMsg({ t: "error", m: e instanceof Error ? e.message : "Failed" }); }
    finally { setLoading(false); }
  };

  const steps = [
    { n: 1, l: "Fund Details", icon: "📝", desc: "Name & description" },
    { n: 2, l: "Configuration", icon: "⚙️", desc: "Members & amount" },
    { n: 3, l: "Review", icon: "✅", desc: "Confirm & create" },
  ];
  const ok1 = form.name.trim() && form.description.trim();
  const ok2 = form.max_members >= 2 && form.monthly_contribution >= 1 && form.duration_months >= 1;
  const progress = ((step - 1) / 2) * 100;

  return (
    <div style={{ width: "100%" }}>
      {/* Step Progress Header */}
      <GlassCard hover={false} depth={false} delay={0}>
        <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 16 }}>
          <div style={{ width: 6, height: 6, borderRadius: 2, background: "var(--color-accent)", boxShadow: "0 0 6px rgba(249,115,22,0.4)" }} />
          <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 1.2 }}>Create New Fund</span>
        </div>

        {/* Progress bar */}
        <div style={{ height: 3, borderRadius: 2, background: "rgba(255,255,255,0.04)", marginBottom: 20, overflow: "hidden" }}>
          <motion.div animate={{ width: `${progress}%` }} transition={{ duration: 0.4, ease: "easeOut" }}
            style={{ height: "100%", background: "var(--gradient-primary)", borderRadius: 2 }} />
        </div>

        {/* Steps */}
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 8 }}>
          {steps.map((s) => {
            const active = step === s.n;
            const done = step > s.n;
            return (
              <motion.div key={s.n}
                animate={{ scale: active ? 1.02 : 1 }}
                style={{
                  background: active ? "rgba(249,115,22,0.06)" : done ? "rgba(34,197,94,0.04)" : "var(--color-bg-subtle)",
                  borderRadius: 8, padding: "12px 14px",
                  boxShadow: active ? "inset 0 0 0 1px rgba(249,115,22,0.15), var(--shadow-card)" : "inset 0 1px 3px rgba(0,0,0,0.15)",
                  transition: "all 0.3s", cursor: "default",
                }}
              >
                <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 4 }}>
                  <span style={{ fontSize: 16 }}>{done ? "✓" : s.icon}</span>
                  <span style={{ fontSize: 11, fontWeight: 700, color: active ? "var(--color-accent)" : done ? "#22c55e" : "var(--color-text-muted)" }}>
                    Step {s.n}
                  </span>
                </div>
                <p style={{ fontSize: 13, fontWeight: 600, color: active ? "var(--color-text)" : "var(--color-text-secondary)", margin: 0 }}>{s.l}</p>
                <p style={{ fontSize: 10, color: "var(--color-text-muted)", margin: "2px 0 0" }}>{s.desc}</p>
              </motion.div>
            );
          })}
        </div>
      </GlassCard>

      {/* Form Content */}
      <div style={{ marginTop: 14 }}>
        <form onSubmit={sub}>
          <AnimatePresence mode="wait">
            {step === 1 && (
              <motion.div key="s1" initial={{ opacity: 0, x: 24 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -24 }} transition={{ duration: 0.25 }}>
                <GlassCard hover={false} depth={false}>
                  <h3 style={{ fontSize: 15, fontWeight: 700, color: "var(--color-text)", margin: "0 0 4px" }}>Fund Details</h3>
                  <p style={{ fontSize: 12, color: "var(--color-text-muted)", marginBottom: 16 }}>Give your fund a name and description.</p>
                  <div style={{ display: "grid", gap: 14 }}>
                    <Input icon="🏷️" label="Fund Name" value={form.name} onChange={up("name")} required placeholder="e.g. Community Savings Group" span2 />
                    <Input icon="📄" label="Description" value={form.description} onChange={up("description")} required rows={3} placeholder="Describe the purpose of this fund..." span2 />
                  </div>
                  <div style={{ display: "flex", justifyContent: "flex-end", marginTop: 16 }}>
                    <AnimatedButton variant="primary" size="md" onClick={() => setStep(2)} disabled={!ok1}>Next: Configuration →</AnimatedButton>
                  </div>
                </GlassCard>
              </motion.div>
            )}

            {step === 2 && (
              <motion.div key="s2" initial={{ opacity: 0, x: 24 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -24 }} transition={{ duration: 0.25 }}>
                <GlassCard hover={false} depth={false}>
                  <h3 style={{ fontSize: 15, fontWeight: 700, color: "var(--color-text)", margin: "0 0 4px" }}>Configuration</h3>
                  <p style={{ fontSize: 12, color: "var(--color-text-muted)", marginBottom: 16 }}>Set the fund parameters.</p>

                  {/* Live pool preview */}
                  <div style={{ background: "var(--color-bg-subtle)", borderRadius: 8, padding: "12px 16px", marginBottom: 16, display: "flex", justifyContent: "space-between", alignItems: "center", boxShadow: "inset 0 1px 3px rgba(0,0,0,0.2)" }}>
                    <span style={{ fontSize: 11, fontWeight: 600, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 0.5 }}>Live Pool Size</span>
                    <span style={{ fontSize: 20, fontWeight: 800, background: "var(--gradient-primary)", WebkitBackgroundClip: "text", WebkitTextFillColor: "transparent", backgroundClip: "text", letterSpacing: -0.5 }}>₹{pool.toLocaleString("en-IN")}</span>
                  </div>

                  <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 }}>
                    <Input icon="👥" label="Max Members" value={form.max_members || ""} onChange={up("max_members")} type="number" required placeholder="5" />
                    <Input icon="💰" label="Monthly (₹)" value={form.monthly_contribution || ""} onChange={up("monthly_contribution")} type="number" required placeholder="1000" />
                    <Input icon="📅" label="Duration (months)" value={form.duration_months || ""} onChange={up("duration_months")} type="number" required placeholder="12" />
                    <Input icon="🗓️" label="Start Date" value={form.start_date} onChange={up("start_date")} type="date" required placeholder="" />
                  </div>

                  <div style={{ display: "flex", justifyContent: "space-between", marginTop: 16 }}>
                    <AnimatedButton variant="ghost" size="sm" onClick={() => setStep(1)}>← Back</AnimatedButton>
                    <AnimatedButton variant="primary" size="md" onClick={() => setStep(3)} disabled={!ok2}>Next: Review →</AnimatedButton>
                  </div>
                </GlassCard>
              </motion.div>
            )}

            {step === 3 && (
              <motion.div key="s3" initial={{ opacity: 0, x: 24 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -24 }} transition={{ duration: 0.25 }}>
                <GlassCard hover={false} depth={false} style={{ position: "relative", overflow: "hidden" }}>
                  {/* Background glow */}
                  <div style={{ position: "absolute", top: -40, right: -40, width: 180, height: 180, borderRadius: "50%", background: "radial-gradient(circle, rgba(249,115,22,0.05), transparent)", pointerEvents: "none" }} />

                  <h3 style={{ fontSize: 15, fontWeight: 700, color: "var(--color-text)", margin: "0 0 4px", position: "relative" }}>Review & Create</h3>
                  <p style={{ fontSize: 12, color: "var(--color-text-muted)", marginBottom: 16, position: "relative" }}>Verify the details before creating your fund.</p>

                  {/* Pool amount hero */}
                  <div style={{
                    position: "relative", overflow: "hidden",
                    background: "var(--color-bg-subtle)", borderRadius: 10, padding: "20px", textAlign: "center", marginBottom: 16,
                    boxShadow: "inset 0 1px 3px rgba(0,0,0,0.2)",
                  }}>
                    <div style={{ position: "absolute", top: -20, right: -20, width: 120, height: 120, borderRadius: "50%", background: "radial-gradient(circle, rgba(249,115,22,0.06), transparent)", pointerEvents: "none" }} />
                    <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 1 }}>Total Pool Size</span>
                    <p style={{
                      fontSize: 36, fontWeight: 800, margin: "6px 0 4px", letterSpacing: -1.5,
                      background: "var(--gradient-primary)", WebkitBackgroundClip: "text", WebkitTextFillColor: "transparent", backgroundClip: "text",
                    }}>₹{pool.toLocaleString("en-IN")}</p>
                    <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>{form.max_members} members × ₹{form.monthly_contribution.toLocaleString("en-IN")}/mo</span>
                  </div>

                  {/* Details grid */}
                  <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8, marginBottom: 14 }}>
                    {[
                      { icon: "🏷️", k: "Fund Name", v: form.name },
                      { icon: "👥", k: "Members", v: `${form.max_members} max` },
                      { icon: "💰", k: "Monthly", v: `₹${form.monthly_contribution.toLocaleString("en-IN")}` },
                      { icon: "📅", k: "Duration", v: `${form.duration_months} months` },
                      { icon: "🗓️", k: "Start Date", v: form.start_date || "Not set" },
                    ].map(({ icon, k, v }) => (
                      <div key={k} style={{
                        background: "var(--color-bg-subtle)", borderRadius: 6, padding: "10px 12px",
                        boxShadow: "inset 0 1px 3px rgba(0,0,0,0.15)",
                        gridColumn: k === "Fund Name" ? "1 / -1" : undefined,
                      }}>
                        <div style={{ display: "flex", alignItems: "center", gap: 4, marginBottom: 3 }}>
                          <span style={{ fontSize: 11 }}>{icon}</span>
                          <span style={{ fontSize: 9, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 0.5 }}>{k}</span>
                        </div>
                        <p style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text)", margin: 0 }}>{v}</p>
                      </div>
                    ))}
                  </div>

                  {msg && <Alert severity={msg.t} sx={{ mb: 2, borderRadius: "8px", fontSize: "0.8rem", border: "none" }}>{msg.m}</Alert>}

                  <div style={{ display: "flex", justifyContent: "space-between", position: "relative" }}>
                    <AnimatedButton variant="ghost" size="sm" onClick={() => setStep(2)}>← Back</AnimatedButton>
                    <AnimatedButton type="submit" variant="primary" size="lg" disabled={loading}>
                      {loading ? "Creating…" : "🚀 Create Fund"}
                    </AnimatedButton>
                  </div>
                </GlassCard>
              </motion.div>
            )}
          </AnimatePresence>
        </form>
      </div>
    </div>
  );
}

/* ───────────────────────────────────────── */
/* ── Dashboard Shell ──                     */
/* ───────────────────────────────────────── */
export default function DashboardPage() {
  const { isAuthenticated, isLoading } = useAuth();
  const router = useRouter();
  const [tab, setTab] = useState<Tab>("overview");
  const [profileComplete, setProfileComplete] = useState<boolean | null>(null); // null=loading

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const tabParam = params.get("tab");
    if (tabParam === "create" || tabParam === "profile" || tabParam === "overview") {
      setTab(tabParam as Tab);
    }
  }, []);

  useEffect(() => { if (!isLoading && !isAuthenticated) router.push("/"); }, [isAuthenticated, isLoading, router]);

  const checkProfile = async () => {
    if (!isAuthenticated) return;
    try {
      const p = await getProfile();
      const pan = (p as any).profile?.pan || (p as any).pan_number;
      const complete = !!(p.full_name && pan);
      setProfileComplete(complete);
    } catch {
      setProfileComplete(false);
    }
  };

  // Check profile completeness on mount
  useEffect(() => {
    if (isAuthenticated) {
      checkProfile().then(() => {
        if (profileComplete === false) setTab("profile");
      });
    }
  }, [isAuthenticated]);

  if (isLoading) return (
    <div style={{ display: "flex", minHeight: "100vh", alignItems: "center", justifyContent: "center", background: "var(--color-bg)" }}>
      <motion.div animate={{ rotate: 360 }} transition={{ repeat: Infinity, duration: 1, ease: "linear" }}
        style={{ width: 32, height: 32, borderRadius: "50%", border: "2px solid rgba(255,255,255,0.04)", borderTopColor: "var(--color-accent)" }} />
    </div>
  );
  if (!isAuthenticated) return null;

  const tabs: { key: Tab; label: string; icon: string }[] = [
    { key: "overview", label: "Overview", icon: "◻" },
    { key: "profile", label: "Profile", icon: "◇" },
    { key: "create", label: "Create", icon: "+" },
  ];

  return (
    <div style={{ background: "var(--color-bg)", minHeight: "100vh" }}>
      <Navbar />
      <main style={{ maxWidth: 1000, margin: "0 auto", padding: "28px 20px" }}>
        <motion.div initial={{ opacity: 0, y: -8 }} animate={{ opacity: 1, y: 0 }}
          style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 24 }}>
          <div>
            <h1 style={{ fontSize: 24, fontWeight: 800, color: "var(--color-text)", margin: 0, letterSpacing: -0.5 }}>Dashboard</h1>
            <p style={{ fontSize: 12, color: "var(--color-text-muted)", margin: "4px 0 0" }}>Manage funds & profile</p>
          </div>
        </motion.div>

        {profileComplete === false && (
          <motion.div initial={{ opacity: 0, y: -6 }} animate={{ opacity: 1, y: 0 }}
            style={{
              display: "flex", alignItems: "center", gap: 10,
              background: "rgba(245,158,11,0.08)", border: "1px solid rgba(245,158,11,0.2)",
              borderRadius: 8, padding: "10px 16px", marginBottom: 16,
            }}>
            <span style={{ fontSize: 18 }}>⚠️</span>
            <div>
              <p style={{ fontSize: 13, fontWeight: 600, color: "#f59e0b", margin: 0 }}>Profile Incomplete</p>
              <p style={{ fontSize: 11, color: "var(--color-text-muted)", margin: "2px 0 0" }}>
                Please complete your profile and KYC verification to join chit funds.
              </p>
            </div>
            {tab !== "profile" && (
              <button onClick={() => setTab("profile")}
                style={{
                  marginLeft: "auto", padding: "5px 14px", fontSize: 11, fontWeight: 600,
                  background: "rgba(245,158,11,0.15)", border: "1px solid rgba(245,158,11,0.3)",
                  borderRadius: 5, color: "#f59e0b", cursor: "pointer", fontFamily: '"Inter",sans-serif',
                  whiteSpace: "nowrap" as const,
                }}>
                Go to Profile →
              </button>
            )}
          </motion.div>
        )}

        <div style={{
          display: "grid", gridTemplateColumns: "repeat(3, 1fr)", width: 380,
          background: "var(--color-bg-card)", borderRadius: 8, padding: 3, marginBottom: 20,
          boxShadow: "var(--shadow-card)",
        }}>
          {tabs.map((t) => (
            <motion.button key={t.key} onClick={() => setTab(t.key)} whileTap={{ scale: 0.97 }}
              style={{
                position: "relative", padding: "8px 0", fontSize: 13, fontWeight: 600,
                textAlign: "center" as const,
                color: tab === t.key ? "#fff" : "var(--color-text-muted)",
                background: tab === t.key ? "var(--gradient-primary)" : "transparent",
                borderRadius: 6, border: "none", cursor: "pointer", fontFamily: '"Inter",sans-serif',
                boxShadow: tab === t.key ? "0 2px 8px rgba(249,115,22,0.2)" : "none",
                transition: "color 0.2s",
              }}>
              <span style={{ marginRight: 4, opacity: 0.7 }}>{t.icon}</span>{t.label}
            </motion.button>
          ))}
        </div>

        <AnimatePresence mode="wait">
          <motion.div key={tab} initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0 }} transition={{ duration: 0.2 }}>
            {tab === "overview" && <Overview />}
            {tab === "profile" && <Profile onUpdate={checkProfile} />}
            {tab === "create" && <CreateFund />}
          </motion.div>
        </AnimatePresence>
      </main>
    </div>
  );
}
