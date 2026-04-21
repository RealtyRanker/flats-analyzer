#!/bin/bash
set -e

NETWORK="realty-net"
APP_IMAGE="flats-analyzer"
APP_CONTAINER="flats-analyzer"
LOG_DIR="/tmp/flats-analyzer-logs"

echo "==> Building image: $APP_IMAGE"
docker build -t "$APP_IMAGE" .

echo "==> Stopping existing container (if any)"
docker rm -f "$APP_CONTAINER" 2>/dev/null || true

echo "==> Creating log directory: $LOG_DIR"
mkdir -p "$LOG_DIR"

echo "==> Starting container: $APP_CONTAINER"
docker run -d \
  --name "$APP_CONTAINER" \
  --network "$NETWORK" \
  --restart unless-stopped \
  -p 9093:9090 \
  -v "$(pwd)/config.yaml:/app/config.yaml:ro" \
  -v "$LOG_DIR:/var/log/flats-analyzer" \
  "$APP_IMAGE"

echo ""
echo "Useful commands:"
echo "  Logs:    docker logs -f $APP_CONTAINER"
echo "  Metrics: curl http://localhost:9092/metrics"
echo "  Health:  curl http://localhost:9092/healthz"
echo "  Stop:    docker stop $APP_CONTAINER"
