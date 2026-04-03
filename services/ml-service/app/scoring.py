def get_score(prob):

    prob = float(prob)

    score = int((1 - prob) * 1000)

    if score > 750:
        band = "Excellent"
    elif score > 650:
        band = "Good"
    elif score > 500:
        band = "Average"
    elif score > 400:
        band = "Risky"
    else:
        band = "High Risk"

    return {
        "score": score,
        "risk_band": band,
        "default_probability": prob
    }