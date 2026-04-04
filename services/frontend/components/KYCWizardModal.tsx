"use client";

import React, { useState, useEffect, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  getKYCStatus, verifyPAN, fetchKYCHistory, runMLPrediction, type KYCStatus,
} from "@/services/api";

type WizardStep = "loading" | "pan_checking" | "pan_failed" | "credit_result" | "bank_details" | "generating" | "history_preview" | "ml_processing" | "complete" | "error";
interface CreditInfo { hasCibil: boolean; cibilScore: number | null; }
interface TrustResult { score: number; riskBand: string; defaultProbability: number; }
interface HistoryWithCredit { has_history: true; LIMIT_BAL: number; AGE: number; PAY: number[]; BILL_AMT: number[]; PAY_AMT: number[]; }
interface HistoryColdStart { has_history: false; income: number; age: number; employment_years: number; loan_amount: number; loan_percent_income: number; }
type SyntheticHistory = HistoryWithCredit | HistoryColdStart | null;
interface Props { isOpen: boolean; onClose: () => void; onComplete?: (result: TrustResult) => void; }

const BAND_COLORS: Record<string, string> = { Excellent: "#22c55e", Good: "#34d399", Average: "#f59e0b", Risky: "#f97316", "High Risk": "#ef4444" };
function cibilLabel(s: number) { return s >= 750 ? "Excellent" : s >= 650 ? "Good" : s >= 550 ? "Fair" : "Poor"; }
function cibilColor(s: number) { return s >= 750 ? "#22c55e" : s >= 650 ? "#34d399" : s >= 550 ? "#f59e0b" : "#ef4444"; }
function fmtINR(n: number) { return "₹" + Math.round(n).toLocaleString("en-IN"); }
function payLabel(p: number) { return p === -2 ? "No use" : p === -1 ? "Paid" : p === 0 ? "Revolving" : `${p}M delay`; }
function payColor(p: number) { return p <= -1 ? "#22c55e" : p === 0 ? "#f59e0b" : "#ef4444"; }
function stepFromKYCStatus(s: KYCStatus): WizardStep { return s === "pan_verified" ? "bank_details" : s === "credit_fetched" ? "ml_processing" : (s === "verified" || s === "ml_ready") ? "complete" : "pan_checking"; }

// ── Tiny components ─────────────────────────────────────────────────────────

const STEPS: WizardStep[] = ["pan_checking", "credit_result", "bank_details", "history_preview", "complete"];

function StepBar({ current }: { current: WizardStep }) {
  const idx = STEPS.indexOf(current);
  return (
    <div style={{ display: "flex", gap: 4, marginBottom: 24 }}>
      {STEPS.map((_, i) => (
        <div key={i} style={{ flex: 1, height: 2, borderRadius: 1, background: i <= idx ? "var(--color-accent)" : "var(--color-border)", transition: "background 0.3s" }} />
      ))}
    </div>
  );
}

function Spin({ size = 28 }: { size?: number }) {
  return <div style={{ width: size, height: size, borderRadius: "50%", border: "2px solid var(--color-border)", borderTopColor: "var(--color-accent)", animation: "spin 0.7s linear infinite" }} />;
}

function AnimScore({ score, color }: { score: number; color: string }) {
  const [d, setD] = React.useState(0);
  React.useEffect(() => { setD(0); const s = Date.now(); const t = setInterval(() => { const p = Math.min((Date.now() - s) / 1200, 1); setD(Math.round((1 - Math.pow(1 - p, 3)) * score)); if (p >= 1) clearInterval(t); }, 16); return () => clearInterval(t); }, [score]);
  return <motion.span initial={{ opacity: 0, scale: 0.8 }} animate={{ opacity: 1, scale: 1 }} transition={{ delay: 0.2, type: "spring" }} style={{ fontSize: 52, fontWeight: 800, color, lineHeight: 1, letterSpacing: -2 }}>{d}</motion.span>;
}

function Btn({ children, onClick, type = "button", v = "primary" }: { children: React.ReactNode; onClick?: () => void; type?: "button" | "submit"; v?: "primary" | "outline" | "ghost" }) {
  const s: Record<string, React.CSSProperties> = {
    primary: { background: "var(--gradient-primary)", color: "#fff", border: "none" },
    outline: { background: "transparent", color: "var(--color-accent)", border: "1px solid var(--color-accent)" },
    ghost: { background: "transparent", color: "var(--color-text-muted)", border: "1px solid var(--color-border)" },
  };
  return <button type={type} onClick={onClick} style={{ ...s[v], width: "100%", padding: "10px 0", borderRadius: 6, fontSize: 13, fontWeight: 600, cursor: "pointer", fontFamily: '"Inter",sans-serif', transition: "opacity 0.2s" }}>{children}</button>;
}

