from app.cold_inference import predict_cold
from app.history_inference import predict_history


def predict_user(user):

    if user.get("has_history"):
        return predict_history(user)
    else:
        return predict_cold(user)