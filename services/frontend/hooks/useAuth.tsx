"use client";

import React, {
  createContext,
  useContext,
  useState,
  useCallback,
} from "react";
import {
  authLogin,
  authRegister,
  setTokens,
  clearTokens,
  getAccessToken,
  type TokenPair,
} from "@/services/api";

interface AuthState {
  isAuthenticated: boolean;
  isLoading: boolean;
  user: AuthUser | null;
}

export interface AuthUser {
  id: string;
  email: string;
  full_name?: string;
  avatar_url?: string;
}

interface AuthContextValue extends AuthState {
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  setOAuthSession: (
    accessToken: string,
    refreshToken: string | null,
    user: AuthUser,
  ) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue>({
  isAuthenticated: false,
  isLoading: true,
  user: null,
  login: async () => {},
  register: async () => {},
  setOAuthSession: () => {},
  logout: () => {},
});

export const useAuth = () => useContext(AuthContext);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>(() => {
    const token = getAccessToken();
    return {
      isAuthenticated: !!token,
      isLoading: false,
      user: null,
    };
  });

  const handleTokens = useCallback((data: TokenPair) => {
    setTokens(data.access_token, data.refresh_token);
    setState((prev) => ({ ...prev, isAuthenticated: true, isLoading: false }));
  }, []);

  const setOAuthSession = useCallback(
    (accessToken: string, refreshToken: string | null, user: AuthUser) => {
      setTokens(accessToken, refreshToken || "");
      setState({ isAuthenticated: true, isLoading: false, user });
    },
    [],
  );

  const login = useCallback(
    async (email: string, password: string) => {
      const data = await authLogin(email, password);
      handleTokens(data);
    },
    [handleTokens],
  );

  const register = useCallback(
    async (email: string, password: string) => {
      const data = await authRegister(email, password);
      handleTokens(data);
    },
    [handleTokens],
  );

  const logout = useCallback(() => {
    clearTokens();
    setState({ isAuthenticated: false, isLoading: false, user: null });
    window.location.href = "/";
  }, []);

  return (
    <AuthContext.Provider
      value={{ ...state, login, register, setOAuthSession, logout }}
    >
      {children}
    </AuthContext.Provider>
  );
}
