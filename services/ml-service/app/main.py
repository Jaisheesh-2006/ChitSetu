from fastapi import FastAPI, HTTPException
from app.inference import predict_user
from app.gemini_service import (
    check_pan_and_credit,
    generate_history_with_cibil,
    generate_history_without_cibil,
)
import uvicorn

app = FastAPI(title="ChitSetu ML Service", version="1.0.0")


@app.get("/")
def home():
    return {"message": "ML Service Running", "version": "1.0.0"}


@app.post("/predict")
def predict(data: dict):
  
    try:
        result = predict_user(data)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Prediction error: {str(e)}")


@app.post("/generate-credit")
def generate_credit(data: dict):

    pan = str(data.get("pan", "")).strip().upper()
    if not pan:
        raise HTTPException(status_code=400, detail="pan is required")

    try:
        age = int(data.get("age", 25))
        income = float(data.get("income", 30000))
        employment_years = int(data.get("employment_years", 1))
    except (TypeError, ValueError) as e:
        raise HTTPException(status_code=400, detail=f"Invalid numeric field: {e}")

    result = check_pan_and_credit(pan, age, income, employment_years)
    return result


@app.post("/generate-history")
def generate_history(data: dict):
    """
    Generate a synthetic transaction history for the ML model.

    Always called regardless of CIBIL existence.
    When has_cibil=true  → returns has_history=true  format (credit card data)
    When has_cibil=false → returns has_history=false format (cold-start data)

    Request body:
        {
            "has_cibil":        true,
            "cibil_score":      720,    // null if has_cibil=false
            "age":              30,
            "income":           50000,
            "employment_years": 3
        }
    """
    try:
        has_cibil = bool(data.get("has_cibil", False))
        age = int(data.get("age", 25))
        income = float(data.get("income", 30000))
        employment_years = int(data.get("employment_years", 1))
    except (TypeError, ValueError) as e:
        raise HTTPException(status_code=400, detail=f"Invalid field: {e}")

    if has_cibil:
        raw_score = data.get("cibil_score")
        try:
            cibil_score = int(raw_score) if raw_score is not None else 600
            cibil_score = max(300, min(900, cibil_score))
        except (TypeError, ValueError):
            cibil_score = 600
        history = generate_history_with_cibil(cibil_score, income, age, employment_years)
    else:
        history = generate_history_without_cibil(income, age, employment_years)

    return history


if __name__ == "__main__":
    uvicorn.run("app.main:app", host="0.0.0.0", port=8000, reload=True)