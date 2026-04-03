import joblib
import os
from app.scoring import get_score

# Load model properly
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MODEL_PATH = os.path.join(BASE_DIR, "models", "history_model.pkl")

history_model = joblib.load(MODEL_PATH)

def predict_history(user):

    # Extract values from user input
    LIMIT_BAL = user["LIMIT_BAL"]
    AGE = user["AGE"]
    PAY = user["PAY"]                 
    BILL_AMT = user["BILL_AMT"]       
    PAY_AMT = user["PAY_AMT"]         

    # Feature engineering (same as training)
    avg_bill = sum(BILL_AMT) / 6
    avg_pay = sum(PAY_AMT) / 6
    utilization = avg_bill / (LIMIT_BAL + 1)
    payment_ratio = avg_pay / (avg_bill + 1)
    delinquency = sum(PAY)

    # FINAL feature vector (IMPORTANT: order must match training)
    features = [[
        LIMIT_BAL,
        user.get("SEX", 1),
        user.get("EDUCATION", 2),
        user.get("MARRIAGE", 1),
        AGE,
        *PAY,
        *BILL_AMT,
        *PAY_AMT,
        avg_bill,
        avg_pay,
        utilization,
        payment_ratio,
        delinquency
    ]]

    # Predict probability of default
    prob = float(history_model.predict_proba(features)[0][1])

    return get_score(prob)