"use client";

import React, { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import Navbar from "@/components/Navbar";
import GlassCard from "@/components/ui/GlassCard";
import AnimatedButton from "@/components/ui/AnimatedButton";
import StatCard from "@/components/ui/StatCard";
import AuctionRulebook from "@/components/AuctionRulebook";
import ChatPanel from "@/components/ChatPanel";
import { useAuctionSocket } from "@/hooks/useAuctionSocket";
import {
  applyToFund,
  getFundDetails,
  getFundMembers,
  approveFundMember,
  rejectFundMember,
  getFundApplicationStatus,
  getAuction,
  startAuction as apiStartAuction,
  getFundContributions,
  getCurrentUserId,
  getProfile,
  type ApplyResult,
  type FundDetails,
  type FundMember,
  type ProfileData,
  type AuctionSnapshot,
  type CurrentCycleContributions,
  type FundApplicationStatus,
} from "@/services/api";

type ProfileWithContainer = ProfileData & {
  profile?: {
    pan?: string;
    pan_number?: string;
  };
};

function fmt(n: number) {
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 0,
  }).format(n);
}

function statusStyle(s?: string) {
  switch (s) {
    case "open":
      return { color: "#f97316", bg: "rgba(249,115,22,0.08)", label: "Open" };
    case "active":
      return { color: "#22c55e", bg: "rgba(34,197,94,0.08)", label: "Active" };
    case "completed":
      return {
        color: "#60a5fa",
        bg: "rgba(96,165,250,0.08)",
        label: "Completed",
      };
    default:
      return {
        color: "var(--color-text-muted)",
        bg: "var(--color-bg-subtle)",
        label: s || "—",
      };
  }
}

function contribStatusBadge(status: string) {
  switch (status) {
    case "paid":
      return { color: "#22c55e", bg: "rgba(34,197,94,0.08)", label: "Paid" };
    case "pending":
      return {
        color: "#f59e0b",
        bg: "rgba(245,158,11,0.08)",
        label: "Pending",
      };
    case "overdue":
      return { color: "#ef4444", bg: "rgba(239,68,68,0.08)", label: "Overdue" };
    default:
      return {
        color: "var(--color-text-muted)",
        bg: "var(--color-bg-subtle)",
        label: status,
      };
  }
}

/** Derive "active" status: members full AND start_date has passed (compare in LOCAL date) */
function derivedStatus(fund: FundDetails): string {
  if (fund.status === "open") {
    const full = fund.current_member_count >= fund.max_members;
    if (full) {
      if (!fund.start_date) return "active";
      // Compare calendar dates in local timezone to avoid UTC-midnight vs IST issues
      const startDateLocal = new Date(fund.start_date);
      const todayLocal = new Date();
      const startMidnight = new Date(
        startDateLocal.getFullYear(),
        startDateLocal.getMonth(),
        startDateLocal.getDate(),
      );
      const todayMidnight = new Date(
        todayLocal.getFullYear(),
        todayLocal.getMonth(),
        todayLocal.getDate(),
      );
      if (startMidnight <= todayMidnight) return "active";
    }
  }
  return fund.status;
}

