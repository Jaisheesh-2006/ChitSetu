import pandas as pd
from sklearn.model_selection import train_test_split
from xgboost import XGBClassifier # type: ignore
import joblib
import os

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DATA_PATH = os.path.join(BASE_DIR, "data", "UCI_Credit_Card.csv")
MODEL_DIR = os.path.join(BASE_DIR, "models")
MODEL_PATH = os.path.join(MODEL_DIR, "history_model.pkl")

os.makedirs(MODEL_DIR, exist_ok=True)
df = pd.read_csv(DATA_PATH)
df = df.drop(columns=["ID"])
bill_cols = ["BILL_AMT1","BILL_AMT2","BILL_AMT3","BILL_AMT4","BILL_AMT5","BILL_AMT6"]
pay_amt_cols = ["PAY_AMT1","PAY_AMT2","PAY_AMT3","PAY_AMT4","PAY_AMT5","PAY_AMT6"]
pay_cols = ["PAY_0","PAY_2","PAY_3","PAY_4","PAY_5","PAY_6"]

df["avg_bill"] = df[bill_cols].mean(axis=1)
df["avg_pay"] = df[pay_amt_cols].mean(axis=1)
df["utilization"] = df["avg_bill"] / (df["LIMIT_BAL"] + 1)
df["payment_ratio"] = df["avg_pay"] / (df["avg_bill"] + 1)
df["delinquency"] = df[pay_cols].sum(axis=1)

y = df["default.payment.next.month"]
X = df.drop(columns=["default.payment.next.month"])
X_train, X_test, y_train, y_test = train_test_split(
    X, y, test_size=0.2, random_state=42
)


model = XGBClassifier(
    n_estimators=250,
    max_depth=6,
    learning_rate=0.05,
    subsample=0.9,
    colsample_bytree=0.9,
    reg_alpha=0.1,
    reg_lambda=1,
    eval_metric="logloss"
)
model.fit(X_train, y_train)
joblib.dump(model, MODEL_PATH)
print(f"History model trained and saved at: {MODEL_PATH}")