// ── History ──────────────────────────────────────────────────────────────────

function HistCredit({ h }: { h: HistoryWithCredit }) {
  return (
    <div>
      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8, marginBottom: 12 }}>
        <div style={{ background: "var(--color-bg)", borderRadius: 6, padding: "8px 12px", border: "1px solid var(--color-border)" }}>
          <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>Limit</span>
          <p style={{ fontSize: 14, fontWeight: 600, color: "var(--color-text)", margin: 0 }}>{fmtINR(h.LIMIT_BAL)}</p>
        </div>
        <div style={{ background: "var(--color-bg)", borderRadius: 6, padding: "8px 12px", border: "1px solid var(--color-border)" }}>
          <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>Age</span>
          <p style={{ fontSize: 14, fontWeight: 600, color: "var(--color-text)", margin: 0 }}>{h.AGE} yrs</p>
        </div>
      </div>
      <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 12 }}>
        <thead><tr style={{ borderBottom: "1px solid var(--color-border)" }}>
          {["Mo", "Status", "Bill", "Paid"].map((c) => <th key={c} style={{ textAlign: "left", color: "var(--color-text-muted)", fontWeight: 600, fontSize: 10, padding: "0 4px 6px 0", textTransform: "uppercase", letterSpacing: 0.5 }}>{c}</th>)}
        </tr></thead>
        <tbody>{["M1", "M2", "M3", "M4", "M5", "M6"].map((m, i) => (
          <tr key={m} style={{ borderBottom: "1px solid var(--color-border)" }}>
            <td style={{ padding: "5px 4px 5px 0", color: "var(--color-text-secondary)" }}>{m}</td>
            <td style={{ padding: "5px 4px 5px 0", color: payColor(h.PAY[i]), fontWeight: 600 }}>{payLabel(h.PAY[i])}</td>
            <td style={{ padding: "5px 4px 5px 0", color: "var(--color-text-secondary)" }}>{fmtINR(h.BILL_AMT[i])}</td>
            <td style={{ padding: "5px 4px 5px 0", color: h.PAY_AMT[i] >= h.BILL_AMT[i] ? "var(--color-success)" : "var(--color-text-secondary)" }}>{fmtINR(h.PAY_AMT[i])}</td>
          </tr>
        ))}</tbody>
      </table>
    </div>
  );
}

function HistCold({ h }: { h: HistoryColdStart }) {
  return <div>{[
    { l: "Income", v: fmtINR(h.income) }, { l: "Age", v: `${h.age}` }, { l: "Employment", v: `${h.employment_years}yr` },
    { l: "Loan", v: fmtINR(h.loan_amount) }, { l: "Loan/Income", v: `${(h.loan_percent_income * 100).toFixed(1)}%` },
  ].map(({ l, v }) => (
    <div key={l} style={{ display: "flex", justifyContent: "space-between", padding: "7px 0", borderBottom: "1px solid var(--color-border)" }}>
      <span style={{ fontSize: 12, color: "var(--color-text-muted)" }}>{l}</span>
      <span style={{ fontSize: 12, fontWeight: 600, color: "var(--color-text)" }}>{v}</span>
    </div>
  ))}</div>;
}

// ── Main ─────────────────────────────────────────────────────────────────────

