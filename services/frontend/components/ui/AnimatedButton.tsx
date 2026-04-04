"use client";
import React from "react";
import { motion } from "framer-motion";

interface Props { 
  children: React.ReactNode; 
  onClick?: React.MouseEventHandler<HTMLButtonElement>; 
  type?: "button" | "submit"; 
  variant?: "primary" | "outline" | "ghost"; 
  size?: "sm" | "md" | "lg"; 
  fullWidth?: boolean; 
  disabled?: boolean; 
  loading?: boolean; 
  className?: string; 
}

export default function AnimatedButton({ 
  children, onClick, type = "button", variant = "primary", size = "md", 
  fullWidth = false, disabled = false, loading = false, className = "" 
}: Props) {
  const sizeS: Record<string, React.CSSProperties> = {
    sm: { padding: "7px 16px", fontSize: 12 },
    md: { padding: "10px 22px", fontSize: 13 },
    lg: { padding: "12px 28px", fontSize: 14 },
  };
  const varS: Record<string, React.CSSProperties> = {
    primary: { background: "var(--gradient-primary)", color: "#fff", border: "none" },
    outline: { background: "transparent", color: "var(--color-accent)", border: "1px solid var(--color-accent)" },
    ghost: { background: "transparent", color: "var(--color-text-secondary)", border: "1px solid var(--color-border)" },
  };
  
  const isActuallyDisabled = disabled || loading;

  return (
    <motion.button 
      type={type} 
      onClick={onClick} 
      disabled={isActuallyDisabled}
      whileHover={isActuallyDisabled ? {} : { opacity: 0.9 }}
      whileTap={isActuallyDisabled ? {} : { scale: 0.98 }}
      className={className}
      style={{ 
        ...sizeS[size], 
        ...varS[variant], 
        width: fullWidth ? "100%" : "auto", 
        borderRadius: 6, 
        fontWeight: 600, 
        fontFamily: '"Inter",sans-serif', 
        cursor: isActuallyDisabled ? "not-allowed" : "pointer", 
        opacity: isActuallyDisabled ? 0.6 : 1, 
        transition: "all 0.2s", 
        letterSpacing: 0.01,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        gap: 8
      }}>
      {loading && (
        <motion.div
          animate={{ rotate: 360 }}
          transition={{ repeat: Infinity, duration: 1, ease: "linear" }}
          style={{ width: 14, height: 14, border: "2px solid rgba(255,255,255,0.3)", borderTopColor: "#fff", borderRadius: "50%" }}
        />
      )}
      {children}
    </motion.button>
  );
}
