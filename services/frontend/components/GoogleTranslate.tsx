"use client";

import React, { useEffect } from "react";

export default function GoogleTranslate() {
  useEffect(() => {
    // Add initialization function to window
    (window as any).googleTranslateElementInit = () => {
      new (window as any).google.translate.TranslateElement(
        { 
          pageLanguage: 'en', 
          includedLanguages: 'hi,bn,te,mr,ta,ur,gu,kn,ml,or,pa,as,mai,sat,ks,ne,sd,kok,doi,mni,brx,sa,en',
          autoDisplay: false,
          layout: (window as any).google.translate.TranslateElement.InlineLayout.SIMPLE
        },
        'google_translate_element'
      );
    };

    // Check if script is already present to prevent duplicates in React StrictMode
    if (document.getElementById('google-translate-script')) return;

    // Create and append the script
    const script = document.createElement("script");
    script.id = "google-translate-script";
    script.src = "//translate.google.com/translate_a/element.js?cb=googleTranslateElementInit";
    script.async = true;
    document.body.appendChild(script);
  }, []);

  return (
    <div 
      id="google_translate_element" 
      style={{ 
        display: "inline-block", 
        height: 28,
        overflow: "hidden" 
      }} 
    />
  );
}
