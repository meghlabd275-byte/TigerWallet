# TigerWallet - Complete Build & Deployment Guide

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Mobile Apps](#mobile-apps-canonical-user_walletandroid-user_walletios)
3. [Android Native](#android-native)
4. [iOS Native](#ios-native)
5. [Desktop App (Tauri)](#desktop-tauri--canonical-desktop_app)
6. [Web App (NextJS)](#web-nextjs)
7. [Browser Extensions](#browser-extensions)
8. [Backend Services](#backend-services)
9. [Production Deployment](#production-deployment)

---

## Prerequisites

### Required Tools
```bash
# Node.js (v18+)
node --version  # v18.x.x

# Go (v1.21+)
go version     # go1.21.x

# Rust (v1.70+)
rustc --version  # 1.70.x

# Flutter SDK (v3.x)
flutter --version  # 3.x.x

# Android SDK
echo $ANDROID_HOME
# Should point to: /opt/android-sdk or ~/Android/Sdk

# Xcode (for iOS)
xcodebuild --version  # 15.x+

# PostgreSQL (v14+)
psql --version  # 14.x+

# Redis
redis-cli --version  # 6.x+
```

---

## Mobile Apps (canonical: user_wallet/android, user_wallet/ios)

### Build Android APK
```bash
cd /workspace/project/TigerWallet/user_wallet/android

# Build debug APK
./gradlew assembleDebug

# Build release APK
./gradlew assembleRelease

# Output: app/build/outputs/apk/release/app-release.apk
```

### Build iOS (requires macOS)
```bash
cd /workspace/project/TigerWallet/user_wallet/ios

# Open the App/ Swift sources in Xcode and build, or:
xcodebuild -scheme TigerWallet \
  -configuration Release \
  -destination 'generic/platform=iOS' \
  build

# Output: build/Build/Products/Release-iphoneos/TigerWallet.app
```

### Build Web
```bash
cd /workspace/project/TigerWallet/user_wallet/web

npm install
npm run build

# Output: build/
```

---

## Android Native

### Build Debug APK
```bash
cd /workspace/project/TigerWallet/user_wallet/android

# Using Gradle
./gradlew assembleDebug

# Or with Android Studio
# File -> Open -> android/
# Build -> Build APK
```

### Build Release APK
```bash
cd /workspace/project/TigerWallet/user_wallet/android

# Create release build
./gradlew assembleRelease

# Or debug APK with signing
./gradlew assembleRelease \
  -Pandroid.signingKeyAlias=your_alias \
  -Pandroid.signingKeyPassword=your_password \
  -Pandroid.signingStoreFile=your_keystore.jks
```

### Output Locations
- Debug: `app/build/outputs/apk/debug/app-debug.apk`
- Release: `app/build/outputs/apk/release/app-release.apk`

---

## iOS Native

### Build with Xcode
```bash
cd /workspace/project/TigerWallet/user_wallet/ios

# Open the App/ Swift sources in Xcode, or build from command line
xcodebuild -scheme TigerWallet \
  -configuration Debug \
  -destination 'platform=iOS Simulator,name=iPhone 15' \
  build

# Build for App Store
xcodebuild -scheme TigerWallet \
  -configuration Release \
  -destination 'generic/platform=iOS' \
  archive
```

### Output
- Debug: `build/Build/Products/Debug-iphoneos/TigerWallet.app`
- Release: `build/Build/Products/Archive/TigerWallet.xcarchive`

---

## Desktop (Tauri — canonical: desktop_app/)

### Build for Linux
```bash
cd /workspace/project/TigerWallet/desktop_app/tauri

cargo build --release

# Output: target/release/tigerwallet
```

### Build for Windows
```bash
cd /workspace/project/TigerWallet/desktop_app/tauri

cargo build --release

# Output: target/release/tigerwallet.exe
```

### Build for macOS
```bash
cd /workspace/project/TigerWallet/desktop_app/tauri

cargo build --release

# Output: build/tigerwallet.app
```

### Dependencies
```bash
# Ubuntu/Debian
sudo apt-get install -y \
  build-essential \
  cmake \
  libssl-dev \
  libcurl4-openssl-dev \
  libjsoncpp-dev \
  libwebsocketpp-dev \
  libboost-all-dev
```

---

## Web NextJS

### Development
```bash
cd /workspace/project/TigerWallet/frontend/web_nextjs

# Install dependencies
npm install

# Copy environment file
cp .env.example .env.local

# Edit .env.local with your API URLs
# NEXT_PUBLIC_API_URL=https://api.tigerwallet.com

# Run development server
npm run dev

# Output: http://localhost:3000
```

### Production Build
```bash
cd /workspace/project/TigerWallet/frontend/web_nextjs

# Build for production
npm run build

# Start production server
npm start

# Or with PM2
npm install -g pm2
pm2 start npm --name "tigerwallet-web" -- start
```

### Output
- Static files: `.next/`
- Standalone output: `.next/standalone/`

---

## Browser Extensions

### Chrome Extension
```bash
cd /workspace/project/TigerWallet/browser_extensions/chrome

# Install dependencies
npm install

# Build
npm run build

# Output: dist/
```

### Load in Chrome
1. Open `chrome://extensions/`
2. Enable "Developer mode"
3. Click "Load unpacked"
4. Select `browser_extensions/chrome/dist/`

### Firefox Extension
```bash
cd /workspace/project/TigerWallet/browser_extensions/firefox

# Build
npm run build

# Output: dist/
```

### Load in Firefox
1. Open `about:debugging#/runtime/this-firefox`
2. Click "Load Temporary Add-on"
3. Select `browser_extensions/firefox/dist/manifest.json`

---

## Backend Services

### Go Backend

#### Prerequisites
```bash
# Install Go dependencies
cd /workspace/project/TigerWallet/backend/go
go mod download
```

#### Development
```bash
cd /workspace/project/TigerWallet/backend/go/cmd/api

# Run with hot reload (using air)
go install github.com/air-verse/air@latest
air

# Or run directly
go run main.go
```

#### Production Build
```bash
cd /workspace/project/TigerWallet/backend/go/cmd/api

# Build
go build -o tigerwallet-api main.go

# Output: tigerwallet-api

# Run
./tigerwallet-api
```

### C++ Order Matching Engine

#### Build
```bash
cd /workspace/project/TigerWallet/backend/cpp

mkdir -p build && cd build
cmake .. -DCMAKE_BUILD_TYPE=Release
cmake --build . --config Release

# Output: build/order_matching_engine
```

### Rust Security Module

#### Build
```bash
cd /workspace/project/TigerWallet/backend/rust

# Build release
cargo build --release

# Output: target/release/security-tool
```

---

## Production Deployment

### 1. Database Setup (PostgreSQL)

```bash
# Using Docker
docker run -d \
  --name tigerwallet-postgres \
  -e POSTGRES_DB=tigerwallet \
  -e POSTGRES_USER=tigerwallet \
  -e POSTGRES_PASSWORD=your_secure_password \
  -p 5432:5432 \
  -v postgres_data:/var/lib/postgresql/data \
  postgres:14-alpine

# Connect and run migrations
docker exec -it tigerwallet-postgres psql -U tigerwallet -d tigerwallet
```

### 2. Redis Setup

```bash
docker run -d \
  --name tigerwallet-redis \
  -p 6379:6379 \
  -v redis_data:/data \
  redis:7-alpine redis-server --appendonly yes
```

### 3. Environment Variables

Create `/etc/tigerwallet/env`:
```bash
# Server
PORT=3000
GIN_MODE=release

# Database
DB_HOST=postgres.internal
DB_PORT=5432
DB_USER=tigerwallet
DB_PASSWORD=your_secure_password
DB_NAME=tigerwallet
DB_SSL=true

# Redis
REDIS_HOST=redis.internal
REDIS_PORT=6379

# JWT
JWT_SECRET=your_very_long_secure_random_string

# Rate Limiting
RATE_LIMIT=100
```

### 4. Systemd Service (Linux)

Create `/etc/systemd/system/tigerwallet-api.service`:
```ini
[Unit]
Description=TigerWallet API
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=tigerwallet
WorkingDirectory=/opt/tigerwallet/api
EnvironmentFile=/etc/tigerwallet/env
ExecStart=/opt/tigerwallet/api/tigerwallet-api
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable tigerwallet-api
sudo systemctl start tigerwallet-api
```

### 5. Nginx Reverse Proxy

Create `/etc/nginx/sites-available/tigerwallet`:
```nginx
upstream api_backend {
    server 127.0.0.1:3000;
}

upstream web_backend {
    server 127.0.0.1:3001;
}

server {
    listen 80;
    listen [::]:80;
    server_name tigerwallet.com www.tigerwallet.com;

    # Web App
    location / {
        proxy_pass http://web_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }

    # API
    location /api {
        proxy_pass http://api_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # WebSocket
    location /ws {
        proxy_pass http://api_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    # SSL (uncomment after getting certificate)
    # listen 443 ssl http2;
    # ssl_certificate /etc/letsencrypt/live/tigerwallet.com/fullchain.pem;
    # ssl_certificate_key /etc/letsencrypt/live/tigerwallet.com/privkey.pem;
}

# HTTP to HTTPS redirect
server {
    listen 80;
    listen [::]:80;
    server_name tigerwallet.com www.tigerwallet.com;
    return 301 https://$server_name$request_uri;
}
```

Enable and reload:
```bash
sudo ln -s /etc/nginx/sites-available/tigerwallet /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### 6. SSL Certificate (Let's Encrypt)

```bash
# Install certbot
sudo apt-get install certbot python3-certbot-nginx

# Get certificate
sudo certbot --nginx -d tigerwallet.com -d www.tigerwallet.com

# Auto-renewal
sudo certbot renew --dry-run
```

### 7. Docker Compose (Full Stack)

Create `docker-compose.yml`:
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:14-alpine
    environment:
      POSTGRES_DB: tigerwallet
      POSTGRES_USER: tigerwallet
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - internal

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
    networks:
      - internal

  api:
    build: ./backend/go
    ports:
      - "3000:3000"
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis
    networks:
      - internal

  web:
    build: ./frontend/web_nextjs
    ports:
      - "3001:3000"
    environment:
      - API_BASE_URL=http://api:3000
    depends_on:
      - api
    networks:
      - internal

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - api
      - web
    networks:
      - internal

volumes:
  postgres_data:
  redis_data:

networks:
  internal:
    driver: bridge
```

Deploy:
```bash
docker-compose up -d
```

### 8. Kubernetes Deployment

Create Kubernetes manifests in `k8s/`:

**deployment.yaml**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tigerwallet-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: tigerwallet-api
  template:
    metadata:
      labels:
        app: tigerwallet-api
    spec:
      containers:
      - name: api
        image: tigerwallet/api:latest
        ports:
        - containerPort: 3000
        env:
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: tigerwallet-secrets
              key: db-host
```

**service.yaml**:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: tigerwallet-api
spec:
  selector:
    app: tigerwallet-api
  ports:
  - port: 80
    targetPort: 3000
  type: ClusterIP
```

**ingress.yaml**:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tigerwallet-ingress
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - tigerwallet.com
    - www.tigerwallet.com
    secretName: tigerwallet-tls
  rules:
  - host: tigerwallet.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: tigerwallet-web
            port:
              number: 80
```

Deploy:
```bash
kubectl apply -f k8s/
```

### 9. Cloud Providers

#### AWS (ECS)
```bash
# Push images to ECR
aws ecr get-login-password | docker login --username AWS --password-stdin 123456789.dkr.ecr.us-east-1.amazonaws.com

docker tag tigerwallet/api:latest 123456789.dkr.ecr.us-east-1.amazonaws.com/tigerwallet/api:latest
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/tigerwallet/api:latest

# Create ECS cluster
aws ecs create-cluster --cluster-name tigerwallet-prod

# Deploy using task definition
aws ecs register-task-definition --cli-input-json file://task-def.json
aws ecs update-service --cluster tigerwallet-prod --service tigerwallet-api --force-new-deployment
```

#### Google Cloud (GKE)
```bash
# Enable services
gcloud services enable container.googleapis.com cloudbuild.googleapis.com

# Create cluster
gcloud container clusters create tigerwallet-prod \
  --num-nodes=3 \
  --machine-type=e2-standard-2

# Deploy
kubectl apply -f k8s/
```

#### DigitalOcean
```bash
# Create Kubernetes cluster via doctl
doctl kubernetes cluster create tigerwallet-prod --region nyc1

# Deploy
kubectl apply -f k8s/
```

---

## Monitoring & Observability

### Logging
```bash
# Install Loki
helm repo add grafana https://grafana.github.io/helm-charts
helm install loki grafana/loki-stack -n monitoring
```

### Metrics
```bash
# Prometheus
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/bundle.yaml
```

### Dashboard
```bash
# Grafana
helm install grafana grafana/grafana -n monitoring \
  --set adminPassword=admin \
  --set service.type=LoadBalancer
```

---

## Security Hardening

1. **Firewall**
```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 22/tcp  # SSH
sudo ufw enable
```

2. **Fail2ban**
```bash
sudo apt-get install fail2ban
sudo systemctl enable fail2ban
```

3. **Database Encryption**
- Enable encryption at rest
- Use SSL for connections
- Rotate credentials regularly

---

## Quick Start Commands

```bash
# Clone and build everything
git clone https://github.com/meghlabd275-byte/TigerWallet.git
cd TigerWallet

# Backend
cd backend/go/cmd/api && go build -o tigerwallet-api .

# Frontend Web
cd ../../frontend/web_nextjs && npm install && npm run build

# Mobile
cd ../../user_wallet/android && ./gradlew assembleRelease

# Desktop
cd ../../desktop_app/tauri && cargo build --release

# Run
./backend/go/cmd/api/tigerwallet-api &
npm start &
```

---

## Support

For issues and contributions, please open an issue on GitHub.
