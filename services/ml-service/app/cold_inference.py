import joblib
import os
from app.scoring import get_score


BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MODEL_PATH = os.path.join(BASE_DIR, "models", "cold_model.pkl")

cold_model = joblib.load(MODEL_PATH)


def predict_cold(user):

    try:
  
        income = float(user["income"])                    # person_income
        age = float(user["age"])                          # person_age
        employment_years = float(user["employment_years"])# person_emp_length
        loan_amount = float(user["loan_amount"])          # loan_amnt

        # IMPORTANT: must match training exactly
        if "loan_percent_income" in user:
            loan_percent_income = float(user["loan_percent_income"])
        else:
            loan_percent_income = loan_amount / (income + 1)

        features = [[
            income,
            age,
            employment_years,
            loan_amount,
            loan_percent_income
        ]]

    
       
        # -----------------------------
        # Predict
        # -----------------------------
        prob = float(cold_model.predict_proba(features)[0][1])

        # -----------------------------
        # Convert to score
        # -----------------------------
        return get_score(prob)

    except Exception as e:
        print("COLD MODEL ERROR:", e)
        return {"error": str(e)}