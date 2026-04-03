from __future__ import annotations
"""
gemini_service.py
-----------------
Handles three things using the Gemini API:
  1. Simulate PAN verification + CIBIL score existence check
  2. Generate synthetic transaction history (has_history=True format)
  3. Generate synthetic credit profile (has_history=False format)

Falls back to deterministic Python simulation if GEMINI_API_KEY is not set
or if the Gemini call fails.
"""

import os
import json
import random
import re
from typing import Optional, Any

from dotenv import load_dotenv

load_dotenv()

GEMINI_API_KEY = os.getenv("GEMINI_API_KEY", "")

# ---------------------------------------------------------------------------
# Gemini client bootstrap
# ---------------------------------------------------------------------------
_gemini_available = False
_model = None

if GEMINI_API_KEY:
    try:
        import google.generativeai as genai  # type: ignore

        genai.configure(api_key=GEMINI_API_KEY)
        _model = genai.GenerativeModel(
            "gemini-1.5-flash",
            generation_config={"response_mime_type": "application/json"},
        )
        _gemini_available = True
        print("[gemini_service] Gemini API configured successfully.")
    except Exception as e:
        print(f"[gemini_service] Gemini import/config failed: {e}. Using fallback.")
else:
    print("[gemini_service] GEMINI_API_KEY not set. Using Python simulation fallback.")


# ---------------------------------------------------------------------------
# Internal helper
# ---------------------------------------------------------------------------

def _clean_json(raw: str) -> str:
    """Strip markdown code fences from Gemini response."""
    raw = raw.strip()
    raw = re.sub(r"^```json\s*", "", raw)
    raw = re.sub(r"^```\s*", "", raw)
    raw = re.sub(r"\s*```$", "", raw)
    return raw.strip()


def _call_gemini(prompt: str) -> Optional[dict]:
    """
    Call Gemini and parse the JSON response.
    Returns None on any failure so callers can use the fallback.
    """
    if not _gemini_available or _model is None:
        return None
    try:
        response = _model.generate_content(prompt)
        raw = _clean_json(response.text)
        return json.loads(raw)
    except Exception as e:
        print(f"[gemini_service] Gemini call error: {e}")
        return None


# ---------------------------------------------------------------------------
# 1. PAN verification + CIBIL check
# ---------------------------------------------------------------------------

def check_pan_and_credit(
    pan: str,
    age: int,
    income: float,
    employment_years: int,
) -> dict:
    """
    Simulate credit bureau behaviour for a given PAN.

    Returns:
        {
            "pan_verified": bool,
            "has_cibil":    bool,
            "cibil_score":  int | None   (300–900 or null)
        }
    """
    prompt = f"""You are a simulated Indian CIBIL credit bureau API.

Given:
  PAN: {pan}
  Age: {age}
  Monthly Income (INR): {income}
  Employment Years: {employment_years}

Simulate realistic bureau behaviour using these rules:
- 80% of format-valid PANs are verifiable (pan_verified = true)
- If pan_verified, 60% chance the person has a credit history
- CIBIL scores range from 300 to 900
- Higher income AND longer employment → better score tendency
- Use the PAN characters as a seed for consistency

Return ONLY valid JSON — no explanation, no markdown:
{{
  "pan_verified": true,
  "has_cibil": true,
  "cibil_score": 720
}}
(cibil_score must be null when has_cibil is false or pan_verified is false)"""

    result = _call_gemini(prompt)

    if result is None:
        # ── Pure Python fallback ──────────────────────────────────────────
        seed = sum(ord(c) for c in pan)
        rng = random.Random(seed)

        pan_verified = rng.random() < 0.80
        if pan_verified:
            has_cibil = rng.random() < 0.60
            cibil_score = rng.randint(350, 850) if has_cibil else None
        else:
            has_cibil = False
            cibil_score = None

        return {
            "pan_verified": pan_verified,
            "has_cibil": has_cibil,
            "cibil_score": cibil_score,
        }

    # ── Sanitise Gemini output ────────────────────────────────────────────
    pan_verified = bool(result.get("pan_verified", False))
    has_cibil = bool(result.get("has_cibil", False)) and pan_verified
    raw_score = result.get("cibil_score")

    cibil_score = None
    if has_cibil and raw_score is not None:
        try:
            cibil_score = max(300, min(900, int(raw_score)))
        except (TypeError, ValueError):
            cibil_score = None
            has_cibil = False

    return {
        "pan_verified": pan_verified,
        "has_cibil": has_cibil,
        "cibil_score": cibil_score,
    }


# ---------------------------------------------------------------------------
# 2. Synthetic history — has_history = True  (credit card history exists)
# ---------------------------------------------------------------------------

