"use client";

import React, { useEffect, useRef, useState } from "react";

const LANGUAGES = [
  { code: "en", label: "English" },
  { code: "hi", label: "हिन्दी" },
  { code: "bn", label: "বাংলা" },
  { code: "te", label: "తెలుగు" },
  { code: "mr", label: "मराठी" },
  { code: "ta", label: "தமிழ்" },
  { code: "ur", label: "اردو" },
  { code: "gu", label: "ગુજરાતી" },
  { code: "kn", label: "ಕನ್ನಡ" },
  { code: "ml", label: "മലയാളം" },
  { code: "or", label: "ଓଡ଼ିଆ" },
  { code: "pa", label: "ਪੰਜਾਬੀ" },
  { code: "as", label: "অসমীয়া" },
  { code: "mai", label: "मैथिली" },
  { code: "ne", label: "नेपाली" },
  { code: "sa", label: "संस्कृतम्" },
  { code: "kok", label: "कोंकणी" },
  { code: "doi", label: "डोगरी" },
  { code: "sd", label: "سنڌي" },
  { code: "ks", label: "كٲشُر" },
  { code: "mni", label: "মৈতৈলোন্" },
  { code: "brx", label: "बड़ो" },
  { code: "sat", label: "ᱥᱟᱱᱛᱟᱲᱤ" },
];

function setGoogTransCookie(langCode: string) {
  // Google Translate reads this cookie to apply translation across the whole page
  const value = langCode === "en" ? "/en/en" : `/en/${langCode}`;
  document.cookie = `googtrans=${value}; path=/`;
  document.cookie = `googtrans=${value}; path=/; domain=${window.location.hostname}`;
  // Also try with leading dot for subdomain support
  const parts = window.location.hostname.split(".");
  if (parts.length >= 2) {
    const rootDomain = "." + parts.slice(-2).join(".");
    document.cookie = `googtrans=${value}; path=/; domain=${rootDomain}`;
  }
}

function getInitialLanguage(): string {
  if (typeof document === "undefined") return "en";
  const match = document.cookie.match(/googtrans=\/en\/([a-z]+)/);
  return match && match[1] && match[1] !== "en" ? match[1] : "en";
}