export default function FundDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;

  const [fund, setFund] = useState<FundDetails | null>(null);
  const [members, setMembers] = useState<FundMember[]>([]);
  const [isCreator, setIsCreator] = useState(false);
  const [contribData, setContribData] =
    useState<CurrentCycleContributions | null>(null);
  const [auctionSnap, setAuctionSnap] = useState<AuctionSnapshot | null>(null);
  const [currentUserId, setCurrentUserId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [applyLoading, setApplyLoading] = useState(false);
  const [startAuctionLoading, setStartAuctionLoading] = useState(false);
  const [applyResult, setApplyResult] = useState<ApplyResult | null>(null);
  const [applicationStatus, setApplicationStatus] =
    useState<FundApplicationStatus["status"]>("none");
  const [error, setError] = useState<string | null>(null);
  const [actionMessage, setActionMessage] = useState<string | null>(null);
  const [showRulebook, setShowRulebook] = useState(false);
  const [rulebookAction, setRulebookAction] = useState<
    "start-auction" | "enter-room" | null
  >(null);
  const [profileIncomplete, setProfileIncomplete] = useState(false);
  const [showProfilePopup, setShowProfilePopup] = useState(false);

  const { lastMessage } = useAuctionSocket(id);
  const isMember = members.some(
    (m) => m.user_id === currentUserId && m.status === "active",
  );
  const isUserPending = applicationStatus === "pending";
  const isUserActive = applicationStatus === "active" || isMember;
  const isUserRejected = applicationStatus === "rejected";
  useEffect(() => {
    if (!id) return;
    // Set current user id synchronously from the token before any async work
    setCurrentUserId(getCurrentUserId());
    (async () => {
      try {
        const [f, membersResult] = await Promise.all([
          getFundDetails(id),
          getFundMembers(id).catch(() => null), // null = not a fund member
        ]);
        setFund(f);
        if (membersResult !== null) {
          setMembers(membersResult);
          // Check if current user is already an active member
          const uid = getCurrentUserId();
          const userIsAlreadyActive = membersResult.some(
            (m) => m.user_id === uid && m.status === "active",
          );
          if (userIsAlreadyActive) {
            setApplyResult(null); // Clear application card if already a member
          }
        }

        const statusResult = await getFundApplicationStatus(id).catch(() => ({
          status: "none" as const,
        }));
        setApplicationStatus(statusResult.status);
        // Determine if current user is the creator
        const uid = getCurrentUserId();
        if (f && uid && f.creator_id === uid) {
          setIsCreator(true);
        }

        // Fetch contributions and auction in parallel (may fail if user isn't a member)
        const [c, a] = await Promise.allSettled([
          getFundContributions(id),
          getAuction(id),
        ]);
        if (c.status === "fulfilled") setContribData(c.value);
        if (a.status === "fulfilled") setAuctionSnap(a.value);
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "Failed to load fund.");
      } finally {
        setLoading(false);
      }
    })();
  }, [id]);

  // ── Polling: re-fetch members + contributions every 10s ──
  useEffect(() => {
    if (!id || !currentUserId) return;
    const interval = setInterval(async () => {
      try {
        const [membersResult, c, a] = await Promise.allSettled([
          getFundMembers(id),
          getFundContributions(id),
          getAuction(id),
        ]);
        if (
          membersResult.status === "fulfilled" &&
          Array.isArray(membersResult.value)
        ) {
          setMembers(membersResult.value);
          // If the current user is now an active member, clear the apply result
          const userIsNowActive = membersResult.value.some(
            (m) => m.user_id === currentUserId && m.status === "active",
          );
          if (userIsNowActive) {
            setApplyResult(null);
          }
        }
        if (id) {
          const statusResult = await getFundApplicationStatus(id).catch(() => ({
            status: "none" as const,
          }));
          setApplicationStatus(statusResult.status);
        }
        if (c.status === "fulfilled") setContribData(c.value);
        if (a.status === "fulfilled") setAuctionSnap(a.value);
      } catch {
        /* ignore polling errors */
      }
    }, 10000);
    return () => clearInterval(interval);
  }, [id, currentUserId]);

  useEffect(() => {
    if (!id || !lastMessage) return;

    const shouldRefreshAuction =
      lastMessage.type === "auction_started" ||
      lastMessage.type === "bidding_started" ||
      lastMessage.type === "auction_ended";

    if (!shouldRefreshAuction) return;

    void getAuction(id)
      .then((next) => setAuctionSnap(next))
      .catch(() => {
        // polling will eventually heal transient failures
      });

    if (lastMessage.type === "auction_started") {
      router.refresh();
    }
  }, [id, lastMessage, router]);

  // Monitor isMember and clear applyResult when user becomes active
  useEffect(() => {
    if (isMember && applyResult) {
      setApplyResult(null);
    }
  }, [isMember, applyResult]);

  // Check profile completeness on mount
  useEffect(() => {
    (async () => {
      try {
        const p = (await getProfile()) as ProfileWithContainer;
        const pan = p.profile?.pan || p.profile?.pan_number || p.pan_number;
        setProfileIncomplete(!p.full_name || !pan);
      } catch {
        setProfileIncomplete(true);
      }
    })();
  }, []);

  const handleApply = async () => {
    if (!id) return;
    // Check profile first
    if (profileIncomplete) {
      setShowProfilePopup(true);
      return;
    }
    setApplyLoading(true);
    setError(null);
    try {
      const res = await applyToFund(id);
      setApplyResult(res);
      setApplicationStatus(res.status as FundApplicationStatus["status"]);
      setActionMessage(res.message || "Application submitted!");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to apply");
    } finally {
      setApplyLoading(false);
    }
  };

  const handleApprove = async (memberId: string) => {
    if (!id) return;
    try {
      await approveFundMember(id, memberId);
      const m = await getFundMembers(id).catch(() => members);
      setMembers(m);
      // Clear the apply result if the current user is the one being approved
      if (memberId === currentUserId) {
        setApplyResult(null);
        setApplicationStatus("active");
      }
      setActionMessage("Member approved!");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to approve");
    }
  };

  const handleReject = async (memberId: string) => {
    if (!id) return;
    try {
      await rejectFundMember(id, memberId);
      const m = await getFundMembers(id).catch(() => members);
      setMembers(m);
      if (memberId === currentUserId) {
        setApplicationStatus("rejected");
      }
      setActionMessage("Member rejected!");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to reject");
    }
  };

  const handleRulebookEnter = async () => {
    if (!id) return;
    setShowRulebook(false);

    if (rulebookAction !== "start-auction") {
      setRulebookAction(null);
      router.push(`/auction/${id}`);
      return;
    }

    setStartAuctionLoading(true);
    try {
      await apiStartAuction(id);
      setRulebookAction(null);
      router.push(`/auction/${id}`);
    } catch (err: unknown) {
      setRulebookAction(null);
      setError(err instanceof Error ? err.message : "Failed to start auction");
    } finally {
      setStartAuctionLoading(false);
    }
  };

  const openStartAuctionRulebook = () => {
    setRulebookAction("start-auction");
    setShowRulebook(true);
  };

  const openEnterRoomRulebook = () => {
    setRulebookAction("enter-room");
    setShowRulebook(true);
  };

  useEffect(() => {
    if (actionMessage) {
      const t = setTimeout(() => setActionMessage(null), 5000);
      return () => clearTimeout(t);
    }
  }, [actionMessage]);

  if (loading)
    return (
      <div style={{ background: "var(--color-bg)", minHeight: "100vh" }}>
        <Navbar />
        <div
          style={{
            display: "flex",
            minHeight: "60vh",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          <motion.div
            animate={{ rotate: 360 }}
            transition={{ repeat: Infinity, duration: 1, ease: "linear" }}
            style={{
              width: 32,
              height: 32,
              borderRadius: "50%",
              border: "2px solid rgba(255,255,255,0.04)",
              borderTopColor: "var(--color-accent)",
            }}
          />
        </div>
      </div>
    );

  if (error && !fund)
    return (
      <div style={{ background: "var(--color-bg)", minHeight: "100vh" }}>
        <Navbar />
        <div
          style={{ maxWidth: 500, margin: "60px auto 0", padding: "0 20px" }}
        >
          <motion.div
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            style={{
              background: "var(--color-danger-light)",
              borderRadius: 8,
              padding: "14px 16px",
              fontSize: 13,
              color: "#f87171",
              boxShadow: "var(--shadow-card)",
            }}
          >
            {error}
          </motion.div>
        </div>
      </div>
    );

  const status = fund ? derivedStatus(fund) : "open";
  const st = statusStyle(status);
  const auctionIsWaiting = auctionSnap?.session?.status === "waiting";
  const auctionIsLive = auctionSnap?.session?.status === "live";
  const auctionRoomOpen = auctionIsWaiting || auctionIsLive;
  const auctionHasResult = !!auctionSnap?.result;
  const allContributionsPaid =
    !!contribData &&
    contribData.contributions.length > 0 &&
    contribData.contributions.every((c) => c.status === "paid");

  return (
    <div
      style={{
        background: "var(--color-bg)",
        minHeight: "100vh",
        color: "var(--color-text)",
      }}
    >
      {/* Background orb */}
      <div
        style={{
          position: "fixed",
          inset: 0,
          pointerEvents: "none",
          zIndex: 0,
        }}
      >
        <div
          style={{
            position: "absolute",
            top: "5%",
            left: "20%",
            width: 400,
            height: 400,
            borderRadius: "50%",
            background:
              "radial-gradient(circle, rgba(249,115,22,0.03), transparent)",
            filter: "blur(80px)",
          }}
        />
      </div>

      <Navbar />

      <main
        style={{
          position: "relative",
          zIndex: 1,
          maxWidth: 1100,
          margin: "0 auto",
          padding: "28px 20px",
        }}
      >
        {/* Action message */}
        <AnimatePresence>
          {actionMessage && (
            <motion.div
              initial={{ opacity: 0, y: -12 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0 }}
              style={{
                marginBottom: 16,
                background: "rgba(34,197,94,0.08)",
                borderRadius: 8,
                padding: "10px 14px",
                fontSize: 13,
                fontWeight: 500,
                color: "#34d399",
                boxShadow: "var(--shadow-card)",
              }}
            >
              ✓ {actionMessage}
            </motion.div>
          )}
        </AnimatePresence>

        {/* Error banner */}
        <AnimatePresence>
          {error && fund && (
            <motion.div
              initial={{ opacity: 0, y: -8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0 }}
              style={{
                marginBottom: 16,
                background: "var(--color-danger-light)",
                borderRadius: 8,
                padding: "10px 14px",
                fontSize: 13,
                color: "#f87171",
                boxShadow: "var(--shadow-card)",
              }}
            >
              {error}
            </motion.div>
          )}
        </AnimatePresence>

        <div style={{ display: "flex", flexWrap: "wrap", gap: 20 }}>
          {/* ── Main Column ── */}
          <motion.div
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
            style={{ flex: "1 1 500px" }}
          >
            {/* Header */}
            <div
              style={{
                marginBottom: 24,
                paddingBottom: 20,
                borderBottom: "1px solid rgba(255,255,255,0.04)",
              }}
            >
              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "flex-start",
                  gap: 12,
                  flexWrap: "wrap",
                }}
              >
                <div>
                  <motion.h1
                    initial={{ opacity: 0, y: 10, filter: "blur(4px)" }}
                    animate={{ opacity: 1, y: 0, filter: "blur(0)" }}
                    transition={{ duration: 0.5, delay: 0.1 }}
                    style={{
                      fontSize: 28,
                      fontWeight: 800,
                      color: "var(--color-text)",
                      letterSpacing: "-0.02em",
                      margin: 0,
                    }}
                  >
                    {fund?.name || "Unnamed Fund"}
                  </motion.h1>
                  <motion.p
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ delay: 0.3 }}
                    style={{
                      fontSize: 11,
                      fontFamily: "monospace",
                      color: "var(--color-text-muted)",
                      marginTop: 6,
                    }}
                  >
                    ID: {id}
                  </motion.p>
                </div>
                <motion.span
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ delay: 0.2 }}
                  style={{
                    display: "inline-flex",
                    alignItems: "center",
                    gap: 5,
                    fontSize: 11,
                    fontWeight: 700,
                    textTransform: "uppercase",
                    letterSpacing: 0.8,
                    color: st.color,
                    background: st.bg,
                    borderRadius: 6,
                    padding: "5px 12px",
                  }}
                >
                  <span
                    style={{
                      width: 5,
                      height: 5,
                      borderRadius: "50%",
                      background: st.color,
                    }}
                  />
                  {st.label}
                </motion.span>
              </div>
              {fund?.description && (
                <motion.p
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.35 }}
                  style={{
                    fontSize: 13,
                    color: "var(--color-text-secondary)",
                    lineHeight: 1.6,
                    marginTop: 12,
                    maxWidth: 500,
                  }}
                >
                  {fund.description}
                </motion.p>
              )}
            </div>

            {/* Stats Grid */}
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))",
                gap: 10,
                marginBottom: 20,
              }}
            >
              <StatCard
                label="Total Pool"
                value={fund?.total_amount || 0}
                delay={0.1}
                icon={<span>💰</span>}
                accent="#f97316"
              />
              <StatCard
                label="Monthly"
                value={fund?.monthly_contribution || 0}
                delay={0.15}
                icon={<span>📅</span>}
                accent="#60a5fa"
              />
              <StatCard
                label="Members"
                value={`${fund?.current_member_count || 0}/${fund?.max_members || 0}`}
                delay={0.2}
                icon={<span>👥</span>}
                accent="#2dd4bf"
                animate={false}
              />
              <StatCard
                label="Duration"
                value={fund?.duration_months || 0}
                suffix="mo"
                delay={0.25}
                icon={<span>⏱️</span>}
                accent="#a78bfa"
              />
            </div>

            {/* ── Contributions Panel ── */}
            {contribData && contribData.contributions.length > 0 && (
              <GlassCard
                hover={false}
                depth={false}
                padding="p-0"
                delay={0.3}
                style={{ marginBottom: 16, padding: "20px" }}
              >
                <div style={{ padding: "18px 20px 4px" }}>
                  <div
                    style={{
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "space-between",
                    }}
                  >
                    <div>
                      <h2
                        style={{
                          fontSize: 15,
                          fontWeight: 700,
                          color: "var(--color-text)",
                          margin: 0,
                        }}
                      >
                        Cycle {contribData.cycle_number} Contributions
                      </h2>
                      <p
                        style={{
                          fontSize: 11,
                          color: "var(--color-text-muted)",
                          margin: "4px 0 14px",
                        }}
                      >
                        {
                          contribData.contributions.filter(
                            (c) => c.status === "paid",
                          ).length
                        }
                        /{contribData.contributions.length} paid
                        {contribData.total_due_amount > 0 && (
                          <>
                            {" "}
                            · {fmt(contribData.total_due_amount)} outstanding
                          </>
                        )}
                      </p>
                    </div>
                    {/* Progress ring */}
                    <div
                      style={{ position: "relative", width: 40, height: 40 }}
                    >
                      <svg width={40} height={40} viewBox="0 0 40 40">
                        <circle
                          cx="20"
                          cy="20"
                          r="16"
                          fill="none"
                          stroke="rgba(255,255,255,0.04)"
                          strokeWidth="3"
                        />
                        <circle
                          cx="20"
                          cy="20"
                          r="16"
                          fill="none"
                          stroke="#22c55e"
                          strokeWidth="3"
                          strokeDasharray={`${(contribData.contributions.filter((c) => c.status === "paid").length / contribData.contributions.length) * 100.53} 100.53`}
                          strokeLinecap="round"
                          transform="rotate(-90 20 20)"
                          style={{ transition: "stroke-dasharray 0.5s" }}
                        />
                      </svg>
                      <span
                        style={{
                          position: "absolute",
                          inset: 0,
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                          fontSize: 9,
                          fontWeight: 700,
                          color: "var(--color-text-muted)",
                        }}
                      >
                        {Math.round(
                          (contribData.contributions.filter(
                            (c) => c.status === "paid",
                          ).length /
                            contribData.contributions.length) *
                            100,
                        )}
                        %
                      </span>
                    </div>
                  </div>
                </div>

                <div style={{ overflowX: "auto" }}>
                  <table
                    style={{
                      width: "100%",
                      borderCollapse: "collapse",
                      textAlign: "left",
                      fontSize: 12,
                    }}
                  >
                    <thead>
                      <tr
                        style={{
                          borderBottom: "1px solid rgba(255,255,255,0.04)",
                        }}
                      >
                        {["Member", "Amount", "Due Date", "Status", ""].map(
                          (h, i) => (
                            <th
                              key={h || i}
                              style={{
                                padding: "8px 16px",
                                fontSize: 10,
                                fontWeight: 700,
                                color: "var(--color-text-muted)",
                                textTransform: "uppercase",
                                letterSpacing: 0.8,
                                textAlign: i === 4 ? "right" : "left",
                                background: "rgba(255,255,255,0.01)",
                              }}
                            >
                              {h}
                            </th>
                          ),
                        )}
                      </tr>
                    </thead>
                    <tbody>
                      {contribData.contributions.map((c, idx) => {
                        const sb = contribStatusBadge(c.status);
                        return (
                          <motion.tr
                            key={c.user_id}
                            initial={{ opacity: 0, x: -8 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ delay: 0.3 + idx * 0.04 }}
                            style={{
                              borderBottom: "1px solid rgba(255,255,255,0.03)",
                            }}
                          >
                            <td
                              style={{
                                padding: "10px 16px",
                                fontSize: 13,
                                fontWeight: 500,
                                color: "var(--color-text)",
                              }}
                            >
                              {members.find((m) => m.user_id === c.user_id)
                                ?.full_name || `${c.user_id.substring(0, 8)}…`}
                            </td>
                            <td
                              style={{
                                padding: "10px 16px",
                                fontSize: 12,
                                fontWeight: 600,
                                color: "var(--color-text)",
                              }}
                            >
                              {fmt(c.amount_due)}
                            </td>
                            <td
                              style={{
                                padding: "10px 16px",
                                fontSize: 12,
                                color: "var(--color-text-secondary)",
                              }}
                            >
                              {new Date(c.due_date).toLocaleDateString(
                                "en-IN",
                                { day: "numeric", month: "short" },
                              )}
                            </td>
                            <td style={{ padding: "10px 16px" }}>
                              <span
                                style={{
                                  fontSize: 10,
                                  fontWeight: 700,
                                  padding: "3px 10px",
                                  borderRadius: 4,
                                  color: sb.color,
                                  background: sb.bg,
                                  textTransform: "uppercase",
                                  letterSpacing: 0.5,
                                }}
                              >
                                {sb.label}
                              </span>
                            </td>
                            <td
                              style={{
                                padding: "10px 16px",
                                textAlign: "right",
                              }}
                            >
                              {c.status === "pending" &&
                                c.user_id === currentUserId && (
                                  <AnimatedButton
                                    variant="primary"
                                    size="sm"
                                    onClick={() =>
                                      router.push(
                                        `/pay?session_id=${c.fund_id}-${c.user_id}-${c.cycle_number}`,
                                      )
                                    }
                                  >
                                    Pay Now
                                  </AnimatedButton>
                                )}
                            </td>
                          </motion.tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </GlassCard>
            )}

            {/* Members Table */}
            {members.length > 0 && (
              <GlassCard hover={false} depth={false} padding="p-0" delay={0.35} style={{ padding: "20px" }}>
                <div style={{ marginBottom: 12 }}>
                  <h2
                    style={{
                      fontSize: 15,
                      fontWeight: 700,
                      color: "var(--color-text)",
                      margin: 0,
                    }}
                  >
                    Fund Members
                  </h2>
                  <p
                    style={{
                      fontSize: 11,
                      color: "var(--color-text-muted)",
                      margin: "4px 0 14px",
                    }}
                  >
                    {members.length} member{members.length !== 1 ? "s" : ""}
                  </p>
                </div>

                <div style={{ overflowX: "auto" }}>
                  <table
                    style={{
                      width: "100%",
                      borderCollapse: "collapse",
                      textAlign: "left",
                      fontSize: 12,
                    }}
                  >
                    <thead>
                      <tr
                        style={{
                          borderBottom: "1px solid rgba(255,255,255,0.04)",
                        }}
                      >
                        {[
                          "Member",
                          "Trust Score",
                          "Status",
                          "Joined",
                          "Actions",
                        ].map((h, i) => (
                          <th
                            key={h}
                            style={{
                              padding: "8px 16px",
                              fontSize: 10,
                              fontWeight: 700,
                              color: "var(--color-text-muted)",
                              textTransform: "uppercase",
                              letterSpacing: 0.8,
                              textAlign: i === 4 ? "right" : "left",
                              background: "rgba(255,255,255,0.01)",
                            }}
                          >
                            {h}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {members.map((m, idx) => {
                        const bandColor =
                          m.risk_band === "Excellent"
                            ? "#22c55e"
                            : m.risk_band === "Good"
                              ? "#34d399"
                              : m.risk_band === "Average"
                                ? "#f59e0b"
                                : m.risk_band === "Risky"
                                  ? "#f97316"
                                  : m.risk_band === "High Risk"
                                    ? "#ef4444"
                                    : "var(--color-text-muted)";
                        return (
                          <motion.tr
                            key={m.user_id}
                            initial={{ opacity: 0, x: -8 }}
                            animate={{ opacity: 1, x: 0 }}
                            transition={{ delay: 0.35 + idx * 0.05 }}
                            style={{
                              borderBottom: "1px solid rgba(255,255,255,0.03)",
                            }}
                          >
                            <td style={{ padding: "10px 16px" }}>
                              <div>
                                <span
                                  style={{
                                    fontSize: 12,
                                    fontWeight: 600,
                                    color: "var(--color-text)",
                                  }}
                                >
                                  {m.full_name || m.email || "—"}
                                </span>
                                <p
                                  style={{
                                    fontSize: 10,
                                    fontFamily: "monospace",
                                    color: "var(--color-text-muted)",
                                    margin: "2px 0 0",
                                  }}
                                >
                                  {m.user_id?.substring(0, 12)}…
                                </p>
                              </div>
                            </td>
                            {/* Trust Score column */}
                            <td style={{ padding: "10px 16px" }}>
                              {m.trust_score > 0 ? (
                                <div
                                  style={{
                                    display: "flex",
                                    flexDirection: "column",
                                    gap: 3,
                                  }}
                                >
                                  <div
                                    style={{
                                      display: "flex",
                                      alignItems: "center",
                                      gap: 6,
                                    }}
                                  >
                                    <span
                                      style={{
                                        fontSize: 16,
                                        fontWeight: 800,
                                        color: bandColor,
                                        lineHeight: 1,
                                      }}
                                    >
                                      {m.trust_score}
                                    </span>
                                    <span
                                      style={{
                                        fontSize: 9,
                                        fontWeight: 700,
                                        padding: "2px 6px",
                                        borderRadius: 3,
                                        color: bandColor,
                                        background: `${bandColor}1a`,
                                        letterSpacing: 0.3,
                                      }}
                                    >
                                      {m.risk_band || "—"}
                                    </span>
                                  </div>
                                  {m.default_probability > 0 && (
                                    <span
                                      style={{
                                        fontSize: 9,
                                        color: "var(--color-text-muted)",
                                      }}
                                    >
                                      {(m.default_probability * 100).toFixed(1)}
                                      % default risk
                                    </span>
                                  )}
                                </div>
                              ) : (
                                <span
                                  style={{
                                    fontSize: 11,
                                    color: "var(--color-text-muted)",
                                  }}
                                >
                                  —
                                </span>
                              )}
                            </td>
                            <td style={{ padding: "10px 16px" }}>
                              <span
                                style={{
                                  fontSize: 10,
                                  fontWeight: 600,
                                  padding: "2px 8px",
                                  borderRadius: 4,
                                  color:
                                    m.status === "active"
                                      ? "#22c55e"
                                      : m.status === "pending"
                                        ? "#f59e0b"
                                        : "var(--color-text-secondary)",
                                  background:
                                    m.status === "active"
                                      ? "rgba(34,197,94,0.08)"
                                      : m.status === "pending"
                                        ? "rgba(245,158,11,0.08)"
                                        : "rgba(255,255,255,0.02)",
                                  textTransform: "capitalize",
                                }}
                              >
                                {m.status}
                              </span>
                            </td>
                            <td
                              style={{
                                padding: "10px 16px",
                                fontSize: 12,
                                color: "var(--color-text-secondary)",
                              }}
                            >
                              {m.joined_at
                                ? new Date(m.joined_at).toLocaleDateString(
                                    "en-IN",
                                    { day: "numeric", month: "short" },
                                  )
                                : "—"}
                            </td>
                            <td
                              style={{
                                padding: "10px 16px",
                                textAlign: "right",
                              }}
                            >
                              {isCreator && m.status === "pending" && (
                                <div style={{ display: "inline-flex", gap: 8 }}>
                                  <AnimatedButton
                                    variant="outline"
                                    size="sm"
                                    onClick={() => handleApprove(m.user_id)}
                                  >
                                    Approve
                                  </AnimatedButton>
                                  <AnimatedButton
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => handleReject(m.user_id)}
                                  >
                                    Reject
                                  </AnimatedButton>
                                </div>
                              )}
                            </td>
                          </motion.tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </GlassCard>
            )}
          </motion.div>

          {/* ── Sidebar ── */}
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.15 }}
            style={{ flex: "0 0 340px", maxWidth: 340 }}
          >
            <div
              style={{
                position: "sticky",
                top: 70,
                display: "flex",
                flexDirection: "column",
                gap: 14,
              }}
            >
              {/* Fund Summary Card */}
              <GlassCard hover={true} depth={true} padding="p-6" style={{ padding: "24px" }}>
                <div style={{ marginBottom: 20 }}>
                  <div
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 8,
                      marginBottom: 14,
                    }}
                  >
                    <div
                      style={{
                        width: 8,
                        height: 8,
                        borderRadius: 2,
                        background: "var(--color-accent)",
                        boxShadow: "0 0 6px rgba(249,115,22,0.4)",
                      }}
                    />
                    <span
                      style={{
                        fontSize: 10,
                        fontWeight: 700,
                        color: "var(--color-text-muted)",
                        textTransform: "uppercase",
                        letterSpacing: 1.2,
                      }}
                    >
                      Summary
                    </span>
                  </div>

                  <div
                    style={{
                      background: "var(--color-bg-subtle)",
                      borderRadius: 8,
                      padding: "16px 18px",
                      marginBottom: 16,
                      boxShadow: "inset 0 1px 3px rgba(0,0,0,0.2)",
                    }}
                  >
                    <span
                      style={{
                        fontSize: 10,
                        fontWeight: 700,
                        color: "var(--color-text-muted)",
                        textTransform: "uppercase",
                        letterSpacing: 0.6,
                      }}
                    >
                      Total Pool
                    </span>
                    <p
                      style={{
                        fontSize: 28,
                        fontWeight: 800,
                        margin: "6px 0 0",
                        letterSpacing: -1,
                        background: "var(--gradient-primary)",
                        WebkitBackgroundClip: "text",
                        WebkitTextFillColor: "transparent",
                        backgroundClip: "text",
                      }}
                    >
                      {fmt(fund?.total_amount || 0)}
                    </p>
                  </div>

                  {[
                    { k: "Monthly", v: fmt(fund?.monthly_contribution || 0) },
                    {
                      k: "Members",
                      v: `${fund?.current_member_count || 0} / ${fund?.max_members || 0}`,
                    },
                    {
                      k: "Duration",
                      v: `${fund?.duration_months || 0} months`,
                    },
                    {
                      k: "Start Date",
                      v: fund?.start_date
                        ? new Date(fund.start_date).toLocaleDateString()
                        : "TBD",
                    },
                  ].map(({ k, v }) => (
                    <div
                      key={k}
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        padding: "10px 0",
                        borderBottom: "1px solid rgba(255,255,255,0.03)",
                      }}
                    >
                      <span
                        style={{
                          fontSize: 12,
                          color: "var(--color-text-muted)",
                        }}
                      >
                        {k}
                      </span>
                      <span
                        style={{
                          fontSize: 12,
                          fontWeight: 600,
                          color: "var(--color-text)",
                        }}
                      >
                        {v}
                      </span>
                    </div>
                  ))}
                </div>

                {/* Apply to join — hide when full or user is the creator */}
                {fund?.status === "open" &&
                  !isCreator &&
                  (fund?.current_member_count ?? 0) <
                    (fund?.max_members ?? 0) && (
                    <div style={{ marginBottom: 14 }}>
                      <h3
                        style={{
                          fontSize: 14,
                          fontWeight: 700,
                          color: "var(--color-text)",
                          margin: "0 0 6px",
                        }}
                      >
                        Join this Fund
                      </h3>
                      <p
                        style={{
                          fontSize: 12,
                          lineHeight: 1.6,
                          color: "var(--color-text-muted)",
                          margin: 0,
                        }}
                      >
                        Your KYC profile will be submitted for organizer review.
                      </p>
                    </div>
                  )}

                {fund?.status === "open" &&
                  !isCreator &&
                  (fund?.current_member_count ?? 0) <
                    (fund?.max_members ?? 0) &&
                  applicationStatus === "none" && (
                    <AnimatedButton
                      onClick={handleApply}
                      variant="primary"
                      size="lg"
                      fullWidth
                      disabled={applyLoading}
                    >
                      {applyLoading ? (
                        <motion.div
                          animate={{ rotate: 360 }}
                          transition={{
                            repeat: Infinity,
                            duration: 0.8,
                            ease: "linear",
                          }}
                          style={{
                            width: 18,
                            height: 18,
                            borderRadius: "50%",
                            border: "2px solid rgba(255,255,255,0.3)",
                            borderTopColor: "#fff",
                            margin: "0 auto",
                          }}
                        />
                      ) : (
                        "Apply Now →"
                      )}
                    </AnimatedButton>
                  )}

                {/* 🟡 Pending */}
                {isUserPending && (
                  <motion.div
                    initial={{ opacity: 0, scale: 0.95 }}
                    animate={{ opacity: 1, scale: 1 }}
                    style={{
                      textAlign: "center",
                      padding: 16,
                      borderRadius: 8,
                      background: "rgba(245,158,11,0.08)",
                      boxShadow: "var(--shadow-card)",
                    }}
                  >
                    <p
                      style={{
                        fontSize: 13,
                        fontWeight: 600,
                        color: "#f59e0b",
                        margin: 0,
                      }}
                    >
                      ⏳ Application under review
                    </p>
                  </motion.div>
                )}

                {/* 🟢 Approved */}
                {isUserActive && (
                  <motion.div
                    initial={{ opacity: 0, scale: 0.95 }}
                    animate={{ opacity: 1, scale: 1 }}
                    style={{
                      textAlign: "center",
                      padding: 16,
                      borderRadius: 8,
                      background: "rgba(34,197,94,0.08)",
                      boxShadow: "var(--shadow-card)",
                    }}
                  >
                    <p
                      style={{
                        fontSize: 13,
                        fontWeight: 600,
                        color: "#22c55e",
                        margin: 0,
                      }}
                    >
                      You joined this fund
                    </p>
                  </motion.div>
                )}

                {isUserRejected && (
                  <motion.div
                    initial={{ opacity: 0, scale: 0.95 }}
                    animate={{ opacity: 1, scale: 1 }}
                    style={{
                      textAlign: "center",
                      padding: 16,
                      borderRadius: 8,
                      background: "rgba(239,68,68,0.08)",
                      boxShadow: "var(--shadow-card)",
                    }}
                  >
                    <p
                      style={{
                        fontSize: 13,
                        fontWeight: 600,
                        color: "#ef4444",
                        margin: 0,
                      }}
                    >
                      Application was rejected
                    </p>
                  </motion.div>
                )}
              </GlassCard>

              {/* ── Auction Card ── */}
              <GlassCard hover={true} depth={true} padding="p-6" style={{ padding: "24px" }}>
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 8,
                    marginBottom: 16,
                  }}
                >
                  <div
                    style={{
                      width: 8,
                      height: 8,
                      borderRadius: 2,
                      background: auctionIsLive
                        ? "#22c55e"
                        : "var(--color-accent)",
                      boxShadow: auctionIsLive
                        ? "0 0 8px rgba(34,197,94,0.5)"
                        : "0 0 6px rgba(249,115,22,0.4)",
                    }}
                  />
                  <span
                    style={{
                      fontSize: 10,
                      fontWeight: 700,
                      color: "var(--color-text-muted)",
                      textTransform: "uppercase",
                      letterSpacing: 1.2,
                    }}
                  >
                    Auction
                  </span>
                  {auctionIsLive && (
                    <motion.span
                      animate={{ opacity: [1, 0.4, 1] }}
                      transition={{ repeat: Infinity, duration: 1.5 }}
                      style={{
                        fontSize: 9,
                        fontWeight: 700,
                        color: "#22c55e",
                        background: "rgba(34,197,94,0.08)",
                        padding: "2px 8px",
                        borderRadius: 4,
                        marginLeft: "auto",
                      }}
                    >
                      ● LIVE
                    </motion.span>
                  )}
                </div>

                {auctionIsLive && auctionSnap?.session && (
                  <div style={{ marginBottom: 16 }}>
                    <div
                      style={{
                        background: "var(--color-bg-subtle)",
                        borderRadius: 8,
                        padding: "16px 18px",
                        marginBottom: 12,
                        boxShadow: "inset 0 1px 3px rgba(0,0,0,0.2)",
                      }}
                    >
                      <span
                        style={{
                          fontSize: 10,
                          fontWeight: 700,
                          color: "var(--color-text-muted)",
                          textTransform: "uppercase",
                          letterSpacing: 0.6,
                        }}
                      >
                        Current Discount
                      </span>
                      <p
                        style={{
                          fontSize: 24,
                          fontWeight: 800,
                          color: "#ef4444",
                          margin: "6px 0 0",
                          letterSpacing: -0.5,
                        }}
                      >
                        −{fmt(auctionSnap.session.current_price)}
                      </p>
                    </div>
                    <div
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        fontSize: 12,
                        marginBottom: 4,
                      }}
                    >
                      <span style={{ color: "var(--color-text-muted)" }}>
                        Cycle
                      </span>
                      <span
                        style={{ fontWeight: 600, color: "var(--color-text)" }}
                      >
                        {auctionSnap.session.cycle_number}
                      </span>
                    </div>
                    {auctionSnap.bids.length > 0 && (
                      <div
                        style={{
                          display: "flex",
                          justifyContent: "space-between",
                          fontSize: 12,
                          marginTop: 4,
                        }}
                      >
                        <span style={{ color: "var(--color-text-muted)" }}>
                          Bids
                        </span>
                        <span
                          style={{
                            fontWeight: 600,
                            color: "var(--color-text)",
                          }}
                        >
                          {auctionSnap.bids.length}
                        </span>
                      </div>
                    )}
                  </div>
                )}

                {auctionHasResult && auctionSnap?.result && (
                  <div
                    style={{
                      background: "rgba(34,197,94,0.06)",
                      borderRadius: 8,
                      padding: "16px 18px",
                      marginBottom: 16,
                    }}
                  >
                    <span
                      style={{
                        fontSize: 10,
                        fontWeight: 700,
                        color: "#22c55e",
                        textTransform: "uppercase",
                        letterSpacing: 0.6,
                      }}
                    >
                      Last Winner
                    </span>
                    <p
                      style={{
                        fontSize: 14,
                        fontWeight: 600,
                        color: "var(--color-text)",
                        margin: "4px 0 0",
                      }}
                    >
                      {members.find(
                        (m) => m.user_id === auctionSnap.result?.winner_user_id,
                      )?.full_name ||
                        auctionSnap.result?.winner_user_id?.substring(0, 16) +
                          "\u2026"}
                    </p>
                    <p
                      style={{
                        fontSize: 14,
                        fontWeight: 700,
                        color: "var(--color-text)",
                        margin: "4px 0 0",
                      }}
                    >
                      Payout: {fmt(auctionSnap.result.payout_amount)}
                    </p>
                  </div>
                )}

                {/* Buttons */}
                <div
                  style={{ display: "flex", flexDirection: "column", gap: 8 }}
                >
                  {/* Organizer: start auction — shown when creator + fund is full + no live auction */}
                  {isCreator &&
                    (fund?.current_member_count ?? 0) >=
                      (fund?.max_members ?? 1) &&
                    !auctionRoomOpen &&
                    !auctionHasResult && (
                      <>
                        <AnimatedButton
                          variant="primary"
                          size="md"
                          fullWidth
                          onClick={openStartAuctionRulebook}
                          disabled={
                            startAuctionLoading || !allContributionsPaid
                          }
                        >
                          {startAuctionLoading
                            ? "Starting…"
                            : "Start Auction 🔨"}
                        </AnimatedButton>
                        {!contribData && (
                          <p
                            style={{
                              fontSize: 11,
                              color: "var(--color-text-muted)",
                              textAlign: "center",
                              margin: 0,
                            }}
                          >
                            Loading contribution status...
                          </p>
                        )}
                        {contribData && !allContributionsPaid && (
                          <p
                            style={{
                              fontSize: 11,
                              color: "var(--color-text-muted)",
                              textAlign: "center",
                              margin: 0,
                            }}
                          >
                            Waiting for all members to pay their contributions.
                          </p>
                        )}
                      </>
                    )}

                  {/* Enter auction room (via rulebook) */}
                  {auctionRoomOpen && (isCreator || isMember) && (
                    <AnimatedButton
                      variant="primary"
                      size="lg"
                      fullWidth
                      onClick={openEnterRoomRulebook}
                    >
                      {auctionIsWaiting
                        ? "Join Auction Waiting Room →"
                        : "Enter Auction Room →"}
                    </AnimatedButton>
                  )}

                  {/* View auction room (ended) */}
                  {auctionHasResult && !auctionIsLive && (
                    <AnimatedButton
                      variant="ghost"
                      size="sm"
                      fullWidth
                      onClick={() => router.push(`/auction/${id}`)}
                    >
                      View Auction Results
                    </AnimatedButton>
                  )}
                </div>
              </GlassCard>
            </div>
          </motion.div>
        </div>
      </main>

      {/* Rulebook Modal */}
      <AuctionRulebook
        isOpen={showRulebook}
        onClose={() => {
          setShowRulebook(false);
          setRulebookAction(null);
        }}
        onEnter={handleRulebookEnter}
        fundName={fund?.name || ""}
        monthlyContribution={fund?.monthly_contribution || 0}
        maxMembers={fund?.max_members || 0}
      />

      {/* Profile Incomplete Popup */}
      <AnimatePresence>
        {showProfilePopup && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            style={{
              position: "fixed",
              inset: 0,
              zIndex: 100,
              background: "rgba(0,0,0,0.6)",
              backdropFilter: "blur(4px)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
            onClick={() => setShowProfilePopup(false)}
          >
            <motion.div
              initial={{ scale: 0.9, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.9, opacity: 0 }}
              onClick={(e) => e.stopPropagation()}
              style={{
                background: "var(--color-bg-card)",
                borderRadius: 12,
                padding: "28px 32px",
                maxWidth: 400,
                width: "90%",
                boxShadow: "0 20px 60px rgba(0,0,0,0.4)",
                border: "1px solid var(--color-border)",
                textAlign: "center",
              }}
            >
              <span
                style={{ fontSize: 36, display: "block", marginBottom: 12 }}
              >
                ⚠️
              </span>
              <h3
                style={{
                  fontSize: 16,
                  fontWeight: 700,
                  color: "var(--color-text)",
                  margin: "0 0 8px",
                }}
              >
                Profile Incomplete
              </h3>
              <p
                style={{
                  fontSize: 13,
                  color: "var(--color-text-muted)",
                  margin: "0 0 20px",
                  lineHeight: 1.5,
                }}
              >
                Please complete your profile and KYC verification before
                applying for chit funds. This is required to verify your
                identity and trustworthiness.
              </p>
              <div
                style={{ display: "flex", gap: 10, justifyContent: "center" }}
              >
                <button
                  onClick={() => setShowProfilePopup(false)}
                  style={{
                    padding: "8px 18px",
                    fontSize: 12,
                    fontWeight: 600,
                    background: "var(--color-bg-subtle)",
                    border: "1px solid var(--color-border)",
                    borderRadius: 6,
                    color: "var(--color-text-muted)",
                    cursor: "pointer",
                    fontFamily: '"Inter",sans-serif',
                  }}
                >
                  Cancel
                </button>
                <button
                  onClick={() => {
                    setShowProfilePopup(false);
                    router.push("/dashboard");
                  }}
                  style={{
                    padding: "8px 18px",
                    fontSize: 12,
                    fontWeight: 600,
                    background: "var(--gradient-primary)",
                    border: "none",
                    borderRadius: 6,
                    color: "#fff",
                    cursor: "pointer",
                    fontFamily: '"Inter",sans-serif',
                    boxShadow: "0 2px 8px rgba(249,115,22,0.3)",
                  }}
                >
                  Go to Profile →
                </button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

      {isMember && (
        <ChatPanel
          fundId={id || ""}
          chatType="fund"
          incomingWsMessage={lastMessage}
        />
      )}

      <style>{`
        @media (max-width: 768px) {
          main {
            padding: 16px 12px !important;
          }
          [style*="flex: 1 1 500px"] {
            min-width: 100% !important;
            max-width: 100% !important;
            flex-basis: 100% !important;
          }
          [style*="flex: 0 0 340px"] {
            max-width: 100% !important;
            position: relative !important;
            top: auto !important;
            flex: 1 1 auto !important;
          }
        }
        @media (max-width: 480px) {
          main {
            padding: 12px 8px !important;
          }
          h1 {
            font-size: 22px !important;
          }
          [style*="gridTemplateColumns: \"repeat(auto-fit, minmax(140px, 1fr))\""] {
            grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)) !important;
            gap: 8px !important;
          }
        }
      `}</style>
    </div>
  );
}
