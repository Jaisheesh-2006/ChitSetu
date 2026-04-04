FROM python:3.11-slim AS builder

WORKDIR /build

COPY requirements.txt /build/requirements.txt
RUN pip wheel --no-cache-dir --wheel-dir /build/wheels -r /build/requirements.txt

FROM python:3.11-slim AS runner

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

COPY requirements.txt /app/requirements.txt
COPY --from=builder /build/wheels /wheels
RUN pip install --no-cache-dir --no-index --find-links=/wheels -r /app/requirements.txt

COPY app /app/app
COPY models /app/models

EXPOSE 8000

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
