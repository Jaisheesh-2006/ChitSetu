"use client";
import React, { useRef, useState } from "react";
import { motion } from "framer-motion";

interface Props {
  children: React.ReactNode;
  className?: string;
  hover?: boolean;
  glow?: boolean;
  gradient?: boolean;
  padding?: string;
  onClick?: () => void;
  style?: React.CSSProperties;
  delay?: number;
  depth?: boolean;
}

export default function GlassCard({
  children, className = "", hover = true, padding = "p-4",
  onClick, style, delay = 0, depth = true,
}: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const [rot, setRot] = useState({ x: 0, y: 0 });
  const [hovering, setHovering] = useState(false);

  const onMove = (e: React.MouseEvent) => {
    if (!depth || !ref.current) return;
    const rect = ref.current.getBoundingClientRect();
    const mx = (e.clientX - (rect.left + rect.width / 2)) / (rect.width / 2);
    const my = (e.clientY - (rect.top + rect.height / 2)) / (rect.height / 2);
    setRot({ x: -my * 3, y: mx * 3 });
  };

  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, ease: "easeOut", delay }}
      onMouseMove={onMove}
      onMouseEnter={() => setHovering(true)}
      onMouseLeave={() => { setRot({ x: 0, y: 0 }); setHovering(false); }}
      onClick={onClick}
      className={`${padding} ${onClick ? "cursor-pointer" : ""} ${className}`}
      style={{
        position: "relative",
        overflow: "hidden",
        background: "var(--color-bg-card)",
        border: "none",
        borderRadius: 10,
        boxShadow: hovering
          ? "var(--shadow-hover), var(--shadow-glow-sm)"
          : "var(--shadow-card)",
        transform: depth && hovering
          ? `perspective(800px) rotateX(${rot.x}deg) rotateY(${rot.y}deg) translateY(-3px)`
          : "perspective(800px) rotateX(0) rotateY(0) translateY(0)",
        transition: "box-shadow 0.35s ease, transform 0.3s ease",
        transformStyle: "preserve-3d",
        willChange: "transform, box-shadow",
        ...style,
      }}
    >


      {/* Top edge highlight */}
      <div style={{
        position: "absolute", top: 0, left: 0, right: 0, height: 1,
        background: hovering
          ? "linear-gradient(90deg, transparent, rgba(255,255,255,0.06), transparent)"
          : "linear-gradient(90deg, transparent, rgba(255,255,255,0.02), transparent)",
        transition: "background 0.3s",
        pointerEvents: "none",
        zIndex: 1,
      }} />

      <div style={{ position: "relative", zIndex: 2 }}>
        {children}
      </div>
    </motion.div>
  );
}