export default function KYCWizardModal({ isOpen, onClose, onComplete }: Props) {
  const [step, setStep] = useState<WizardStep>("loading");
  const [credit, setCredit] = useState<CreditInfo | null>(null);
  const [synHist, setSynHist] = useState<SyntheticHistory>(null);
  const [result, setResult] = useState<TrustResult | null>(null);
  const [bankAcc, setBankAcc] = useState("");
  const [ifsc, setIfsc] = useState("");
  const [fErr, setFErr] = useState<{ f: string; m: string } | null>(null);
  const [errMsg, setErrMsg] = useState("");

  const doPAN = useCallback(async () => {
    setStep("pan_checking");
    try {
      const r = await verifyPAN();
      if (!r.pan_verified) { setStep("pan_failed"); return; }
      setCredit({ hasCibil: r.has_cibil, cibilScore: r.cibil_score ?? null });
      if (r.skipped && r.kyc_status !== "pan_verified") setStep(stepFromKYCStatus(r.kyc_status));
      else setStep("credit_result");
    } catch (e: unknown) { setErrMsg(e instanceof Error ? e.message : "PAN check failed"); setStep("error"); }
  }, []);

  const doHist = useCallback(async () => {
    setStep("generating");
    try {
      const r = await fetchKYCHistory(bankAcc, ifsc.toUpperCase());
      if (r.history) setSynHist(r.history as unknown as SyntheticHistory);
      setStep("history_preview");
    } catch (e: unknown) {
      const m = e instanceof Error ? e.message : "Failed";
      if (m.toLowerCase().includes("bank") || m.toLowerCase().includes("ifsc")) { setFErr({ f: m.toLowerCase().includes("ifsc") ? "ifsc" : "acc", m }); setStep("bank_details"); }
      else { setErrMsg(m); setStep("error"); }
    }
  }, [bankAcc, ifsc]);

  const doML = useCallback(async () => {
    setStep("ml_processing");
    try {
      const r = await runMLPrediction();
      const res: TrustResult = { score: r.score, riskBand: r.risk_band, defaultProbability: r.default_probability };
      setResult(res); setStep("complete"); onComplete?.(res);
    } catch (e: unknown) { setErrMsg(e instanceof Error ? e.message : "ML failed"); setStep("error"); }
  }, [onComplete]);

  useEffect(() => {
    if (!isOpen) return;
    setStep("loading"); setErrMsg(""); setFErr(null); setSynHist(null);
    (async () => {
      try {
        const s = await getKYCStatus();
        if (s.kyc_status === "verified" || s.kyc_status === "ml_ready") { setResult({ score: s.trust_score, riskBand: s.risk_band, defaultProbability: s.default_probability }); setCredit({ hasCibil: s.has_cibil, cibilScore: s.cibil_score ?? null }); setStep("complete"); return; }
        if (s.kyc_status === "credit_fetched") { setCredit({ hasCibil: s.has_cibil, cibilScore: s.cibil_score ?? null }); await doML(); return; }
        if (s.kyc_status === "pan_verified") { setCredit({ hasCibil: s.has_cibil, cibilScore: s.cibil_score ?? null }); setStep("bank_details"); return; }
        await doPAN();
      } catch { await doPAN(); }
    })();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen]);

  function onBank(e: React.FormEvent) {
    e.preventDefault(); setFErr(null);
    const a = bankAcc.trim(), c = ifsc.trim().toUpperCase();
    if (!/^[0-9]{9,18}$/.test(a)) { setFErr({ f: "acc", m: "9–18 digits required" }); return; }
    if (!/^[A-Z]{4}0[A-Z0-9]{6}$/.test(c)) { setFErr({ f: "ifsc", m: "Invalid IFSC" }); return; }
    doHist();
  }

  if (!isOpen) return null;

  const back: Partial<Record<WizardStep, WizardStep>> = { bank_details: "credit_result", history_preview: "bank_details" };

  return (
    <>
      <div onClick={onClose} style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.7)", zIndex: 1000 }} />
      <div style={{ position: "fixed", inset: 0, display: "flex", alignItems: "center", justifyContent: "center", zIndex: 1001, padding: 16, pointerEvents: "none" }}>
        <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.2 }}
          onClick={(e) => e.stopPropagation()}
          style={{
            pointerEvents: "auto", width: "100%", maxWidth: step === "history_preview" ? 480 : 420,
            background: "var(--color-bg-card)", border: "1px solid var(--color-border)", borderRadius: 8,
            boxShadow: "0 24px 64px rgba(0,0,0,0.6)", overflow: "hidden",
          }}>

          {/* Header */}
          <div style={{ padding: "12px 16px", borderBottom: "1px solid var(--color-border)", display: "flex", alignItems: "center", justifyContent: "space-between" }}>
            <div>
              <span style={{ fontSize: 10, fontWeight: 600, color: "var(--color-accent)", letterSpacing: 1, textTransform: "uppercase" }}>KYC Verification</span>
              <h3 style={{ fontSize: 14, fontWeight: 700, color: "var(--color-text)", margin: 0 }}>Trust Score Setup</h3>
            </div>
            <button onClick={onClose} style={{ width: 24, height: 24, borderRadius: 4, border: "1px solid var(--color-border)", background: "transparent", color: "var(--color-text-muted)", fontSize: 14, cursor: "pointer", display: "flex", alignItems: "center", justifyContent: "center" }}>×</button>
          </div>

          {/* Body */}
          <div style={{ padding: "16px 20px 20px" }}>
            {!["loading", "error", "pan_failed"].includes(step) && <StepBar current={step} />}

            <AnimatePresence mode="wait">
              <motion.div key={step} initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0 }} transition={{ duration: 0.15 }}>

                {back[step] && <button onClick={() => setStep(back[step]!)} style={{ background: "none", border: "none", color: "var(--color-text-muted)", fontSize: 12, fontWeight: 600, cursor: "pointer", padding: "0 0 12px", fontFamily: '"Inter",sans-serif' }}>← Back</button>}

                {/* Loading */}
                {step === "loading" && <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 10, padding: "24px 0" }}><Spin /><p style={{ fontSize: 13, color: "var(--color-text-secondary)", margin: 0 }}>Checking KYC status…</p></div>}

                {/* PAN Checking */}
                {step === "pan_checking" && <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 10, padding: "24px 0" }}><Spin size={32} /><p style={{ fontSize: 14, fontWeight: 600, color: "var(--color-text)", margin: 0 }}>Verifying PAN</p><p style={{ fontSize: 12, color: "var(--color-text-muted)", margin: 0 }}>Querying credit bureau…</p></div>}

                {/* PAN Failed */}
                {step === "pan_failed" && <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 8, padding: "16px 0" }}>
                  <div style={{ width: 36, height: 36, borderRadius: 6, background: "var(--color-danger-light)", border: "1px solid rgba(239,68,68,0.2)", display: "flex", alignItems: "center", justifyContent: "center", color: "var(--color-danger)", fontWeight: 700 }}>✕</div>
                  <p style={{ fontSize: 14, fontWeight: 600, color: "var(--color-danger)", margin: 0 }}>PAN Verification Failed</p>
                  <p style={{ fontSize: 12, color: "var(--color-text-muted)", textAlign: "center", margin: 0 }}>Check your PAN number in profile and retry.</p>
                  <div style={{ width: "100%", marginTop: 8, display: "flex", flexDirection: "column", gap: 6 }}>
                    <Btn v="outline" onClick={doPAN}>Retry</Btn>
                    <Btn v="ghost" onClick={onClose}>Close</Btn>
                  </div>
                </div>}

                {/* Credit Result */}
                {step === "credit_result" && credit && <div>
                  <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 16 }}>
                    <div style={{ width: 28, height: 28, borderRadius: 6, background: "rgba(34,197,94,0.1)", border: "1px solid rgba(34,197,94,0.2)", display: "flex", alignItems: "center", justifyContent: "center", color: "#22c55e", fontSize: 14, fontWeight: 700 }}>✓</div>
                    <span style={{ fontSize: 14, fontWeight: 600, color: "var(--color-text)" }}>PAN Verified</span>
                  </div>
                  {credit.hasCibil && credit.cibilScore !== null ? (
                    <div style={{ background: "var(--color-bg)", border: "1px solid var(--color-border)", borderRadius: 6, padding: "14px 16px", marginBottom: 16 }}>
                      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                        <span style={{ fontSize: 11, color: "var(--color-text-muted)", fontWeight: 600, textTransform: "uppercase", letterSpacing: 0.5 }}>CIBIL Score</span>
                        <span style={{ fontSize: 11, color: cibilColor(credit.cibilScore), fontWeight: 600 }}>{cibilLabel(credit.cibilScore)}</span>
                      </div>
                      <p style={{ fontSize: 32, fontWeight: 800, color: cibilColor(credit.cibilScore), margin: "8px 0 0", letterSpacing: -1 }}>{credit.cibilScore}<span style={{ fontSize: 12, fontWeight: 500, color: "var(--color-text-muted)" }}> / 900</span></p>
                    </div>
                  ) : (
                    <div style={{ background: "var(--color-bg)", border: "1px solid rgba(245,158,11,0.15)", borderRadius: 6, padding: "10px 12px", marginBottom: 16, display: "flex", alignItems: "center", gap: 8 }}>
                      <span style={{ fontSize: 16 }}>⚠️</span>
                      <div><p style={{ fontSize: 13, fontWeight: 600, color: "#f59e0b", margin: 0 }}>No Credit History</p><p style={{ fontSize: 11, color: "var(--color-text-muted)", margin: 0 }}>We&apos;ll generate a profile from bank data</p></div>
                    </div>
                  )}
                  <p style={{ fontSize: 12, color: "var(--color-text-muted)", marginBottom: 12 }}>Provide bank details to complete scoring.</p>
                  <Btn onClick={() => setStep("bank_details")}>Continue to Bank Details →</Btn>
                </div>}

                {/* Bank Details */}
                {step === "bank_details" && <form onSubmit={onBank}>
                  <p style={{ fontSize: 14, fontWeight: 600, color: "var(--color-text)", marginBottom: 2 }}>Bank Account</p>
                  <p style={{ fontSize: 12, color: "var(--color-text-muted)", marginBottom: 14 }}>Used only for synthetic scoring — never stored.</p>
                  <label style={{ display: "block", fontSize: 11, fontWeight: 600, color: "var(--color-text-secondary)", letterSpacing: 0.5, marginBottom: 4 }}>Account Number</label>
                  <input id="kyc-bank" type="text" inputMode="numeric" placeholder="9–18 digits" value={bankAcc} onChange={(e) => { setBankAcc(e.target.value.replace(/\D/g, "")); setFErr(null); }} className="glow-input" autoComplete="off"
                    style={{ marginBottom: fErr?.f === "acc" ? 0 : 12, borderColor: fErr?.f === "acc" ? "var(--color-danger)" : undefined }} />
                  {fErr?.f === "acc" && <p style={{ color: "var(--color-danger)", fontSize: 11, margin: "3px 0 10px" }}>{fErr.m}</p>}
                  <label style={{ display: "block", fontSize: 11, fontWeight: 600, color: "var(--color-text-secondary)", letterSpacing: 0.5, marginBottom: 4 }}>IFSC Code</label>
                  <input id="kyc-ifsc" type="text" placeholder="e.g. HDFC0001234" value={ifsc} onChange={(e) => { setIfsc(e.target.value.toUpperCase()); setFErr(null); }} maxLength={11} className="glow-input" autoComplete="off"
                    style={{ marginBottom: fErr?.f === "ifsc" ? 0 : 16, borderColor: fErr?.f === "ifsc" ? "var(--color-danger)" : undefined }} />
                  {fErr?.f === "ifsc" && <p style={{ color: "var(--color-danger)", fontSize: 11, margin: "3px 0 14px" }}>{fErr.m}</p>}
                  <Btn type="submit">Generate History →</Btn>
                </form>}

                {/* Generating */}
                {step === "generating" && <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 8, padding: "24px 0" }}><Spin size={32} /><p style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text)", margin: 0 }}>Generating history…</p><p style={{ fontSize: 12, color: "var(--color-text-muted)", margin: 0 }}>AI analysis in progress</p></div>}

                {/* History Preview */}
                {step === "history_preview" && <div>
                  <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 12 }}>
                    <span style={{ fontSize: 16 }}>📊</span>
                    <div><p style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text)", margin: 0 }}>Synthetic History</p><p style={{ fontSize: 11, color: "var(--color-text-muted)", margin: 0 }}>AI-generated for ML scoring</p></div>
                  </div>
                  <div style={{ background: "var(--color-bg)", border: "1px solid var(--color-border)", borderRadius: 6, padding: "12px 14px", marginBottom: 12 }}>
                    {synHist ? (synHist.has_history ? <HistCredit h={synHist as HistoryWithCredit} /> : <HistCold h={synHist as HistoryColdStart} />) : <p style={{ fontSize: 12, color: "var(--color-text-muted)", textAlign: "center", margin: 0 }}>Ready to calculate.</p>}
                  </div>
                  <div style={{ background: "var(--color-bg)", border: "1px solid var(--color-border)", borderRadius: 6, padding: "8px 10px", marginBottom: 14, display: "flex", gap: 6 }}>
                    <span style={{ fontSize: 12, flexShrink: 0 }}>ℹ️</span>
                    <p style={{ fontSize: 11, color: "var(--color-text-muted)", margin: 0, lineHeight: 1.5 }}>Synthetically generated. Used only for score calculation.</p>
                  </div>
                  <Btn onClick={doML}>Calculate Trust Score →</Btn>
                </div>}

                {/* ML Processing */}
                {step === "ml_processing" && <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 8, padding: "24px 0" }}><Spin size={32} /><p style={{ fontSize: 13, fontWeight: 600, color: "var(--color-text)", margin: 0 }}>Running ML Model…</p><p style={{ fontSize: 12, color: "var(--color-text-muted)", margin: 0 }}>Calculating trust score</p></div>}

                {/* Complete */}
                {step === "complete" && result && <div style={{ textAlign: "center" }}>
                  <div style={{ width: 32, height: 32, borderRadius: 6, background: "rgba(34,197,94,0.1)", border: "1px solid rgba(34,197,94,0.2)", display: "inline-flex", alignItems: "center", justifyContent: "center", color: "#22c55e", fontSize: 16, fontWeight: 700 }}>✓</div>
                  <p style={{ fontSize: 16, fontWeight: 700, color: "var(--color-text)", margin: "10px 0 2px" }}>Score Ready</p>
                  <p style={{ fontSize: 12, color: "var(--color-text-muted)", marginBottom: 16 }}>Your trust score has been calculated</p>

                  <div style={{ background: "var(--color-bg)", border: "1px solid var(--color-border)", borderRadius: 8, padding: "20px 16px", marginBottom: 14 }}>
                    <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", letterSpacing: 1, textTransform: "uppercase" }}>TRUST SCORE</span>
                    <div style={{ margin: "10px 0" }}><AnimScore score={result.score} color={BAND_COLORS[result.riskBand] || "var(--color-accent)"} /></div>
                    <span style={{ display: "inline-flex", alignItems: "center", gap: 5, fontSize: 12, fontWeight: 600, color: BAND_COLORS[result.riskBand], background: `${BAND_COLORS[result.riskBand]}12`, border: `1px solid ${BAND_COLORS[result.riskBand]}25`, borderRadius: 4, padding: "3px 10px" }}>
                      <span style={{ width: 5, height: 5, borderRadius: "50%", background: BAND_COLORS[result.riskBand] }} />{result.riskBand}
                    </span>
                    <p style={{ fontSize: 11, color: "var(--color-text-muted)", marginTop: 10, marginBottom: 0 }}>Default probability: {(result.defaultProbability * 100).toFixed(1)}%</p>
                  </div>

                  {credit?.hasCibil && credit.cibilScore && <p style={{ fontSize: 11, color: "var(--color-text-muted)", marginBottom: 8 }}>CIBIL: <span style={{ color: cibilColor(credit.cibilScore), fontWeight: 600 }}>{credit.cibilScore}</span></p>}

                  <div style={{ textAlign: "left", marginBottom: 14 }}>
                    {["Based on credit & financial profile", "Determines fund eligibility", "Updated with new data"].map((t, i) => (
                      <motion.div key={t} initial={{ opacity: 0, x: -4 }} animate={{ opacity: 1, x: 0 }} transition={{ delay: 0.8 + i * 0.08 }} style={{ display: "flex", alignItems: "center", gap: 6, padding: "3px 0" }}>
                        <span style={{ width: 3, height: 3, borderRadius: "50%", background: "var(--color-accent)", flexShrink: 0 }} />
                        <span style={{ fontSize: 11, color: "var(--color-text-muted)" }}>{t}</span>
                      </motion.div>
                    ))}
                  </div>
                  <Btn onClick={onClose}>Go to Dashboard →</Btn>
                </div>}

                {/* Error */}
                {step === "error" && <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 8, padding: "16px 0" }}>
                  <div style={{ width: 36, height: 36, borderRadius: 6, background: "var(--color-danger-light)", border: "1px solid rgba(239,68,68,0.2)", display: "flex", alignItems: "center", justifyContent: "center", color: "var(--color-danger)", fontWeight: 700 }}>✕</div>
                  <p style={{ fontSize: 14, fontWeight: 600, color: "var(--color-danger)", margin: 0 }}>Error</p>
                  <p style={{ fontSize: 12, color: "var(--color-text-muted)", textAlign: "center", margin: 0 }}>{errMsg || "Something went wrong."}</p>
                  <div style={{ width: "100%", marginTop: 8, display: "flex", flexDirection: "column", gap: 6 }}>
                    <Btn v="outline" onClick={() => doPAN()}>Start Over</Btn>
                    <Btn v="ghost" onClick={onClose}>Close</Btn>
                  </div>
                </div>}

              </motion.div>
            </AnimatePresence>
          </div>
        </motion.div>
      </div>
      <style>{`@keyframes spin{to{transform:rotate(360deg)}}@keyframes pulse{0%,100%{opacity:.3}50%{opacity:1}}`}</style>
    </>
  );
}
