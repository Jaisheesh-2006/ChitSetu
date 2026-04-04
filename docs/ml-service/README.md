# ML Service Documentation

## Overview
The ML service is a FastAPI app located at `services/ml-service`.
It supports ChitSetu KYC and trust scoring with two paths:
- Cold-start scoring (no credit history)
- History-based scoring (credit history present)

## Service Entry
- `services/ml-service/app/main.py`

## Endpoints
### `GET /`
Health/status endpoint.

### `POST /predict`
Runs inference using either cold or history model based on input structure.

### `POST /generate-credit`
Simulates PAN verification + CIBIL availability.

### `POST /generate-history`
Generates synthetic transaction/credit profile data.
- If `has_cibil=true`: returns history format
- If `has_cibil=false`: returns cold-start format

## Inference Pipeline
- `app/inference.py`: routes to proper model path
- `app/cold_inference.py`: cold model scoring
- `app/history_inference.py`: history model scoring
- `app/scoring.py`: probability to trust/risk output mapping

## Model Artifacts
Stored in `services/ml-service/models`:
- `cold_model.pkl`
- `history_model.pkl`

## Training Pipeline
Training scripts:
- `training/train_cold.py`
- `training/train_history.py`

Datasets:
- `data/credit_risk_dataset.csv`
- `data/UCI_Credit_Card.csv`

## Gemini-Assisted Data Simulation
`app/gemini_service.py` can use Gemini API for richer synthetic outputs.
If Gemini is unavailable, the service falls back to deterministic Python simulation.

Optional env:
- `GEMINI_API_KEY`

## Python Dependencies
From `requirements.txt`:
- fastapi, uvicorn
- pandas, numpy, scikit-learn
- xgboost, joblib
- pydantic

## Local Development
```bash
cd services/ml-service
pip install -r requirements.txt
python -m uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload
```

## Validation Tips
- Hit `GET /` to ensure service is up.
- Use `/generate-credit` then `/generate-history` before calling `/predict` for full KYC simulation flow.