def generate_history_with_cibil(
    cibil_score: int,
    income: float,
    age: int,
    employment_years: int,
) -> dict:
    """
    Generate a realistic 6-month credit card transaction history.
    Used when the user HAS a CIBIL credit score.

    ML model input format:
        has_history  : True
        LIMIT_BAL    : credit limit in INR
        AGE          : age
        PAY          : [6 payment statuses]  -2=-no consumption, -1=paid full, 0=revolving, 1=1m delay, 2=2m delay
        BILL_AMT     : [6 monthly bill amounts in INR]
        PAY_AMT      : [6 payment amounts in INR]
    """
    prompt = f"""You are generating synthetic credit card data for an Indian ML trust-scoring model.

User:
  CIBIL Score     : {cibil_score} / 900
  Monthly Income  : ₹{income}
  Age             : {age}
  Employment Yrs  : {employment_years}

Rules:
- LIMIT_BAL : credit limit INR, typically 2×–5× monthly income
- AGE        : exactly {age}
- PAY        : array of 6 integers, each in [-2, -1, 0, 1, 2].
               Higher CIBIL → more -1 (paid in full) and 0 values.
               Lower CIBIL  → more 1 and 2 (delayed payments).
- BILL_AMT   : 6 monthly bills in INR, realistic relative to LIMIT_BAL
- PAY_AMT    : 6 payment amounts in INR. Higher CIBIL → PAY_AMT closer to BILL_AMT.

Return ONLY this exact JSON — no markdown, no extra keys:
{{
  "has_history": true,
  "LIMIT_BAL": 150000,
  "AGE": {age},
  "PAY": [-1, -1, 0, 0, -1, 0],
  "BILL_AMT": [20000, 18000, 22000, 19000, 21000, 17000],
  "PAY_AMT":  [20000, 18000, 10000, 19000, 21000, 17000]
}}"""

    result = _call_gemini(prompt)

    if result is None:
        # ── Pure Python fallback ──────────────────────────────────────────
        norm = max(0.0, min(1.0, (cibil_score - 300) / 600))  # 0..1
        limit_bal = int(income * random.uniform(2.5, 5.0))

        pay = []
        for _ in range(6):
            r = random.random()
            if r < norm * 0.65:
                pay.append(-1)
            elif r < norm:
                pay.append(0)
            elif r < 0.90:
                pay.append(1)
            else:
                pay.append(2)

        bills = [int(limit_bal * random.uniform(0.15, 0.75)) for _ in range(6)]
        pay_amts = [int(b * random.uniform(norm * 0.5 + 0.1, 1.0)) for b in bills]

        return {
            "has_history": True,
            "LIMIT_BAL": limit_bal,
            "AGE": age,
            "PAY": pay,
            "BILL_AMT": bills,
            "PAY_AMT": pay_amts,
        }

    # ── Sanitise Gemini output ────────────────────────────────────────────
    result["has_history"] = True
    result["AGE"] = age
    result["LIMIT_BAL"] = int(result.get("LIMIT_BAL") or income * 3)

    pay = result.get("PAY", [0] * 6)
    if not isinstance(pay, list) or len(pay) != 6:
        pay = [0] * 6
    result["PAY"] = [max(-2, min(2, int(p))) for p in pay]

    bills = result.get("BILL_AMT", [int(income * 0.5)] * 6)
    if not isinstance(bills, list) or len(bills) != 6:
        bills = [int(income * 0.5)] * 6
    result["BILL_AMT"] = [int(b) for b in bills]

    pay_amts = result.get("PAY_AMT", [int(income * 0.3)] * 6)
    if not isinstance(pay_amts, list) or len(pay_amts) != 6:
        pay_amts = [int(income * 0.3)] * 6
    result["PAY_AMT"] = [int(p) for p in pay_amts]

    return result


# ---------------------------------------------------------------------------
# 3. Synthetic profile — has_history = False  (no credit card history)
# ---------------------------------------------------------------------------

def generate_history_without_cibil(
    income: float,
    age: int,
    employment_years: int,
) -> dict:
    """
    Generate a synthetic financial profile for a user with NO credit history.

    ML model input format:
        has_history      : False
        income           : annual income in INR
        age              : age
        employment_years : years employed
        loan_amount      : hypothetical loan amount
        loan_percent_income : loan_amount / annual_income (0.1–0.8)
    """
    annual_income = income * 12

    prompt = f"""You are generating a synthetic financial profile for an Indian ML trust-scoring model.
The person has NO prior credit history.

User:
  Monthly Income  : ₹{income}  (Annual: ₹{annual_income})
  Age             : {age}
  Employment Yrs  : {employment_years}

Rules:
- income          : annual income in INR (use {int(annual_income)})
- age             : exactly {age}
- employment_years: exactly {employment_years}
- loan_amount     : a realistic hypothetical personal or home loan INR amount
- loan_percent_income : loan_amount ÷ annual_income, must be between 0.10 and 0.80

Return ONLY this exact JSON — no markdown, no extra keys:
{{
  "has_history": false,
  "income": {int(annual_income)},
  "age": {age},
  "employment_years": {employment_years},
  "loan_amount": 200000,
  "loan_percent_income": 0.33
}}"""

    result = _call_gemini(prompt)

    if result is None:
        # ── Pure Python fallback ──────────────────────────────────────────
        ann = int(annual_income)
        loan = int(ann * random.uniform(0.4, 1.8))
        ratio = round(min(0.80, max(0.10, loan / max(ann, 1))), 4)

        return {
            "has_history": False,
            "income": ann,
            "age": age,
            "employment_years": employment_years,
            "loan_amount": loan,
            "loan_percent_income": ratio,
        }

    # ── Sanitise Gemini output ────────────────────────────────────────────
    result["has_history"] = False
    result["age"] = age
    result["employment_years"] = employment_years

    try:
        result["income"] = int(result.get("income") or annual_income)
    except (TypeError, ValueError):
        result["income"] = int(annual_income)

    try:
        result["loan_amount"] = int(result.get("loan_amount") or annual_income * 0.7)
    except (TypeError, ValueError):
        result["loan_amount"] = int(annual_income * 0.7)

    try:
        ratio = float(result.get("loan_percent_income") or 0.3)
        result["loan_percent_income"] = round(max(0.10, min(0.80, ratio)), 4)
    except (TypeError, ValueError):
        result["loan_percent_income"] = 0.30

    return result
