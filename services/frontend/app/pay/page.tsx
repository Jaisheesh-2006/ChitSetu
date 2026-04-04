"use client";

import React, { useState, useEffect, Suspense } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import Navbar from "@/components/Navbar";
import GlassCard from "@/components/ui/GlassCard";
import AnimatedButton from "@/components/ui/AnimatedButton";
import UPIPaymentWidget from "@/components/UPIPaymentWidget";
import {
    getPaymentSession,
    createPaymentOrder,
    verifyPayment,
    type SessionDetails,
} from "@/services/api";
import { useAuth } from "@/hooks/useAuth";

const fadeUp = {
    initial: { opacity: 0, y: 16 },
    animate: { opacity: 1, y: 0 },
    transition: { duration: 0.4, ease: "easeOut" as const },
};

function PaymentFlow() {
    const searchParams = useSearchParams();
    const sessionId = searchParams.get("session_id");
    const router = useRouter();
    const { isAuthenticated, isLoading: authLoading } = useAuth();

    const [session, setSession] = useState<SessionDetails | null>(null);
    const [orderId, setOrderId] = useState<string | null>(null);
    const [keyId, setKeyId] = useState<string | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [paymentFailed, setPaymentFailed] = useState(false);
    const [paymentError, setPaymentError] = useState<string | null>(null);
    const [success, setSuccess] = useState(false);

    useEffect(() => {
        if (!authLoading && !isAuthenticated) {
            router.push("/");
        }
    }, [authLoading, isAuthenticated, router]);

    useEffect(() => {
        if (!sessionId) {
            setError("No session ID provided in the URL.");
            setLoading(false);
            return;
        }

        const initPayment = async () => {
            try {
                setLoading(true);
                const details = await getPaymentSession(sessionId);
                setSession(details);
                const orderRes = await createPaymentOrder(sessionId);
                setOrderId(orderRes.order_id);
                setKeyId(orderRes.key_id);
            } catch (err: unknown) {
                setError(
                    err instanceof Error
                        ? err.message
                        : "Failed to initialize payment session."
                );
            } finally {
                setLoading(false);
            }
        };

        if (isAuthenticated) {
            initPayment();
        }
    }, [sessionId, isAuthenticated]);

    const handlePaymentFailed = (errMsg: string) => {
        setPaymentError(errMsg);
        setPaymentFailed(true);
    };

    const handleRetryRedirect = () => {
        if (session?.fund_id) {
            router.push(`/fund/${session.fund_id}?payment_failed=1`);
        } else {
            router.push("/funds");
        }
    };

    const handlePaymentSuccess = async (data: {
        razorpay_payment_id: string;
        razorpay_order_id: string;
        razorpay_signature: string;
    }) => {
        if (!sessionId) return;
        try {
            setLoading(true);
            await verifyPayment({
                session_id: sessionId,
                razorpay_order_id: data.razorpay_order_id,
                razorpay_payment_id: data.razorpay_payment_id,
                razorpay_signature: data.razorpay_signature,
            });
            setSuccess(true);
        } catch (err: unknown) {
            handlePaymentFailed(
                err instanceof Error
                    ? err.message
                    : "Payment verification failed on server."
            );
        } finally {
            setLoading(false);
        }
    };

    if (authLoading || (!error && !paymentFailed && loading && !session)) {
        return (
            <div className="flex min-h-[50vh] flex-col items-center justify-center">
                <div
                    style={{
                        width: 44,
                        height: 44,
                        borderRadius: "50%",
                        border: "3px solid rgba(255,255,255,0.06)",
                        borderTopColor: "#10b981",
                        animation: "spin 0.8s linear infinite",
                    }}
                />
                <p
                    className="mt-4 text-sm"
                    style={{ color: "var(--color-text-secondary)" }}
                >
                    Initializing secure payment...
                </p>
            </div>
        );
    }

    /* ── Payment Failed Popup Modal ── */
    if (paymentFailed) {
        return (
            <>
                {/* Backdrop */}
                <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    style={{
                        position: "fixed",
                        inset: 0,
                        zIndex: 100,
                        background: "rgba(0,0,0,0.65)",
                        backdropFilter: "blur(8px)",
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                    }}
                >
                    {/* Modal card */}
                    <motion.div
                        initial={{ scale: 0.7, opacity: 0, y: 30 }}
                        animate={{ scale: 1, opacity: 1, y: 0 }}
                        transition={{ type: "spring", stiffness: 300, damping: 22, delay: 0.1 }}
                        style={{
                            background: "var(--color-bg-card, #1a1a2e)",
                            borderRadius: 16,
                            padding: "36px 32px 28px",
                            maxWidth: 400,
                            width: "90%",
                            boxShadow: "0 24px 80px rgba(239,68,68,0.15), 0 0 0 1px rgba(239,68,68,0.12)",
                            border: "1px solid rgba(239,68,68,0.15)",
                            textAlign: "center",
                            position: "relative",
                            overflow: "hidden",
                        }}
                    >
                        {/* Ambient glow behind icon */}
                        <div
                            style={{
                                position: "absolute",
                                top: -30,
                                left: "50%",
                                transform: "translateX(-50%)",
                                width: 200,
                                height: 200,
                                borderRadius: "50%",
                                background: "radial-gradient(circle, rgba(239,68,68,0.12), transparent 70%)",
                                pointerEvents: "none",
                            }}
                        />

                        {/* Animated X icon */}
                        <motion.div
                            initial={{ scale: 0, rotate: -180 }}
                            animate={{ scale: 1, rotate: 0 }}
                            transition={{ type: "spring", stiffness: 260, damping: 20, delay: 0.25 }}
                            style={{
                                width: 72,
                                height: 72,
                                borderRadius: "50%",
                                background: "rgba(239,68,68,0.1)",
                                display: "flex",
                                alignItems: "center",
                                justifyContent: "center",
                                margin: "0 auto 20px",
                                boxShadow: "0 0 30px rgba(239,68,68,0.2)",
                                position: "relative",
                            }}
                        >
                            <motion.div
                                animate={{ boxShadow: ["0 0 0px rgba(239,68,68,0.3)", "0 0 20px rgba(239,68,68,0.15)", "0 0 0px rgba(239,68,68,0.3)"] }}
                                transition={{ repeat: Infinity, duration: 2 }}
                                style={{
                                    width: 72,
                                    height: 72,
                                    borderRadius: "50%",
                                    position: "absolute",
                                    inset: 0,
                                }}
                            />
                            <svg
                                width="32"
                                height="32"
                                fill="none"
                                viewBox="0 0 24 24"
                                stroke="#ef4444"
                                strokeWidth={3}
                            >
                                <motion.path
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    d="M6 18L18 6M6 6l12 12"
                                    initial={{ pathLength: 0 }}
                                    animate={{ pathLength: 1 }}
                                    transition={{ duration: 0.5, delay: 0.4 }}
                                />
                            </svg>
                        </motion.div>

                        {/* Title */}
                        <motion.h2
                            initial={{ opacity: 0, y: 8 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 0.35 }}
                            style={{
                                fontSize: 22,
                                fontWeight: 800,
                                color: "#f87171",
                                marginBottom: 8,
                                letterSpacing: "-0.01em",
                            }}
                        >
                            Payment Failed
                        </motion.h2>

                        {/* Description */}
                        <motion.p
                            initial={{ opacity: 0, y: 8 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 0.45 }}
                            style={{
                                fontSize: 13,
                                color: "var(--color-text-muted)",
                                lineHeight: 1.6,
                                marginBottom: 8,
                            }}
                        >
                            Your payment could not be processed.
                        </motion.p>

                        {paymentError && (
                            <motion.p
                                initial={{ opacity: 0 }}
                                animate={{ opacity: 1 }}
                                transition={{ delay: 0.55 }}
                                style={{
                                    fontSize: 11,
                                    color: "rgba(248,113,113,0.7)",
                                    background: "rgba(239,68,68,0.06)",
                                    borderRadius: 6,
                                    padding: "8px 12px",
                                    marginBottom: 20,
                                    fontFamily: "monospace",
                                }}
                            >
                                {paymentError}
                            </motion.p>
                        )}

                        {!paymentError && <div style={{ height: 12 }} />}

                        {/* Buttons */}
                        <motion.div
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 0.55 }}
                            style={{ display: "flex", flexDirection: "column", gap: 10 }}
                        >
                            <AnimatedButton
                                variant="primary"
                                size="lg"
                                fullWidth
                                onClick={handleRetryRedirect}
                            >
                                🔄 Retry Payment
                            </AnimatedButton>
                            <AnimatedButton
                                variant="ghost"
                                size="sm"
                                fullWidth
                                onClick={() => router.push("/dashboard")}
                            >
                                Return to Dashboard
                            </AnimatedButton>
                        </motion.div>
                    </motion.div>
                </motion.div>
            </>
        );
    }

    if (error) {
        return (
            <div className="mx-auto max-w-md pt-12">
                <GlassCard hover={false} padding="p-6">
                    <div className="text-center">
                        <div
                            style={{
                                width: 56,
                                height: 56,
                                borderRadius: 16,
                                background: "rgba(239,68,68,0.1)",
                                display: "flex",
                                alignItems: "center",
                                justifyContent: "center",
                                fontSize: 24,
                                margin: "0 auto 16px",
                            }}
                        >
                            ⚠️
                        </div>
                        <h4
                            style={{
                                fontSize: 18,
                                fontWeight: 700,
                                color: "#f87171",
                                marginBottom: 8,
                            }}
                        >
                            Error loading payment
                        </h4>
                        <p
                            style={{
                                fontSize: 13,
                                color: "var(--color-text-muted)",
                                marginBottom: 20,
                            }}
                        >
                            {error}
                        </p>
                        <AnimatedButton
                            variant="outline"
                            fullWidth
                            onClick={() => router.push("/dashboard")}
                        >
                            Return to Dashboard
                        </AnimatedButton>
                    </div>
                </GlassCard>
            </div>
        );
    }

    if (success) {
        return (
            <motion.div
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                className="mx-auto max-w-md pt-12 text-center"
            >
                <GlassCard hover={false} glow padding="p-8">
                    <motion.div
                        initial={{ scale: 0 }}
                        animate={{ scale: 1 }}
                        transition={{
                            type: "spring",
                            stiffness: 260,
                            damping: 20,
                            delay: 0.2,
                        }}
                        style={{
                            width: 72,
                            height: 72,
                            borderRadius: "50%",
                            background: "rgba(16,185,129,0.12)",
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                            margin: "0 auto 20px",
                            boxShadow: "0 0 30px rgba(16,185,129,0.2)",
                        }}
                    >
                        <svg
                            width="32"
                            height="32"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="#10b981"
                            strokeWidth={3}
                        >
                            <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                d="M5 13l4 4L19 7"
                            />
                        </svg>
                    </motion.div>
                    <h2
                        style={{
                            fontSize: 24,
                            fontWeight: 800,
                            color: "var(--color-text)",
                            marginBottom: 8,
                        }}
                    >
                        Payment Successful!
                    </h2>
                    <p
                        style={{
                            color: "var(--color-text-muted)",
                            fontSize: 14,
                            marginBottom: 24,
                        }}
                    >
                        Your contribution to Fund{" "}
                        {session?.fund_id.substring(0, 8)} has been recorded.
                    </p>
                    <AnimatedButton
                        variant="primary"
                        fullWidth
                        size="lg"
                        onClick={() => router.push("/dashboard")}
                    >
                        Done →
                    </AnimatedButton>
                </GlassCard>
            </motion.div>
        );
    }

    return (
        <div className="mx-auto max-w-lg pt-8">
            <motion.div {...fadeUp} className="mb-8">
                <h1
                    style={{
                        fontSize: 32,
                        fontWeight: 900,
                        letterSpacing: "-0.02em",
                        color: "var(--color-text)",
                    }}
                >
                    Checkout
                </h1>
                <p
                    className="mt-2 text-sm"
                    style={{ color: "var(--color-text-secondary)" }}
                >
                    Complete your monthly chit contribution.
                </p>
            </motion.div>

            {/* Order Summary */}
            <GlassCard hover={false} padding="p-5" delay={0.1} className="mb-6">
                <h3
                    style={{
                        fontSize: 11,
                        fontWeight: 700,
                        color: "var(--color-text-muted)",
                        letterSpacing: "0.08em",
                        textTransform: "uppercase",
                        marginBottom: 16,
                    }}
                >
                    ORDER SUMMARY
                </h3>
                {[
                    { k: "Fund ID", v: session?.fund_id },
                    { k: "Cycle Number", v: session?.cycle_no },
                    {
                        k: "Due Date",
                        v: session && new Date(session.due_date).toLocaleDateString(),
                    },
                ].map(({ k, v }) => (
                    <div
                        key={k}
                        className="flex justify-between py-2.5"
                        style={{ borderBottom: "1px solid rgba(255,255,255,0.04)" }}
                    >
                        <span
                            style={{ fontSize: 13, color: "var(--color-text-secondary)" }}
                        >
                            {k}
                        </span>
                        <span
                            style={{
                                fontSize: 13,
                                fontWeight: 600,
                                color: "var(--color-text)",
                                fontFamily: k === "Fund ID" ? "monospace" : "inherit",
                            }}
                        >
                            {v}
                        </span>
                    </div>
                ))}
                <div
                    className="mt-3 flex justify-between pt-3"
                    style={{ borderTop: "1px solid rgba(255,255,255,0.08)" }}
                >
                    <span
                        style={{ fontSize: 14, fontWeight: 600, color: "var(--color-text)" }}
                    >
                        Amount
                    </span>
                    <span
                        className="gradient-text"
                        style={{ fontSize: 22, fontWeight: 900 }}
                    >
                        ₹{session?.amount?.toLocaleString("en-IN")}
                    </span>
                </div>
            </GlassCard>

            {/* Widget */}
            {session && orderId && keyId && (
                <AnimatePresence>
                    {!loading && (
                        <motion.div
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                        >
                            <UPIPaymentWidget
                                amount={session.amount}
                                razorpayKeyId={keyId}
                                orderId={orderId}
                                fundId={session.fund_id}
                                cycleNo={session.cycle_no}
                                onSuccess={handlePaymentSuccess}
                                onError={(err) => setError(err)}
                            />
                        </motion.div>
                    )}
                </AnimatePresence>
            )}

            {loading && session && (
                <div className="flex justify-center p-8">
                    <div
                        style={{
                            width: 36,
                            height: 36,
                            borderRadius: "50%",
                            border: "3px solid rgba(255,255,255,0.06)",
                            borderTopColor: "#10b981",
                            animation: "spin 0.8s linear infinite",
                        }}
                    />
                </div>
            )}
        </div>
    );
}

export default function PaymentPage() {
    return (
        <div
            className="relative min-h-screen"
            style={{ backgroundColor: "var(--color-bg)", color: "var(--color-text)" }}
        >
            {/* Ambient */}
            <div
                className="pointer-events-none fixed inset-0 z-0"
                style={{
                    background:
                        "radial-gradient(ellipse 500px 500px at 50% 30%, rgba(16,185,129,0.06) 0%, transparent 70%)",
                }}
            />
            <Navbar />
            <main className="relative z-10 app-container py-10 sm:py-12">
                <Suspense
                    fallback={
                        <div className="flex min-h-[50vh] items-center justify-center">
                            <div
                                style={{
                                    width: 44,
                                    height: 44,
                                    borderRadius: "50%",
                                    border: "3px solid rgba(255,255,255,0.06)",
                                    borderTopColor: "#10b981",
                                    animation: "spin 0.8s linear infinite",
                                }}
                            />
                        </div>
                    }
                >
                    <PaymentFlow />
                </Suspense>
            </main>
        </div>
    );
}
