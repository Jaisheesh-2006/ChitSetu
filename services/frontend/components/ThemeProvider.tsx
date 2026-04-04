"use client";

import React, { createContext, useContext, useState, useEffect, useMemo } from "react";
import { createTheme, ThemeProvider as MuiThemeProvider } from "@mui/material/styles";
import CssBaseline from "@mui/material/CssBaseline";

type ThemeMode = "dark" | "light";
interface ThemeContextType { mode: ThemeMode; toggleMode: () => void; }
const ThemeContext = createContext<ThemeContextType>({ mode: "dark", toggleMode: () => {} });
export const useThemeMode = () => useContext(ThemeContext);

export default function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [mode, setMode] = useState<ThemeMode>(() => {
    if (typeof window === "undefined") return "dark";
    const saved = localStorage.getItem("theme-mode");
    return saved === "light" || saved === "dark" ? saved : "dark";
  });

  useEffect(() => {
    localStorage.setItem("theme-mode", mode);
    document.documentElement.classList.remove("light", "dark");
    document.documentElement.classList.add(mode);
  }, [mode]);

  const toggleMode = () => setMode((p) => (p === "dark" ? "light" : "dark"));

  const muiTheme = useMemo(() => {
    const isDark = mode === "dark";
    return createTheme({
      palette: {
        mode,
        primary: { main: "#f97316" },
        secondary: { main: "#f59e0b" },
        background: {
          default: isDark ? "#0d0d0d" : "#fafafa",
          paper: isDark ? "#161616" : "#ffffff",
        },
        text: {
          primary: isDark ? "#e8e8e8" : "#111111",
          secondary: isDark ? "#999999" : "#555555",
        },
        error: { main: isDark ? "#ef4444" : "#dc2626" },
        success: { main: "#22c55e" },
        divider: isDark ? "rgba(255,255,255,0.08)" : "#e5e5e5",
      },
      typography: {
        fontFamily: '"Inter", ui-sans-serif, system-ui, sans-serif',
        button: { textTransform: "none", fontWeight: 600 },
        h1: { fontWeight: 800, letterSpacing: "-0.03em" },
        h2: { fontWeight: 700, letterSpacing: "-0.02em" },
      },
      shape: { borderRadius: 6 },
      components: {
        MuiButton: {
          styleOverrides: {
            root: {
              borderRadius: 6,
              padding: "10px 24px",
              fontSize: "0.875rem",
              fontWeight: 600,
              boxShadow: "none",
              "&:hover": { boxShadow: "none" },
            },
            containedPrimary: {
              background: "linear-gradient(135deg, #f97316, #ea580c)",
              "&:hover": { background: "linear-gradient(135deg, #ea580c, #c2410c)" },
            },
          },
        },
        MuiPaper: {
          styleOverrides: {
            root: {
              backgroundImage: "none",
              backgroundColor: isDark ? "#141414" : "#ffffff",
              border: "none",
              boxShadow: isDark ? "0 2px 8px rgba(0,0,0,0.4)" : "0 2px 8px rgba(0,0,0,0.06)",
            },
          },
        },
        MuiTextField: {
          styleOverrides: {
            root: {
              "& .MuiOutlinedInput-root": {
                borderRadius: 6,
                backgroundColor: isDark ? "#1a1a1a" : "#f0f0f0",
                transition: "all 0.2s",
                boxShadow: isDark ? "0 1px 4px rgba(0,0,0,0.3)" : "0 1px 3px rgba(0,0,0,0.04)",
                "& fieldset": { borderColor: "transparent" },
                "&:hover fieldset": { borderColor: "rgba(249,115,22,0.2)" },
                "&.Mui-focused fieldset": { borderColor: "#f97316", borderWidth: "1px" },
              },
              "& .MuiInputLabel-root": { color: isDark ? "#666666" : "#999999", fontSize: "0.875rem" },
              "& .MuiInputLabel-root.Mui-focused": { color: "#f97316" },
              "& .MuiOutlinedInput-input": { color: isDark ? "#e8e8e8" : "#111111", fontSize: "0.875rem" },
            },
          },
        },
        MuiAlert: {
          styleOverrides: {
            root: { borderRadius: 6, fontSize: "0.8rem" },
          },
        },
      },
    });
  }, [mode]);

  return (
    <ThemeContext.Provider value={{ mode, toggleMode }}>
      <MuiThemeProvider theme={muiTheme}>
        <CssBaseline />
        {children}
      </MuiThemeProvider>
    </ThemeContext.Provider>
  );
}
