"use client";

import React, { useState, useEffect } from "react";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import { motion } from "framer-motion";

// Load UI components
function UPILogo() {
    return (
        <svg width="40" height="15" viewBox="0 0 100 30" fill="currentColor">
            <path d="M10 5H0V25H10V5ZM25 5H15V25H25C30 25 35 20 35 15C35 10 30 5 25 5ZM25 20H20V10H25C27.5 10 30 12.5 30 15C30 17.5 27.5 20 25 20ZM40 5H50V25H40V5Z" />
        </svg>
    );
}

// Ensure Razorpay shape
declare global {
    interface RazorpayPaymentSuccessResponse {
        razorpay_payment_id: string;
        razorpay_order_id: string;
        razorpay_signature: string;
    }

    interface RazorpayPaymentFailedResponse {
        error?: {
            description?: string;
        };
    }

    interface RazorpayOptions {
        key: string;
        amount: number;
        currency: string;
        name: string;
        description: string;
        order_id: string;
        handler: (response: RazorpayPaymentSuccessResponse) => void;
        prefill: {
            name: string;
            email: string;
        };
        theme: {
            color: string;
        };
        method: {
            upi: boolean;
            card: boolean;
        };
    }

    interface RazorpayInstance {
        on(event: "payment.failed", handler: (response: RazorpayPaymentFailedResponse) => void): void;
        open(): void;
    }

    interface RazorpayConstructor {
        new (options: RazorpayOptions): RazorpayInstance;
    }

    interface Window {
        Razorpay?: RazorpayConstructor;
    }
}

interface UPIPaymentWidgetProps {
    amount: number;
    razorpayKeyId: string;
    orderId: string;
    fundId: string;
    cycleNo: number;
    onSuccess: (data: {
        razorpay_payment_id: string;
        razorpay_order_id: string;
        razorpay_signature: string;
    }) => void;
    onError: (error: string) => void;
}

export default function UPIPaymentWidget({
    amount,
    razorpayKeyId,
    orderId,
    fundId,
    cycleNo,
    onSuccess,
    onError,
}: UPIPaymentWidgetProps) {
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        // Load Razorpay Script
        const script = document.createElement("script");
        script.src = "https://checkout.razorpay.com/v1/checkout.js";
        script.async = true;
        document.body.appendChild(script);
        return () => {
            document.body.removeChild(script);
        };
    }, []);

    const handlePayment = () => {
        if (!window.Razorpay) {
            onError("Payment gateway could not load. Please check your connection.");
            return;
        }

        setLoading(true);

        const options: RazorpayOptions = {
            key: razorpayKeyId,
            amount: Math.round(amount * 100), // convert to paise
            currency: "INR",
            name: "ChitSetu",
            description: `Contribution for Fund ${fundId} - Cycle ${cycleNo}`,
            order_id: orderId,
            handler: function (response: RazorpayPaymentSuccessResponse) {
                setLoading(false);
                onSuccess(response);
            },
            prefill: {
                name: "User", // Can be passed as prop if available
                email: "user@example.com",
            },
            theme: {
                color: "#059669",
            },
            method: {
                upi: true,
                card: true,
            },
        };

        const rzp = new window.Razorpay(options);
        rzp.on("payment.failed", function (response: RazorpayPaymentFailedResponse) {
            setLoading(false);
            onError(response.error?.description || "Payment failed");
        });

        rzp.open();
    };

    return (
        <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            className="rounded-2xl border p-5 sm:p-6"
            style={{
                backgroundColor: "var(--color-bg-card)",
                borderColor: "var(--color-border)",
                boxShadow: "var(--shadow-card)",
            }}
        >
            <div className="mb-6 flex items-center justify-between">
                <div>
                    <h3 className="font-semibold" style={{ color: "var(--color-text)" }}>
                        Pay Securely via UPI
                    </h3>
                    <p className="text-sm" style={{ color: "var(--color-text-secondary)" }}>
                        Zero transaction fees for UPI payments
                    </p>
                </div>
                <div style={{ color: "var(--color-text-muted)" }}>
                    <UPILogo />
                </div>
            </div>

            <div className="mb-6 rounded-xl p-4 flex items-center justify-between" style={{ backgroundColor: "var(--color-bg-subtle)" }}>
                <span className="text-sm" style={{ color: "var(--color-text-secondary)" }}>Total Amount Due</span>
                <span className="text-xl font-bold" style={{ color: "var(--color-accent)" }}>
                    ₹{amount.toLocaleString("en-IN")}
                </span>
            </div>

            <Button
                onClick={handlePayment}
                variant="contained"
                size="large"
                fullWidth
                disabled={loading}
                sx={{ borderRadius: "12px", py: 1.5 }}
            >
                {loading ? <CircularProgress size={24} color="inherit" /> : "Proceed to Pay"}
            </Button>

            <p className="mt-4 text-center text-xs" style={{ color: "var(--color-text-muted)" }}>
                Secured by Razorpay. By proceeding, you accept our Terms & Conditions.
            </p>
        </motion.div>
    );
}
