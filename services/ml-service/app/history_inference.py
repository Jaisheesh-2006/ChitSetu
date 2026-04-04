import joblib
import os
import pandas as pd
from app.scoring import get_score

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MODEL_PATH = os.path.join(BASE_DIR, "models", "history_model.pkl")

model = joblib.load(MODEL_PATH)

def predict_history(user):

    # Raw input
    LIMIT_BAL = user["LIMIT_BAL"]
    AGE = user["AGE"]
    PAY = user["PAY"]
    BILL_AMT = user["BILL_AMT"]
    PAY_AMT = user["PAY_AMT"]

    # 🔥 FEATURE ENGINEERING (AUTO FROM INPUT)

    avg_bill = sum(BILL_AMT) / len(BILL_AMT)
    avg_pay = sum(PAY_AMT) / len(PAY_AMT)

    utilization = avg_bill / (LIMIT_BAL + 1)
    payment_ratio = avg_pay / (avg_bill + 1)

    late_count = sum(1 for p in PAY if p > 0)
    max_delay = max(PAY)
    severe_delay = sum(1 for p in PAY if p >= 2)
    payment_gap = avg_bill - avg_pay

    # 🔥 STRONG FEATURES
    consistent_delay = int(all(p > 0 for p in PAY))
    delay_weight = sum(p * 2 for p in PAY if p > 0)
    underpayment = int(payment_ratio < 0.8)

    # ✅ EXACT FEATURE ORDER (same as training)
    df = pd.DataFrame([[
        LIMIT_BAL,
        AGE,
        avg_bill,
        avg_pay,
        utilization,
        payment_ratio,
        late_count,
        max_delay,
        severe_delay,
        payment_gap,
        consistent_delay,
        delay_weight,
        underpayment
    ]], columns=[
        "LIMIT_BAL",
        "AGE",
        "avg_bill",
        "avg_pay",
        "utilization",
        "payment_ratio",
        "late_count",
        "max_delay",
        "severe_delay",
        "payment_gap",
        "consistent_delay",
        "delay_weight",
        "underpayment"
    ])

    # Predict
    prob = float(model.predict_proba(df)[0][1])

    # 🔥 BUSINESS LOGIC LAYER (VERY IMPORTANT)
    penalty = 0

    if late_count >= 4:
        penalty += 0.15

    if payment_ratio < 0.7:
        penalty += 0.1

    if consistent_delay == 1:
        penalty += 0.2

    prob = min(prob + penalty, 1.0)

    return get_score(prob)