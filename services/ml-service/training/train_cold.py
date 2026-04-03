import pandas as pd
from sklearn.model_selection import train_test_split
from xgboost import XGBClassifier # type: ignore
import joblib
import os

# Get base directory (ml-service/)
BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# Paths
DATA_PATH = os.path.join(BASE_DIR, "data", "credit_risk_dataset.csv")
MODEL_DIR = os.path.join(BASE_DIR, "models")
MODEL_PATH = os.path.join(MODEL_DIR, "cold_model.pkl")

# Ensure models folder exists
os.makedirs(MODEL_DIR, exist_ok=True)

# Load dataset
df = pd.read_csv(DATA_PATH)

# Clean
df = df.dropna()

# Features
X = df[[
    "person_income",
    "person_age",
    "person_emp_length",
    "loan_amnt",
    "loan_percent_income"
]]

# Target
y = df["loan_status"]

# Split
X_train, X_test, y_train, y_test = train_test_split(
    X, y, test_size=0.2, random_state=42
)

# Model
model = XGBClassifier(
    n_estimators=150,
    max_depth=5,
    learning_rate=0.07,
    use_label_encoder=False,
    eval_metric="logloss"
)

# Train
model.fit(X_train, y_train)

# Save model
joblib.dump(model, MODEL_PATH)

print(f"Cold-start model trained and saved at: {MODEL_PATH}")