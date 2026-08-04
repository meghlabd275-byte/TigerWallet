# TigerWallet MasterWallet - Complete Build & Deployment Guide

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Backend Services](#backend-services)
3. [Android App](#android-app)
4. [iOS App](#ios-app)
5. [Desktop App (C++)](#desktop-app-c)
6. [Web App](#web-app)
7. [Browser Extensions](#browser-extensions)
8. [Flutter App](#flutter-app)
9. [Cloud Deployment](#cloud-deployment)
10. [Environment Variables](#environment-variables)

---

## Prerequisites

### Required Tools
```bash
# System packages
sudo apt-get update
sudo apt-get install -y \
    build-essential \
    curl \
    git \
    wget \
    pkg-config \
    libssl-dev

# Go (v1.21+)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Rust (latest)
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source ~/.cargo/env

# Node.js (v18+)
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# Flutter SDK
git clone https://github.com/flutter/flutter.git -b stable --depth 1
export PATH="$PATH:$HOME/flutter/bin"

# Android SDK
mkdir -p ~/android-sdk/cmdline-tools
cd ~/android-sdk/cmdline-tools
wget https://dl.google.com/android/repository/commandlinetools-linux-11076708_latest.zip
unzip commandlinetools-linux-11076708_latest.zip
mv cmdline-tools latest
export ANDROID_HOME=~/android-sdk
export PATH=$PATH:$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools

# Xcode (macOS only)
xcode-select --install
```

---

## Backend Services

### 1. Go Master Wallet Service

```bash
cd /workspace/project/TigerWallet/go/services/master_wallet_service

# Build
go build -o master-wallet-service .

# Run
./master-wallet-service

# Or run directly
go run main.go
```

**Environment Variables:**
```bash
export DB_HOST=postgres
export DB_PORT=5432
export DB_USER=tigerwallet
export DB_PASSWORD=your_secure_password
export DB_NAME=tigerwallet_master
export REDIS_HOST=redis
export REDIS_PORT=6379
export JWT_SECRET=your_jwt_secret_32_bytes
export ENCRYPTION_KEY=your_encryption_key_32_bytes
export MASTER_WALLET_PORT=9095
export PLATFORM_FEE_PERCENT=0.3
export FEE_WALLET=0x742d35Cc6634C0532925a3b844Bc9e7595f1234
```

### 2. Rust Fetchers Service

```bash
cd /workspace/project/TigerWallet/rust/masterwallet_fetchers

# Build
cargo build --release

# Run
./target/release/masterwallet-fetchers-server

# Or with custom config
./target/release/masterwallet-fetchers-server --config config.toml
```

**Configuration (config.toml):**
```toml
[server]
host = "0.0.0.0"
port = 9096

[database]
host = "postgres"
port = 5432
user = "tigerwallet"
password = "your_secure_password"
database = "tigerwallet_master"
max_connections = 20

[redis]
host = "redis"
port = 6379
password = ""
db = 0
pool_size = 20

[blockchain]
rpc_timeout_ms = 5000
max_retries = 3

[security]
encryption_key = "your_32_byte_encryption_key"
jwt_secret = "your_jwt_secret"
```

---

## Android App

### Build APK/AAB

```bash
cd /workspace/project/TigerWallet/master_wallet/android

# Install dependencies
./gradlew dependencies

# Build Debug APK
./gradlew assembleDebug

# Build Release APK
./gradlew assembleRelease

# Build AAB (for Play Store)
./gradlew bundleRelease
```

**Output:**
- Debug: `app/build/outputs/apk/debug/app-debug.apk`
- Release: `app/build/outputs/apk/release/app-release.apk`
- AAB: `app/build/outputs/bundle/release/app-release.aab`

### Build with Custom Configuration

```bash
# Set API endpoint
export MASTER_API_URL=https://api.tigerwallet.io/master

# Build with custom config
./gradlew assembleRelease -PMASTER_API_URL=$MASTER_API_URL
```

---

## iOS App

### Prerequisites (macOS)
```bash
# Install CocoaPods
sudo gem install cocoapods

# Install dependencies
cd /workspace/project/TigerWallet/master_wallet/ios/TigerMasterWallet
pod install
```

### Build

```bash
cd /workspace/project/TigerWallet/master_wallet/ios/TigerMasterWallet

# Open in Xcode
open TigerMasterWallet.xcworkspace

# Build via command line
xcodebuild -workspace TigerMasterWallet.xcworkspace \
  -scheme TigerMasterWallet \
  -configuration Debug \
  -destination 'platform=iOS Simulator,name=iPhone 15 Pro' \
  build

# Build for Simulator
xcodebuild -workspace TigerMasterWallet.xcworkspace \
  -scheme TigerMasterWallet \
  -configuration Debug \
  -destination generic/platform=iOS Simulator \
  build

# Build for Device (requires signing)
xcodebuild -workspace TigerMasterWallet.xcworkspace \
  -scheme TigerMasterWallet \
  -configuration Release \
  -destination 'generic/platform=iOS' \
  CODE_SIGN_IDENTITY="Your Code Signing Identity" \
  PROVISIONING_PROFILE_SPECIFIER="Your Profile Name" \
  build
```

**Output:**
- Simulator: `build/Build/Products/Debug-iphonesimulator/TigerMasterWallet.app`
- Device: `build/Build/Products/Release-iphoneos/TigerMasterWallet.ipa`

---

## Desktop App (C++)

### Prerequisites

```bash
# Install CMake
sudo apt-get install cmake

# Install OpenSSL
sudo apt-get install libssl-dev

# Install curl
sudo apt-get install libcurl4-openssl-dev

# Install Boost
sudo apt-get install libboost-all-dev
```

### Build

```bash
cd /workspace/project/TigerWallet/desktop_wallet

# Create build directory
mkdir -p build && cd build

# Configure
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DENABLE_TLS=ON \
  -DENABLE_HARDWARE_WALLET=ON

# Build
cmake --build . --config Release -j$(nproc)

# Install
sudo cmake --install .
```

**Output:** `build/src/tiger-wallet`

### Run

```bash
# Set environment
export MASTER_API_URL=https://api.tigerwallet.io/master
export LOG_LEVEL=info

# Run
./build/src/tiger-wallet
```

---

## Web App

### Build

```bash
cd /workspace/project/TigerWallet/master_wallet/web

# Install dependencies
npm install

# Build for development
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

### Environment Variables

```bash
# .env.local
NEXT_PUBLIC_MASTER_WALLET_API=https://api.tigerwallet.io/master
NEXT_PUBLIC_WS_URL=wss://api.tigerwallet.io/ws
NEXT_PUBLIC_NETWORK=mainnet
```

### Output

- Development: `http://localhost:3000`
- Production: `master-wallet.tigerwallet.io` (after deployment)

---

## Browser Extensions

### Build for Chrome/Edge/Brave

```bash
cd /workspace/project/TigerWallet/browser_extensions/chrome_extension

# Install dependencies
npm install

# Build
npm run build

# Package
npm run package
```

### Build for Firefox

```bash
cd /workspace/project/TigerWallet/browser_extensions/firefox_extension

# Install dependencies
npm install

# Build
npm run build
```

### Load Extension

**Chrome/Edge/Brave:**
1. Open `chrome://extensions/`
2. Enable "Developer mode"
3. Click "Load unpacked"
4. Select `browser_extensions/chrome_extension/dist`

**Firefox:**
1. Open `about:debugging#/runtime/this-firefox`
2. Click "Load Temporary Add-on"
3. Select `browser_extensions/firefox_extension/dist/manifest.json`

---

## Flutter App

### Build Android

```bash
cd /workspace/project/TigerWallet/master_wallet/flutter

# Get dependencies
flutter pub get

# Build debug APK
flutter build apk --debug

# Build release APK
flutter build apk --release

# Build AAB
flutter build appbundle --release
```

### Build iOS

```bash
cd /workspace/project/TigerWallet/master_wallet/flutter

# Get dependencies
flutter pub get

# Build for simulator
flutter build ios --simulator --no-codesign

# Build for device (requires signing)
flutter build ios --release
```

### Build Web

```bash
cd /workspace/project/TigerWallet/master_wallet/flutter

# Build web
flutter build web --release
```

### Build Desktop

```bash
# Enable desktop support
flutter config --enable-linux-desktop
flutter config --enable-macos-desktop
flutter config --enable-windows-desktop

# Build
flutter build linux --release
flutter build macos --release
flutter build windows --release
```

---

## Cloud Deployment

### Docker Compose (Recommended)

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  # PostgreSQL Database
  postgres:
    image: postgres:15-alpine
    container_name: tiger-master-postgres
    environment:
      POSTGRES_USER: tigerwallet
      POSTGRES_PASSWORD: your_secure_password
      POSTGRES_DB: tigerwallet_master
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    networks:
      - tiger-master-net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U tigerwallet"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redis Cache
  redis:
    image: redis:7-alpine
    container_name: tiger-master-redis
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    networks:
      - tiger-master-net
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Go Master Wallet Service
  master-wallet-api:
    build:
      context: ./go/services/master_wallet_service
      dockerfile: Dockerfile
    container_name: tiger-master-api
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: tigerwallet
      DB_PASSWORD: your_secure_password
      DB_NAME: tigerwallet_master
      REDIS_HOST: redis
      REDIS_PORT: 6379
      JWT_SECRET: your_jwt_secret_32_bytes
      ENCRYPTION_KEY: your_encryption_key_32_bytes
      MASTER_WALLET_PORT: 9095
      PLATFORM_FEE_PERCENT: 0.3
    ports:
      - "9095:9095"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - tiger-master-net

  # Rust Fetchers
  master-wallet-fetchers:
    build:
      context: ./rust/masterwallet_fetchers
      dockerfile: Dockerfile
    container_name: tiger-master-fetchers
    environment:
      DATABASE_HOST: postgres
      DATABASE_PORT: 5432
      DATABASE_USER: tigerwallet
      DATABASE_PASSWORD: your_secure_password
      DATABASE_NAME: tigerwallet_master
      REDIS_HOST: redis
      REDIS_PORT: 6379
    ports:
      - "9096:9096"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - tiger-master-net

  # Web Frontend
  master-wallet-web:
    build:
      context: ./master_wallet/web
      dockerfile: Dockerfile
    container_name: tiger-master-web
    environment:
      NEXT_PUBLIC_MASTER_WALLET_API: http://master-wallet-api:9095
      NEXT_PUBLIC_WS_URL: ws://master-wallet-api:9095
    ports:
      - "3000:3000"
    depends_on:
      - master-wallet-api
    networks:
      - tiger-master-net

volumes:
  postgres_data:
  redis_data:

networks:
  tiger-master-net:
    driver: bridge
```

### Deploy

```bash
# Start all services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

### Kubernetes Deployment

Create `k8s/` directory with:

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: master-wallet-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: master-wallet-api
  template:
    metadata:
      labels:
        app: master-wallet-api
    spec:
      containers:
      - name: api
        image: tigerwallet/master-wallet-api:latest
        ports:
        - containerPort: 9095
        env:
        - name: DB_HOST
          value: postgres-service
        - name: REDIS_HOST
          value: redis-service
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

---

## Environment Variables

### Backend Services

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | localhost |
| `DB_PORT` | PostgreSQL port | 5432 |
| `DB_USER` | Database user | tigerwallet |
| `DB_PASSWORD` | Database password | - |
| `DB_NAME` | Database name | tigerwallet_master |
| `REDIS_HOST` | Redis host | localhost |
| `REDIS_PORT` | Redis port | 6379 |
| `JWT_SECRET` | JWT signing secret | - |
| `ENCRYPTION_KEY` | AES encryption key | - |
| `MASTER_WALLET_PORT` | Service port | 9095 |
| `PLATFORM_FEE_PERCENT` | Platform fee % | 0.3 |
| `FEE_WALLET` | Fee collection wallet | - |

### Frontend Apps

| Variable | Description | Default |
|----------|-------------|---------|
| `NEXT_PUBLIC_MASTER_WALLET_API` | API base URL | http://localhost:9095 |
| `NEXT_PUBLIC_WS_URL` | WebSocket URL | ws://localhost:9095 |
| `NEXT_PUBLIC_NETWORK` | Network (mainnet/testnet) | mainnet |

---

## SSL/TLS Setup

### Using Let's Encrypt (Production)

```bash
# Install Certbot
sudo apt-get install certbot python3-certbot-nginx

# Get certificate
sudo certbot --nginx -d master-wallet.tigerwallet.io

# Auto-renewal
sudo certbot renew --dry-run
```

---

## Monitoring & Logging

### Prometheus Metrics

```bash
# Add to docker-compose.yml
metrics:
  image: prom/prometheus:latest
  volumes:
    - ./prometheus.yml:/etc/prometheus/prometheus.yml
  ports:
    - "9090:9090"
```

### Log Aggregation

```bash
# Add to docker-compose.yml
logging:
  image: grafana/loki:latest
  ports:
    - "3100:3100"
```

---

## Quick Start Script

```bash
#!/bin/bash
# deploy-master-wallet.sh

set -e

echo "🚀 Deploying TigerWallet Master Wallet..."

# 1. Start infrastructure
echo "📦 Starting PostgreSQL & Redis..."
docker-compose up -d postgres redis

# 2. Wait for services
echo "⏳ Waiting for services..."
sleep 10

# 3. Run migrations
echo "🗄️ Running migrations..."
docker-compose run --rm master-wallet-api migrate

# 4. Start API
echo "🔌 Starting API..."
docker-compose up -d master-wallet-api

# 5. Start Fetchers
echo "📡 Starting Fetchers..."
docker-compose up -d master-wallet-fetchers

# 6. Build & Start Web
echo "🌐 Building Web App..."
cd master_wallet/web
npm install
npm run build
cd ../..

# 7. Start Web
docker-compose up -d master-wallet-web

echo "✅ Deployment complete!"
echo "🌐 Web UI: http://localhost:3000"
echo "🔌 API: http://localhost:9095"
echo "📡 Metrics: http://localhost:9090"
```

---

## Support & Troubleshooting

### Check Service Health

```bash
# API health
curl http://localhost:9095/health

# PostgreSQL
docker exec -it tiger-master-postgres pg_isready -U tigerwallet

# Redis
docker exec -it tiger-master-redis redis-cli ping
```

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f master-wallet-api
docker-compose logs -f master-wallet-fetchers
```
