"use client";

import React, { useState, useEffect, useRef, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import Navbar from "@/components/Navbar";
import GlassCard from "@/components/ui/GlassCard";
import AnimatedButton from "@/components/ui/AnimatedButton";
import ChatPanel from "@/components/ChatPanel";
import { useAuctionSocket } from "@/hooks/useAuctionSocket";
import {
    getAuction, placeBid as apiPlaceBid, getFundDetails, getFundMembers, activateAuction as apiActivateAuction,
    getWalletHistory, getCurrentUserId, type AuctionSnapshot, type AuctionBid, type FundDetails,
    type WalletHistoryEntry
} from "@/services/api";

function fmt(n: number) {
    return new Intl.NumberFormat("en-IN", { style: "currency", currency: "INR", maximumFractionDigits: 0 }).format(n);
}

const IDLE_WINDOW = 20; // seconds
const INCREMENTS = [10, 100, 200];

export default function AuctionRoomPage() {
    const params = useParams();
    const router = useRouter();
    const id = Array.isArray(params.id) ? params.id[0] : params.id;

    // ── State ──
    const [fund, setFund] = useState<FundDetails | null>(null);
    const [snapshot, setSnapshot] = useState<AuctionSnapshot | null>(null);
    const [bids, setBids] = useState<AuctionBid[]>([]);
    const [currentPrice, setCurrentPrice] = useState(0);
    const [auctionStatus, setAuctionStatus] = useState<"idle" | "live" | "ended">("idle");
    const [countdown, setCountdown] = useState(IDLE_WINDOW);
    const [loading, setLoading] = useState(true);
    const [bidLoading, setBidLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [bidError, setBidError] = useState<string | null>(null);
    const [winnerInfo, setWinnerInfo] = useState<{ userId: string; price: number; payout: number } | null>(null);
    const [members, setMembers] = useState<any[]>([]);
    const [participantCount, setParticipantCount] = useState(0);
    const [waitingForParticipants, setWaitingForParticipants] = useState(true);
    const [currentLeaderUserId, setCurrentLeaderUserId] = useState<string | null>(null);
    const [isSpectator, setIsSpectator] = useState(false);
    const currentUserId = getCurrentUserId();
    const [isActivating, setIsActivating] = useState(false);
    const isCreator = fund && currentUserId === fund.creator_id;

    // ── History Modal State ──
    const [historyAddress, setHistoryAddress] = useState<string | null>(null);
    // const [historyData, setHistoryData] = useState<WalletHistoryEntry[]>([]);
    const [historyLoading, setHistoryLoading] = useState(false);

    const openHistory = async (addr: string) => {
        if (!addr) return;
        setHistoryAddress(addr);
        setHistoryLoading(true);
        try {
            
            setHistoryData(h);
        } catch {
            setHistoryData([]);
        } finally {
            setHistoryLoading(false);
        }
    };

    const countdownRef = useRef<ReturnType<typeof setInterval> | null>(null);
    const lastBidTimeRef = useRef<number>(Date.now());
    const bidFeedRef = useRef<HTMLDivElement>(null);

    // ── WebSocket ──
    const { lastMessage, isConnected, sendMessage } = useAuctionSocket(id);

    useEffect(() => {
        if (!isConnected) return;
        sendMessage({ type: "auction_room_join" });
    }, [isConnected, sendMessage]);

    // ── Initial load ──
    useEffect(() => {
        if (!id) return;
        (async () => {
            try {
                const [f, snap, membersResult] = await Promise.all([
                    getFundDetails(id),
                    getAuction(id),
                    getFundMembers(id).catch(() => []),
                ]);
                setFund(f);
                setSnapshot(snap);
                setBids(snap.bids || []);
                if (Array.isArray(membersResult)) setMembers(membersResult);

                if (snap.session?.status === "live") {
                    setAuctionStatus("live");
                    setCurrentPrice(snap.session.current_price);
                    setCurrentLeaderUserId(snap.session.last_bid_user_id || null);
                    setWaitingForParticipants(false);
                    if (snap.live_countdown_seconds !== undefined && snap.live_countdown_seconds !== null) {
                        setCountdown(Math.max(0, snap.live_countdown_seconds));
                        lastBidTimeRef.current = Date.now() - ((IDLE_WINDOW - snap.live_countdown_seconds) * 1000);
                    }
                } else if (snap.session?.status === "waiting") {
                    setAuctionStatus("idle");
                    setCurrentPrice(snap.session.current_price);
                    setCurrentLeaderUserId(snap.session.last_bid_user_id || null);
                    setWaitingForParticipants(true);
                } else if (snap.result) {
                    setAuctionStatus("ended");
                    setCurrentPrice(snap.result.winning_price);
                    setWinnerInfo({
                        userId: snap.result.winner_user_id,
                        price: snap.result.winning_price,
                        payout: snap.result.payout_amount,
                    });
                }
            } catch (err: unknown) {
                setError(err instanceof Error ? err.message : "Failed to load auction");
            } finally {
                setLoading(false);
            }
        })();
    }, [id]);

    // ── Handle WebSocket messages ──
    useEffect(() => {
        if (!lastMessage) return;

        switch (lastMessage.type) {
            case "auction_started":
                // This is sent when a new session record is created (status=waiting)
                setAuctionStatus("idle");
                setCurrentPrice(lastMessage.current_price || 0);
                setWaitingForParticipants(true);
                setCurrentLeaderUserId(null);
                setWinnerInfo(null);
                setBids([]);
                break;

            case "participants":
                setParticipantCount(lastMessage.count || 0);
                break;

            case "bidding_started":
                setAuctionStatus("live");
                setWaitingForParticipants(false);
                setCurrentLeaderUserId(null);
                setCountdown(IDLE_WINDOW);
                lastBidTimeRef.current = Date.now();
                break;

            case "new_bid": {
                const newBid: AuctionBid = {
                    _id: `ws-${Date.now()}`,
                    fund_id: lastMessage.fund_id,
                    cycle_number: lastMessage.cycle_number || 0,
                    user_id: lastMessage.user_id || "",
                    increment: lastMessage.increment || 0,
                    resulting_price: lastMessage.new_price || 0,
                    created_at: lastMessage.timestamp || new Date().toISOString(),
                };
                setBids(prev => [newBid, ...prev].slice(0, 50));
                setCurrentPrice(lastMessage.new_price || 0);
                setCurrentLeaderUserId(lastMessage.best_bid_user_id ?? null);
                setCountdown(IDLE_WINDOW);
                lastBidTimeRef.current = Date.now();
                break;
            }

            case "auction_ended":
                setAuctionStatus("ended");
                setWinnerInfo({
                    userId: lastMessage.winner_user_id || "",
                    price: lastMessage.winning_price || 0,
                    payout: lastMessage.payout || 0,
                });
                setCountdown(0);
                break;
        }
    }, [lastMessage, fund]);

    // ── Countdown timer ──
    useEffect(() => {
        if (auctionStatus !== "live" || waitingForParticipants) {
            if (countdownRef.current) clearInterval(countdownRef.current);
            return;
        }

        countdownRef.current = setInterval(() => {
            const elapsed = (Date.now() - lastBidTimeRef.current) / 1000;
            const remaining = Math.max(0, IDLE_WINDOW - elapsed);
            setCountdown(Math.ceil(remaining));
        }, 200);

        return () => {
            if (countdownRef.current) clearInterval(countdownRef.current);
        };
    }, [auctionStatus, waitingForParticipants]);

    // ── Check if spectator ──
    useEffect(() => {
        if (snapshot?.members_info && currentUserId) {
            const me = snapshot.members_info.find(m => m.user_id === currentUserId);
            if (me?.is_winner) setIsSpectator(true);
        }
    }, [snapshot, currentUserId]);

    const handleBid = useCallback(async (increment: number) => {
        if (!id || bidLoading) return;
        setBidLoading(true);
        setBidError(null);
        try {
            await apiPlaceBid(id, increment);
        } catch (err: unknown) {
            setBidError(err instanceof Error ? err.message : "Bid failed");
        } finally {
            setBidLoading(false);
        }
    }, [id, bidLoading]);

    const handleActivate = async () => {
        if (!id) return;
        setIsActivating(true);
        try {
            await apiActivateAuction(id);
        } catch (err: any) {
            setBidError(err.message || "Failed to activate bidding");
        } finally {
            setIsActivating(false);
        }
    };

    if (loading) return (
        <div style={{ background: "var(--color-bg)", minHeight: "100vh" }}>
            <Navbar />
            <div style={{ display: "flex", minHeight: "60vh", alignItems: "center", justifyContent: "center" }}>
                <motion.div animate={{ rotate: 360 }} transition={{ repeat: Infinity, duration: 1, ease: "linear" }}
                    style={{ width: 32, height: 32, borderRadius: "50%", border: "2px solid rgba(255,255,255,0.04)", borderTopColor: "var(--color-accent)" }} />
            </div>
        </div>
    );

    const totalPool = (fund?.monthly_contribution || 0) * (fund?.max_members || 0);
    const payoutRemaining = totalPool - currentPrice;
    const countdownPct = (countdown / IDLE_WINDOW) * 100;
    const countdownColor = countdown > 10 ? "#22c55e" : countdown > 5 ? "#f59e0b" : "#ef4444";
    const activeParticipantTarget = members.filter((m) => m.status === "active").length || (fund?.max_members || 0);

    return (
        <div style={{ background: "var(--color-bg)", minHeight: "100vh", color: "var(--color-text)" }}>
            {/* Ambient glow */}
            <div style={{ position: "fixed", inset: 0, pointerEvents: "none", zIndex: 0 }}>
                <div style={{
                    position: "absolute", top: "10%", left: "50%", transform: "translateX(-50%)",
                    width: 600, height: 600, borderRadius: "50%",
                    background: auctionStatus === "live"
                        ? `radial-gradient(circle, rgba(34,197,94,0.04), transparent)`
                        : "radial-gradient(circle, rgba(249,115,22,0.03), transparent)",
                    filter: "blur(100px)",
                    transition: "background 1s",
                }} />
            </div>

            <Navbar />

            <AnimatePresence>
                {historyAddress && (
                    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                        style={{ position: "fixed", inset: 0, zIndex: 2000, background: "rgba(0,0,0,0.85)", backdropFilter: "blur(10px)", display: "flex", alignItems: "center", justifyContent: "center", padding: 20 }}
                        onClick={() => setHistoryAddress(null)}>
                        <motion.div initial={{ scale: 0.9, y: 20 }} animate={{ scale: 1, y: 0 }}
                            style={{ background: "var(--color-bg-card)", borderRadius: 16, width: "100%", maxWidth: 500, maxHeight: "80vh", overflow: "hidden", display: "flex", flexDirection: "column", boxShadow: "var(--shadow-elevated)" }}
                            onClick={e => e.stopPropagation()}>
                            <div style={{ padding: "20px 24px", borderBottom: "1px solid rgba(255,255,255,0.05)", display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                                <div>
                                    <h3 style={{ fontSize: 16, fontWeight: 800, margin: 0 }}>On-Chain History</h3>
                                    <code style={{ fontSize: 10, color: "var(--color-text-muted)" }}>{historyAddress}</code>
                                </div>
                                <button onClick={() => setHistoryAddress(null)} style={{ background: "transparent", border: "none", color: "var(--color-text-muted)", cursor: "pointer", fontSize: 24 }}>×</button>
                            </div>
                            <div style={{ padding: 20, overflowY: "auto", flex: 1 }}>
                                {historyLoading ? (
                                    <div style={{ textAlign: "center", padding: 40 }}><motion.div animate={{ rotate: 360 }} transition={{ repeat: Infinity, duration: 1, ease: "linear" }} style={{ width: 24, height: 24, borderRadius: "50%", border: "2px solid rgba(255,255,255,0.05)", borderTopColor: "var(--color-accent)", margin: "0 auto" }} /></div>
                                ) : (
                                    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
                                        {historyData.map((h, idx) => (
                                            <div key={idx} style={{ padding: 12, borderRadius: 10, background: "rgba(255,255,255,0.02)", border: "1px solid rgba(255,255,255,0.04)" }}>
                                                <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 6 }}>
                                                    <span style={{ fontSize: 10, fontWeight: 700, color: h.type === "credit" ? "#22c55e" : "#ef4444", textTransform: "uppercase" }}>{h.type}</span>
                                                    <span style={{ fontSize: 13, fontWeight: 800 }}>{h.type === "credit" ? "+" : "-"}{h.value.toLocaleString()} CHIT</span>
                                                </div>
                                                <div style={{ display: "flex", justifyContent: "space-between", fontSize: 10, color: "var(--color-text-muted)" }}>
                                                    <span>{h.from.substring(0, 6)}... ➔ {h.to.substring(0, 6)}...</span>
                                                    <a href={`https://explorer.sepolia.era.zksync.dev/tx/${h.tx_hash}`} target="_blank" style={{ color: "var(--color-accent)", textDecoration: "none" }}>Explorer →</a>
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                )}
                            </div>
                        </motion.div>
                    </motion.div>
                )}
            </AnimatePresence>

            <main style={{ position: "relative", zIndex: 1, maxWidth: 1100, margin: "0 auto", padding: "24px 20px" }}>
                <AnimatePresence mode="wait">
                    {auctionStatus === "idle" ? (
                        <motion.div key="waiting-room" initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -10 }}
                            style={{ maxWidth: 640, margin: "40px auto 0" }}>
                            <GlassCard hover={false} padding="p-8">
                                <div style={{ textAlign: "center" }}>
                                    <div style={{ position: "relative", width: 80, height: 80, margin: "0 auto 24px" }}>
                                        <motion.div animate={{ rotate: 360 }} transition={{ repeat: Infinity, duration: 8, ease: "linear" }}
                                            style={{ position: "absolute", inset: 0, border: "2px dashed var(--color-accent)", borderRadius: "50%", opacity: 0.3 }} />
                                        <div style={{ position: "absolute", inset: 10, background: "var(--color-bg-subtle)", borderRadius: "50%", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 32 }}>🏛️</div>
                                    </div>

                                    <h2 style={{ fontSize: 24, fontWeight: 800, color: "var(--color-text)", marginBottom: 8, letterSpacing: -0.5 }}>Waiting Room</h2>
                                    <p style={{ fontSize: 14, color: "var(--color-text-muted)", marginBottom: 32 }}>Bidding will begin once the organiser starts the clock after everyone joins.</p>

                                    <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16, marginBottom: 32 }}>
                                        <div style={{ background: "rgba(255,255,255,0.03)", borderRadius: 12, padding: "16px", border: "1px solid rgba(255,255,255,0.04)" }}>
                                            <p style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 1, marginBottom: 4 }}>Participants</p>
                                            <p style={{ fontSize: 24, fontWeight: 800, color: "var(--color-text)", margin: 0 }}>
                                                {participantCount} <span style={{ fontSize: 14, color: "var(--color-text-muted)", fontWeight: 500 }}>/ {activeParticipantTarget || "—"}</span>
                                            </p>
                                        </div>
                                        <div style={{ background: "rgba(255,255,255,0.03)", borderRadius: 12, padding: "16px", border: "1px solid rgba(255,255,255,0.04)" }}>
                                            <p style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 1, marginBottom: 4 }}>Server status</p>
                                            <p style={{ fontSize: 14, fontWeight: 700, color: isConnected ? "#22c55e" : "#ef4444", margin: "8px 0 0" }}>
                                                {isConnected ? "Connected ●" : "Offline ○"}
                                            </p>
                                        </div>
                                    </div>

                                    {isCreator ? (
                                        <div style={{ background: "rgba(249,115,22,0.05)", borderRadius: 12, padding: "20px", border: "1px solid rgba(249,115,22,0.15)" }}>
                                            {participantCount >= activeParticipantTarget && activeParticipantTarget > 0 ? (
                                                <>
                                                    <p style={{ fontSize: 13, color: "var(--color-text)", fontWeight: 600, marginBottom: 14 }}>Everyone is ready. Start the auction?</p>
                                                    <AnimatedButton variant="primary" fullWidth size="lg" onClick={handleActivate} loading={isActivating}>
                                                        START BIDDING NOW →
                                                    </AnimatedButton>
                                                </>
                                            ) : (
                                                <p style={{ fontSize: 13, color: "var(--color-text-muted)", margin: 0 }}>
                                                    Waiting for {Math.max(0, activeParticipantTarget - participantCount)} more member(s) to join...
                                                </p>
                                            )}
                                        </div>
                                    ) : (
                                        <div style={{ background: "rgba(255,255,255,0.02)", borderRadius: 12, padding: "20px", border: "1px dashed rgba(255,255,255,0.1)" }}>
                                            <p style={{ fontSize: 13, color: "var(--color-text-muted)", margin: 0 }}>
                                                {participantCount >= activeParticipantTarget && activeParticipantTarget > 0
                                                    ? "All members are here! Waiting for the Organiser to start the bidding..."
                                                    : `Waiting for members to join... (${participantCount}/${activeParticipantTarget || "—"})`}
                                            </p>
                                        </div>
                                    )}

                                    {bidError && (
                                        <p style={{ color: "#ef4444", fontSize: 12, marginTop: 16 }}>{bidError}</p>
                                    )}
                                </div>
                            </GlassCard>
                        </motion.div>
                    ) : (
                        <motion.div key="auction-live" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
                            {/* Live Header */}
                            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 20 }}>
                                <div>
                                    <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                                        <AnimatedButton variant="ghost" size="sm" onClick={() => router.push(`/fund/${id}`)}>← Back</AnimatedButton>
                                        <h1 style={{ fontSize: 22, fontWeight: 800, color: "var(--color-text)", margin: 0, letterSpacing: "-0.02em" }}>
                                            {fund?.name || "Auction Room"}
                                        </h1>
                                        <span style={{ fontSize: 10, fontWeight: 700, padding: "2px 8px", borderRadius: 4, background: auctionStatus === "live" ? "rgba(34,197,94,0.1)" : "rgba(255,255,255,0.05)", color: auctionStatus === "live" ? "#22c55e" : "var(--color-text-muted)", textTransform: "uppercase" }}>
                                            {auctionStatus}
                                        </span>
                                    </div>
                                    <p style={{ fontSize: 11, color: "var(--color-text-muted)", margin: "4px 0 0", paddingLeft: 60 }}>
                                        Cycle #{snapshot?.session?.cycle_number || 1} • {participantCount} online
                                    </p>
                                </div>

                                {auctionStatus === "live" && !waitingForParticipants && (
                                    <div style={{ textAlign: "right" }}>
                                        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
                                            <div style={{ textAlign: "right" }}>
                                                <span style={{ fontSize: 10, fontWeight: 700, color: countdownColor, textTransform: "uppercase", display: "block" }}>Closing In</span>
                                                <span style={{ fontSize: 24, fontWeight: 900, color: countdownColor, fontFamily: "monospace" }}>0:{countdown.toString().padStart(2, "0")}</span>
                                            </div>
                                            <div style={{ width: 44, height: 44, position: "relative" }}>
                                                <svg width="44" height="44" viewBox="0 0 44 44">
                                                    <circle cx="22" cy="22" r="18" fill="none" stroke="rgba(255,255,255,0.05)" strokeWidth="4" />
                                                    <motion.circle cx="22" cy="22" r="18" fill="none" stroke={countdownColor} strokeWidth="4" strokeDasharray="113" animate={{ strokeDashoffset: 113 - (113 * countdownPct) / 100 }} transition={{ duration: 0.5 }} strokeLinecap="round" transform="rotate(-90 22 22)" />
                                                </svg>
                                            </div>
                                        </div>
                                    </div>
                                )}
                            </div>

                            <div style={{ display: "grid", gridTemplateColumns: "1fr 340px", gap: 20, alignItems: "start" }}>
                                {/* Left: Controls */}
                                <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
                                    <GlassCard hover={false} depth={false}>
                                        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 24 }}>
                                            <div>
                                                <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 1 }}>Bid Discount</span>
                                                <p style={{ fontSize: 32, fontWeight: 900, color: "#ef4444", margin: "4px 0 0" }}>−{fmt(currentPrice)}</p>
                                            </div>
                                            <div style={{ textAlign: "right" }}>
                                                <span style={{ fontSize: 10, fontWeight: 700, color: "var(--color-text-muted)", textTransform: "uppercase", letterSpacing: 1 }}>Winner Gets</span>
                                                <p style={{ fontSize: 24, fontWeight: 800, color: "#22c55e", margin: "4px 0 0" }}>{fmt(payoutRemaining)}</p>
                                            </div>
                                        </div>

                                        <div style={{ display: "flex", gap: 10 }}>
                                            {INCREMENTS.map(inc => (
                                                <button key={inc} onClick={() => handleBid(inc)} disabled={bidLoading || isSpectator || currentLeaderUserId === currentUserId}
                                                    style={{
                                                        flex: 1, padding: "20px 0", borderRadius: 12, background: "rgba(255,255,255,0.03)", border: "1px solid rgba(255,255,255,0.06)", color: "var(--color-text)", cursor: "pointer",
                                                        opacity: (bidLoading || isSpectator || currentLeaderUserId === currentUserId) ? 0.4 : 1,
                                                        transition: "all 0.2s"
                                                    }}>
                                                    <span style={{ display: "block", fontSize: 18, fontWeight: 800 }}>+{inc}</span>
                                                    <span style={{ fontSize: 9, opacity: 0.6, textTransform: "uppercase" }}>Discount</span>
                                                </button>
                                            ))}
                                        </div>
                                        {isSpectator && <p style={{ fontSize: 11, color: "var(--color-text-muted)", textAlign: "center", marginTop: 12 }}>You win a previous cycle. Spectator Mode.</p>}
                                        {bidError && <p style={{ fontSize: 11, color: "#ef4444", textAlign: "center", marginTop: 12 }}>{bidError}</p>}
                                    </GlassCard>

                                    {/* Bids Table */}
                                    <GlassCard hover={false} depth={false} padding="p-0" style={{ maxHeight: 400, overflowY: "auto" }}>
                                        <div style={{ padding: "16px 20px", borderBottom: "1px solid rgba(255,255,255,0.04)", fontSize: 14, fontWeight: 700 }}>Bids</div>
                                        {bids.length === 0 ? (
                                            <div style={{ padding: 40, textAlign: "center", color: "var(--color-text-muted)" }}>Waiting for first bid...</div>
                                        ) : (
                                            <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 12 }}>
                                                <tbody>
                                                    {bids.map((b, i) => (
                                                        <tr key={b._id} style={{ borderBottom: "1px solid rgba(255,255,255,0.02)", background: i === 0 ? "rgba(249,115,22,0.05)" : "transparent" }}>
                                                            <td style={{ padding: "12px 20px", fontWeight: 700 }}>{members.find(m => m.user_id === b.user_id)?.full_name || b.user_id.substring(0, 8)}</td>
                                                            <td style={{ padding: "12px 20px", color: "var(--color-accent)", fontWeight: 800 }}>+{fmt(b.increment)}</td>
                                                            <td style={{ padding: "12px 20px", textAlign: "right", opacity: 0.5 }}>{new Date(b.created_at).toLocaleTimeString()}</td>
                                                        </tr>
                                                    ))}
                                                </tbody>
                                            </table>
                                        )}
                                    </GlassCard>
                                </div>

                                {/* Right Column: Participants List */}
                                <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
                                    <GlassCard hover={false} depth={false} padding="p-0">
                                        <div style={{ padding: "16px 20px", borderBottom: "1px solid rgba(255,255,255,0.04)", fontSize: 14, fontWeight: 700 }}>Participants</div>
                                        <div style={{ padding: 10 }}>
                                            {members.map(m => (
                                                <div key={m.user_id} style={{ display: "flex", alignItems: "center", gap: 12, padding: "8px 12px", opacity: m.has_won ? 0.5 : 1 }}>
                                                    <div style={{ width: 32, height: 32, borderRadius: 8, background: "var(--color-bg-subtle)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 12 }}>{m.full_name?.charAt(0)}</div>
                                                    <div style={{ flex: 1 }}>
                                                        <p style={{ fontSize: 13, fontWeight: 600, margin: 0 }}>{m.full_name}</p>
                                                        {m.has_won && <span style={{ fontSize: 9, color: "var(--color-accent)" }}>COMPLETED CYCLE</span>}
                                                    </div>
                                                </div>
                                            ))}
                                        </div>
                                    </GlassCard>
                                </div>
                            </div>
                        </motion.div>
                    )}
                </AnimatePresence>
            </main>

            {/* Winner Overlay */}
            <AnimatePresence>
                {auctionStatus === "ended" && winnerInfo && (
                    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                        style={{ position: "fixed", inset: 0, zIndex: 1000, background: "rgba(0,0,0,0.8)", backdropFilter: "blur(10px)", display: "flex", alignItems: "center", justifyContent: "center", padding: 20 }}>
                        <motion.div initial={{ scale: 0.9 }} animate={{ scale: 1 }}
                            style={{ background: "var(--color-bg-card)", borderRadius: 20, padding: 40, textAlign: "center", maxWidth: 460, width: "100%" }}>
                            <div style={{ fontSize: 48, marginBottom: 16 }}>🏆</div>
                            <h2 style={{ fontSize: 24, fontWeight: 800, marginBottom: 8 }}>Auction Complete!</h2>
                            <div style={{ background: "var(--color-bg-subtle)", borderRadius: 12, padding: 24, marginBottom: 24 }}>
                                <p style={{ fontSize: 12, color: "var(--color-text-muted)", marginBottom: 4 }}>WINNER</p>
                                <p style={{ fontSize: 18, fontWeight: 800, marginBottom: 16 }}>{members.find(m => m.user_id === winnerInfo.userId)?.full_name || "Member"}</p>
                                <div style={{ height: 1, background: "rgba(255,255,255,0.05)", marginBottom: 16 }} />
                                <div style={{ display: "flex", justifyContent: "space-between" }}>
                                    <div>
                                        <p style={{ fontSize: 10, color: "var(--color-text-muted)" }}>PAYOUT</p>
                                        <p style={{ fontSize: 16, fontWeight: 900, color: "#22c55e" }}>{fmt(winnerInfo.payout)}</p>
                                    </div>
                                    <div>
                                        <p style={{ fontSize: 10, color: "var(--color-text-muted)" }}>DISCOUNT</p>
                                        <p style={{ fontSize: 16, fontWeight: 900, color: "#ef4444" }}>{fmt(winnerInfo.price)}</p>
                                    </div>
                                </div>
                            </div>
                            <AnimatedButton variant="primary" fullWidth onClick={() => router.push(`/fund/${id}`)}>Back to Fund Panel</AnimatedButton>
                        </motion.div>
                    </motion.div>
                )}
            </AnimatePresence>

            <ChatPanel fundId={id || ""} chatType="auction" cycleNumber={snapshot?.session?.cycle_number || 0} incomingWsMessage={lastMessage} />
        </div>
    );
}
