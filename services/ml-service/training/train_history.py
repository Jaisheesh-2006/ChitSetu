import pandas as pd
from sklearn.model_selection import train_test_split
from xgboost import XGBClassifier
from sklearn.metrics import roc_auc_score, classification_report
import joblib
import os

# 🔥 PATH SETUP
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DATA_PATH = os.path.join(BASE_DIR, "data", "UCI_Credit_Card.csv")
MODEL_PATH = os.path.join(BASE_DIR, "models", "history_model.pkl")

os.makedirs(os.path.dirname(MODEL_PATH), exist_ok=True)

print("📂 Loading data from:", DATA_PATH)

# Load dataset
df = pd.read_csv(DATA_PATH)
df = df.drop(columns=["ID"])

bill_cols = ["BILL_AMT1","BILL_AMT2","BILL_AMT3","BILL_AMT4","BILL_AMT5","BILL_AMT6"]
pay_amt_cols = ["PAY_AMT1","PAY_AMT2","PAY_AMT3","PAY_AMT4","PAY_AMT5","PAY_AMT6"]
pay_cols = ["PAY_0","PAY_2","PAY_3","PAY_4","PAY_5","PAY_6"]

# 🔥 FEATURE ENGINEERING
df["avg_bill"] = df[bill_cols].mean(axis=1)
df["avg_pay"] = df[pay_amt_cols].mean(axis=1)

df["utilization"] = df["avg_bill"] / (df["LIMIT_BAL"] + 1)
df["payment_ratio"] = df["avg_pay"] / (df["avg_bill"] + 1)

df["late_count"] = (df[pay_cols] > 0).sum(axis=1)
df["max_delay"] = df[pay_cols].max(axis=1)
df["severe_delay"] = (df[pay_cols] >= 2).sum(axis=1)
df["payment_gap"] = df["avg_bill"] - df["avg_pay"]

# 🔥 NEW IMPORTANT FEATURES (CRITICAL FIX)
df["consistent_delay"] = (df[pay_cols] > 0).all(axis=1).astype(int)
df["delay_weight"] = df[pay_cols].apply(
    lambda x: sum([p * 2 for p in x if p > 0]), axis=1
)
df["underpayment"] = (df["payment_ratio"] < 0.8).astype(int)

# Target
y = df["default.payment.next.month"]

# 🔥 FINAL FEATURE SET (MUST MATCH INFERENCE)
X = df[[
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
]]

# Split
X_train, X_test, y_train, y_test = train_test_split(
    X, y, test_size=0.2, random_state=42, stratify=y
)

# 🔥 HANDLE CLASS IMBALANCE
scale_pos_weight = (len(y_train) - sum(y_train)) / sum(y_train)

# Model
model = XGBClassifier(
    n_estimators=350,
    max_depth=6,
    learning_rate=0.05,
    subsample=0.9,
    colsample_bytree=0.9,
    reg_alpha=0.3,
    reg_lambda=2,
    scale_pos_weight=scale_pos_weight,
    eval_metric="auc",
    random_state=42
)

# Train with evaluation
model.fit(
    X_train,
    y_train,
    eval_set=[(X_test, y_test)],
    verbose=False
)

# 🔥 EVALUATION
y_pred_proba = model.predict_proba(X_test)[:, 1]
auc = roc_auc_score(y_test, y_pred_proba)

print("\n📊 Model Performance")
print("AUC Score:", round(auc, 4))

y_pred = (y_pred_proba > 0.5).astype(int)
print("\nClassification Report:\n", classification_report(y_test, y_pred))

# Save model
joblib.dump(model, MODEL_PATH)

print("\n✅ Model trained and saved at:", MODEL_PATH)