def get_score(prob):

    prob = float(prob)

    # 🔥 Clamp probability (avoid extreme 0 or 1)
    prob = max(0.02, min(prob, 0.95))

    # 🔥 Convert to score (scaled)
    score = int((1 - prob) * 900 + 100)  
    # → range becomes 100–1000 (more stable)

    # 🔥 Optional: smooth extremes
    if score < 150:
        score = 150
    if score > 950:
        score = 950

    # 🔥 Risk bands (better distribution)
    if score >= 800:
        band = "Excellent"
    elif score >= 700:
        band = "Good"
    elif score >= 600:
        band = "Average"
    elif score >= 450:
        band = "Risky"
    else:
        band = "High Risk"

    return {
        "score": score,
        "risk_band": band,
        "default_probability": prob
    }