export default function GoogleTranslate() {
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState(getInitialLanguage);
  const [pendingLanguage, setPendingLanguage] = useState<string | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const processedRef = useRef<string | null>(null);

  useEffect(() => {
    // Initialize Google Translate widget and set up callbacks
    if (typeof document === "undefined") return;

    window.googleTranslateElementInit = () => {
      if (!window.google?.translate) return;
      new window.google.translate.TranslateElement(
        {
          pageLanguage: "en",
          includedLanguages:
            "hi,bn,te,mr,ta,ur,gu,kn,ml,or,pa,as,mai,sat,ks,ne,sd,kok,doi,mni,brx,sa,en",
          autoDisplay: false,
          layout: window.google!.translate!.TranslateElement.InlineLayout.SIMPLE,
        },
        "google_translate_element"
      );
    };

    if (!document.getElementById("google-translate-script")) {
      const script = document.createElement("script");
      script.id = "google-translate-script";
      script.src =
        "//translate.google.com/translate_a/element.js?cb=googleTranslateElementInit";
      script.async = true;
      document.body.appendChild(script);
    }
  }, []);

  // Close on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  // Handle pending language change (side effects in effect, not in handlers)
  useEffect(() => {
    if (!pendingLanguage || processedRef.current === pendingLanguage) return;
    if (typeof document === "undefined") return;

    processedRef.current = pendingLanguage;

    if (pendingLanguage === "en") {
      // Reset translation — clear cookie and reload
      document.cookie =
        "googtrans=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT";
      document.cookie = `googtrans=; path=/; domain=${window.location.hostname}; expires=Thu, 01 Jan 1970 00:00:00 GMT`;
      const parts = window.location.hostname.split(".");
      if (parts.length >= 2) {
        const rootDomain = "." + parts.slice(-2).join(".");
        document.cookie = `googtrans=; path=/; domain=${rootDomain}; expires=Thu, 01 Jan 1970 00:00:00 GMT`;
      }
      window.location.reload();
      return;
    }

    // For non-English: trigger translation
    const gtSelect = document.querySelector(
      ".goog-te-combo"
    ) as HTMLSelectElement | null;

    if (gtSelect) {
      gtSelect.value = pendingLanguage;
      gtSelect.dispatchEvent(new Event("change", { bubbles: true }));
    } else {
      // Widget not ready — set cookie and reload
      setGoogTransCookie(pendingLanguage);
      window.location.reload();
    }
  }, [pendingLanguage]); 

  const handleSelect = (code: string) => {
    setSelected(code);
    setOpen(false);
    setPendingLanguage(code);
  };

  const currentLabel =
    LANGUAGES.find((l) => l.code === selected)?.label ?? "English";

  return (
    <>
      {/* Hidden Google Translate widget */}
      <div
        id="google_translate_element"
        style={{
          position: "absolute",
          opacity: 0,
          pointerEvents: "none",
          width: 0,
          height: 0,
          overflow: "hidden",
        }}
      />

      {/* Custom dropdown */}
      <div
        ref={dropdownRef}
        style={{ position: "relative", display: "inline-block" }}
      >
        <button
          onClick={() => setOpen((o) => !o)}
          title="Translate"
          aria-label="Translate page"
          style={{
            display: "inline-flex",
            alignItems: "center",
            gap: 6,
            height: 40,
            padding: "0 12px",
            borderRadius: 10,
            background: open
              ? "rgba(249,115,22,0.18)"
              : "rgba(249,115,22,0.08)",
            border: "1px solid rgba(249,115,22,0.25)",
            cursor: "pointer",
            transition: "background 0.2s",
          }}
          onMouseEnter={(e) => {
            if (!open)
              (e.currentTarget as HTMLButtonElement).style.background =
                "rgba(249,115,22,0.18)";
          }}
          onMouseLeave={(e) => {
            if (!open)
              (e.currentTarget as HTMLButtonElement).style.background =
                "rgba(249,115,22,0.08)";
          }}
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="#F97316"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="m5 8 6 6" />
            <path d="m4 14 6-6 2-3" />
            <path d="M2 5h12" />
            <path d="M7 2h1" />
            <path d="m22 22-5-10-5 10" />
            <path d="M14 18h6" />
          </svg>
          <span
            style={{
              fontSize: 13,
              fontWeight: 500,
              color: "#F97316",
              maxWidth: 80,
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
            }}
          >
            {currentLabel}
          </span>
          <svg
            width="12"
            height="12"
            viewBox="0 0 24 24"
            fill="none"
            stroke="#F97316"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
            style={{
              transform: open ? "rotate(180deg)" : "rotate(0deg)",
              transition: "transform 0.2s",
            }}
          >
            <polyline points="6 9 12 15 18 9" />
          </svg>
        </button>

        {open && (
          <div
            style={{
              position: "absolute",
              top: "calc(100% + 6px)",
              right: 0,
              background: "#1A1A1A",
              border: "1px solid rgba(249,115,22,0.25)",
              borderRadius: 10,
              overflow: "hidden",
              zIndex: 1000,
              minWidth: 160,
              maxHeight: 280,
              overflowY: "auto",
              boxShadow: "0 8px 24px rgba(0,0,0,0.4)",
            }}
          >
            {LANGUAGES.map((lang) => (
              <button
                key={lang.code}
                onClick={() => handleSelect(lang.code)}
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "space-between",
                  width: "100%",
                  padding: "9px 14px",
                  background:
                    selected === lang.code
                      ? "rgba(249,115,22,0.15)"
                      : "transparent",
                  border: "none",
                  borderBottom: "1px solid rgba(255,255,255,0.05)",
                  cursor: "pointer",
                  textAlign: "left",
                  color: selected === lang.code ? "#F97316" : "#ccc",
                  fontSize: 13,
                  fontWeight: selected === lang.code ? 600 : 400,
                  transition: "background 0.15s",
                }}
                onMouseEnter={(e) => {
                  if (selected !== lang.code)
                    (e.currentTarget as HTMLButtonElement).style.background =
                      "rgba(255,255,255,0.05)";
                }}
                onMouseLeave={(e) => {
                  if (selected !== lang.code)
                    (e.currentTarget as HTMLButtonElement).style.background =
                      "transparent";
                }}
              >
                {lang.label}
                {selected === lang.code && (
                  <svg
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="#F97316"
                    strokeWidth="2.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                )}
              </button>
            ))}
          </div>
        )}
      </div>
    </>
  );
}