# TigerWallet AI prediction service.
#
# Serves real indicator-based price predictions over real exchange candles.
# The dependency set is deliberately small (FastAPI + httpx); the heavy ML
# stack in ai_layer/requirements.txt is not needed to serve predictions.
FROM python:3.12-slim

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*

COPY ai_layer/python/requirements-service.txt ./
RUN pip install --no-cache-dir -r requirements-service.txt

COPY ai_layer/python/price_prediction.py ai_layer/python/ai_service.py ./

# Run unprivileged.
RUN useradd -m -u 10001 aiservice && chown -R aiservice:aiservice /app
USER aiservice

EXPOSE 8091

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8091/health || exit 1

CMD ["python", "-m", "uvicorn", "ai_service:app", "--host", "0.0.0.0", "--port", "8091"]
