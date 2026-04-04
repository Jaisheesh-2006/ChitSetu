"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { getAccessToken } from "@/services/api";

const WS_BASE = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080";

export interface AuctionWSMessage {
  type: "auction_started" | "bidding_started" | "new_bid" | "auction_ended" | "participants" | "chat_message" | "member_joined" ;
  fund_id: string;
  cycle_number?: number;
  current_price?: number;
  user_id?: string;
  best_bid_user_id?: string;
  increment?: number;
  new_price?: number;
  timestamp?: string;
  started_at?: string;
  winner_user_id?: string;
  winning_price?: number;
  payout?: number;
  // participants
  count?: number;
  // chat
  message?: string;
  full_name?: string;
  chat_type?: string;
  _id?: string;
  created_at?: string;
}

interface UseAuctionSocketReturn {
  lastMessage: AuctionWSMessage | null;
  isConnected: boolean;
  connectionError: string | null;
  sendMessage: (payload: unknown) => boolean;
}

export function useAuctionSocket(fundId: string | undefined): UseAuctionSocketReturn {
  const [lastMessage, setLastMessage] = useState<AuctionWSMessage | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const sendMessage = useCallback((payload: unknown): boolean => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      return false;
    }

    try {
      wsRef.current.send(JSON.stringify(payload));
      return true;
    } catch {
      return false;
    }
  }, []);

  useEffect(() => {
    if (!fundId) return;

    let isUnmounted = false;

    const openSocket = () => {
      if (isUnmounted) return;

      const token = getAccessToken();
      if (!token) {
        setConnectionError("Not authenticated");
        return;
      }

      // Send token as query param — the backend middleware can parse it
      const url = `${WS_BASE}/ws/funds/${fundId}?token=${encodeURIComponent(token)}`;

      try {
        const ws = new WebSocket(url);
        wsRef.current = ws;

        ws.onopen = () => {
          setIsConnected(true);
          setConnectionError(null);
        };

        ws.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data) as AuctionWSMessage;
            setLastMessage(data);
          } catch {
            // Ignore non-JSON messages
          }
        };

        ws.onerror = () => {
          setConnectionError("WebSocket connection error");
        };

        ws.onclose = () => {
          setIsConnected(false);
          wsRef.current = null;
          if (isUnmounted) return;
          // Auto-reconnect after 3s
          reconnectTimer.current = setTimeout(() => {
            openSocket();
          }, 3000);
        };
      } catch {
        setConnectionError("Failed to create WebSocket connection");
      }
    };

    openSocket();

    return () => {
      isUnmounted = true;
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
      if (wsRef.current) {
        wsRef.current.onclose = null; // prevent reconnect on unmount
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [fundId]);

  return { lastMessage, isConnected, connectionError, sendMessage };
}
