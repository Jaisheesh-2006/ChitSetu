"use client";
import React, { useEffect, useState, useRef } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { ChatMessage, getChatMessages, sendChatMessage, getCurrentUserId } from "@/services/api";
import type { AuctionWSMessage } from "@/hooks/useAuctionSocket";

interface Props {
  fundId: string;
  chatType: "fund" | "auction";
  cycleNumber?: number;
  incomingWsMessage?: AuctionWSMessage | null;
}

export default function ChatPanel({ fundId, chatType, cycleNumber, incomingWsMessage }: Props) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputStr, setInputStr] = useState("");
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [isExpanded, setIsExpanded] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const currentUserId = getCurrentUserId();

  // Load history on mount
  useEffect(() => {
    getChatMessages(fundId, chatType, cycleNumber, 50)
      .then(d => {
        setMessages(d.reverse()); // old to new for display
      })
      .catch()
      .finally(() => setLoading(false));
  }, [fundId, chatType, cycleNumber]);

  // Handle incoming WS message
  useEffect(() => {
    if (incomingWsMessage && incomingWsMessage.type === "chat_message") {
      if (incomingWsMessage.chat_type === chatType && (!cycleNumber || incomingWsMessage.cycle_number === cycleNumber)) {
        setMessages(prev => {
          // avoid duplicates if we just sent it
          if (prev.find(m => m._id === incomingWsMessage._id)) return prev;
          const newMsg: ChatMessage = {
            _id: incomingWsMessage._id || `ws-${Date.now()}`,
            fund_id: incomingWsMessage.fund_id,
            user_id: incomingWsMessage.user_id,
            full_name: incomingWsMessage.full_name,
            message: incomingWsMessage.message,
            chat_type: incomingWsMessage.chat_type,
            cycle_number: incomingWsMessage.cycle_number,
            created_at: incomingWsMessage.created_at || new Date().toISOString(),
          };
          return [...prev, newMsg];
        });
      }
    }
  }, [incomingWsMessage, chatType, cycleNumber]);

  useEffect(() => {
    if (isExpanded) {
      messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages, isExpanded]);

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputStr.trim() || sending) return;
    const txt = inputStr.trim();
    setInputStr("");
    setSending(true);
    try {
      await sendChatMessage(fundId, chatType, txt, cycleNumber);
    } catch {
      // restore on fail
      setInputStr(txt);
    } finally {
      setSending(false);
    }
  };

  const title = chatType === "auction" ? "Auction Chat" : "Fund Chat";

  return (
    <div style={{
      position: "fixed", bottom: 24, right: 24, zIndex: 100,
      width: isExpanded ? 360 : "auto", display: "flex", flexDirection: "column",
      alignItems: "flex-end"
    }}>
      <AnimatePresence>
        {isExpanded && (
          <motion.div
            initial={{ opacity: 0, y: 20, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 20, scale: 0.95 }}
            transition={{ duration: 0.2 }}
            style={{
              width: "100%", height: 450, background: "rgba(15,23,42,0.95)",
              border: "1px solid rgba(255,255,255,0.1)", borderRadius: 16,
              boxShadow: "0 20px 40px rgba(0,0,0,0.5), inset 0 1px 0 rgba(255,255,255,0.05)",
              backdropFilter: "blur(20px)", display: "flex", flexDirection: "column",
              marginBottom: 16, overflow: "hidden"
            }}
          >
            <div style={{
              padding: "16px 20px", borderBottom: "1px solid rgba(255,255,255,0.08)",
              display: "flex", justifyContent: "space-between", alignItems: "center",
              background: "linear-gradient(180deg, rgba(255,255,255,0.05) 0%, transparent 100%)"
            }}>
              <span style={{ fontSize: 14, fontWeight: 700, color: "var(--color-text)", letterSpacing: 0.5 }}>💬 {title}</span>
              <button onClick={() => setIsExpanded(false)} style={{
                background: "transparent", border: "none", color: "var(--color-text-muted)",
                cursor: "pointer", fontSize: 20, padding: 0, lineHeight: 1
              }}>×</button>
            </div>

            <div style={{ flex: 1, overflowY: "auto", padding: "16px 20px", display: "flex", flexDirection: "column", gap: 12 }}>
              {loading ? (
                <div style={{ textAlign: "center", color: "var(--color-text-muted)", fontSize: 12, marginTop: 40 }}>Loading messages...</div>
              ) : messages.length === 0 ? (
                <div style={{ textAlign: "center", color: "var(--color-text-muted)", fontSize: 12, marginTop: 40 }}>No messages yet. Say hi! 👋</div>
              ) : (
                messages.map((m, i) => {
                  const isMe = m.user_id === currentUserId;
                  return (
                    <div key={i} style={{ display: "flex", flexDirection: "column", alignItems: isMe ? "flex-end" : "flex-start" }}>
                      {!isMe && <span style={{ fontSize: 10, color: "var(--color-text-muted)", marginBottom: 4, paddingLeft: 4 }}>{m.full_name}</span>}
                      <div style={{
                        background: isMe ? "var(--gradient-primary)" : "rgba(255,255,255,0.08)",
                        padding: "8px 12px", borderRadius: 12, borderBottomRightRadius: isMe ? 4 : 12, borderBottomLeftRadius: !isMe ? 4 : 12,
                        maxWidth: "85%", fontSize: 13, lineHeight: 1.4, color: isMe ? "#fff" : "var(--color-text)",
                        boxShadow: isMe ? "0 2px 8px rgba(249,115,22,0.3)" : "none"
                      }}>
                        {m.message}
                      </div>
                    </div>
                  );
                })
              )}
              <div ref={messagesEndRef} />
            </div>

            <form onSubmit={handleSend} style={{
              padding: "12px 16px", borderTop: "1px solid rgba(255,255,255,0.08)",
              display: "flex", gap: 10, background: "rgba(0,0,0,0.2)"
            }}>
              <input
                type="text" value={inputStr} onChange={e => setInputStr(e.target.value)}
                placeholder="Type a message..."
                style={{
                  flex: 1, background: "rgba(255,255,255,0.05)", border: "1px solid rgba(255,255,255,0.1)",
                  borderRadius: 20, padding: "8px 16px", color: "var(--color-text)",
                  fontSize: 13, outline: "none"
                }}
              />
              <button type="submit" disabled={sending || !inputStr.trim()} style={{
                background: "var(--color-accent)", color: "#fff", border: "none",
                borderRadius: "50%", width: 36, height: 36, display: "flex",
                alignItems: "center", justifyContent: "center", cursor: "pointer",
                opacity: (sending || !inputStr.trim()) ? 0.5 : 1, transition: "opacity 0.2s"
              }}>
                ➤
              </button>
            </form>
          </motion.div>
        )}
      </AnimatePresence>

      {!isExpanded && (
        <motion.button
          onClick={() => setIsExpanded(true)}
          whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}
          style={{
            width: 56, height: 56, borderRadius: "50%", background: "var(--gradient-primary)",
            border: "none", color: "#fff", fontSize: 24, cursor: "pointer",
            boxShadow: "0 8px 24px rgba(249,115,22,0.4), inset 0 1px 0 rgba(255,255,255,0.2)",
            display: "flex", alignItems: "center", justifyContent: "center"
          }}
        >
          💬
        </motion.button>
      )}
    </div>
  );
}
