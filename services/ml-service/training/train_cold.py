import pandas as pd
from sklearn.model_selection import train_test_split
from xgboost import XGBClassifier # type: ignore
import joblib
import os

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

DATA_PATH = os.path.join(BASE_DIR, "data", "credit_risk_dataset.csv")
MODEL_DIR = os.path.join(BASE_DIR, "models")
MODEL_PATH = os.path.join(MODEL_DIR, "cold_model.pkl")

os.makedirs(MODEL_DIR, exist_ok=True)

df = pd.read_csv(DATA_PATH)

df = df.dropna()

X = df[[
    "person_income",
    "person_age",
    "person_emp_length",
    "loan_amnt",
    "loan_percent_income"
]]

y = df["loan_status"]

X_train, X_test, y_train, y_test = train_test_split(
    X, y, test_size=0.2, random_state=42
)


model = XGBClassifier(
    n_estimators=100,
    max_depth=4,
    learning_rate=0.08,
    subsample=0.8,
    colsample_bytree=0.8,
    reg_alpha=0.1,
    reg_lambda=1,
    eval_metric="logloss"
)
model.fit(X_train, y_train)

joblib.dump(model, MODEL_PATH)

print(f"Cold-start model trained and saved at: {MODEL_PATH}")