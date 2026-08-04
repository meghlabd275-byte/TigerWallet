# TigerWallet Cloud Installation & Deployment Guide

## Complete Production Deployment Instructions

---

## 📋 Table of Contents

1. [Prerequisites](#prerequisites)
2. [Server Setup](#server-setup)
3. [Database Configuration](#database-configuration)
4. [Backend Services](#backend-services)
5. [Frontend Deployment](#frontend-deployment)
6. [Domain & SSL](#domain--ssl)
7. [Launch Checklist](#launch-checklist)

---

## 1. Prerequisites

### System Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 8 cores | 16+ cores |
| RAM | 32GB | 64GB+ |
| Storage | 500GB SSD | 1TB NVMe |
| Bandwidth | 1Gbps | 10Gbps |

### Software Requirements

| Software | Version |
|----------|---------|
| Ubuntu | 22.04 LTS |
| Go | 1.21+ |
| Node.js | 18+ |
| PostgreSQL | 14+ |
| Redis | 7.0+ |
| Docker | 24.0+ |
| Nginx | 1.24+ |

---

## 2. Server Setup

### Step 2.1: Create Cloud Servers

**Recommended Architecture:**

```
┌─────────────────────────────────────────────────────────┐
│                    Load Balancer                         │
│                    (AWS ALB / CloudFlare)                │
└─────────────────────────────────────────────────────────┘
                           │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
   ┌─────────┐       ┌─────────┐        ┌─────────┐
   │ Web App │       │ Web App │        │ Web App │
   │ Server  │       │ Server  │        │ Server  │
   └─────────┘       └─────────┘        └─────────┘
        │                   │                   │
        └───────────────────┼───────────────────┘
                           ▼
                ┌─────────────────────┐
                │   API Gateway      │
                │   (Nginx/HAProxy) │
                └─────────────────────┘
                           │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
   ┌─────────┐       ┌─────────┐        ┌─────────┐
   │   Go     │       │   Go    │        │   Go    │
   │ Services │       │ Services │        │ Services │
   └─────────┘       └─────────┘        └─────────┘
        │                   │                   │
        ▼                   ▼                   ▼
   ┌─────────┐       ┌─────────┐        ┌─────────┐
   │PostgreSQL│       │  Redis  │        │   C++   │
   │ Cluster  │       │ Cluster │        │  Engine │
   └─────────┘       └─────────┘        └─────────┘
```

### Step 2.2: Initial Server Setup

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install essential packages
sudo apt install -y curl wget git vim htop net-tools ufw

# Disable firewall temporarily for installation
sudo ufw disable

# Create deployment user
sudo useradd -m -s /bin/bash deploy
sudo usermod -aG docker deploy
sudo usermod -aG sudo deploy
```

### Step 2.3: Install Docker

```bash
# Install Docker
curl -fsSL https://get.docker.com | sh

# Start Docker
sudo systemctl start docker
sudo systemctl enable docker

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.24.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Verify installation
docker --version
docker-compose --version
```

### Step 2.4: Install Node.js

```bash
# Install Node.js 18
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs

# Verify
node --version
npm --version
```

### Step 2.5: Install Go

```bash
# Download Go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz

# Extract
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify
go version
```

---

## 3. Database Configuration

### Step 3.1: Install PostgreSQL

```bash
# Install PostgreSQL
sudo apt install -y postgresql postgresql-contrib

# Start PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Login as postgres
sudo -u postgres psql
```

### Step 3.2: Create Database and User

```sql
-- Create database
CREATE DATABASE tigerwallet;

-- Create user
CREATE USER tigerwallet WITH PASSWORD 'your_secure_password';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE tigerwallet TO tigerwallet;

-- Connect to database
\c tigerwallet

-- Grant schema privileges
GRANT ALL ON SCHEMA public TO tigerwallet;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO tigerwallet;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO tigerwallet;

-- Exit
\q
```

### Step 3.3: Run Database Schema

```bash
# Navigate to project
cd /opt/TigerWallet

# Run schema
sudo -u postgres psql -d tigerwallet -f database/postgres/schema.sql
```

### Step 3.4: Install Redis

```bash
# Install Redis
sudo apt install -y redis-server

# Configure Redis
sudo cp /etc/redis/redis.conf /etc/redis/redis.conf.bak

# Edit Redis config
sudo vim /etc/redis/redis.conf

# Make these changes:
# bind 127.0.0.1 ::1
# requirepass your_redis_password
# maxmemory 4gb
# maxmemory-policy allkeys-lru

# Restart Redis
sudo systemctl restart redis-server
sudo systemctl enable redis-server
```

---

## 4. Backend Services

### Step 4.1: Clone Repository

```bash
# Navigate to /opt
cd /opt

# Clone repository
sudo git clone https://github.com/meghlabd275-byte/TigerWallet.git
cd TigerWallet

# Set ownership
sudo chown -R deploy:deploy /opt/TigerWallet
```

### Step 4.2: Environment Configuration

```bash
# Create environment file
sudo cp .env.example .env
sudo vim .env
```

**Add these configurations:**

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=tigerwallet
DB_PASSWORD=your_secure_password
DB_NAME=tigerwallet

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password

# JWT
JWT_SECRET=your_very_secure_jwt_secret_min_32_chars
JWT_EXPIRY=86400

# Encryption
ENCRYPTION_KEY=your_32_char_encryption_key

# API Keys (Add your own)
BINANCE_API_KEY=your_binance_api_key
BINANCE_SECRET_KEY=your_binance_secret_key
COINBASE_API_KEY=your_coinbase_api_key

# Admin
ADMIN_EMAIL=admin@tigerwallet.com
ADMIN_PASSWORD=your_secure_admin_password

# Server
SERVER_PORT=8080
NODE_ENV=production
```

### Step 4.3: Build Backend Services

```bash
# Build API Gateway
cd /opt/TigerWallet/api_gateway
go build -o api-gateway .

# Build Listing Service
cd /opt/TigerWallet/go/listing_service
go build -o listing-service .

# Build Fiat On-Ramp
cd /opt/TigerWallet/go/fiat_onramp
go build -o fiat-onramp .

# Build Fiat Off-Ramp
cd /opt/TigerWallet/go/fiat_offramp
go build -o fiat-offramp .

# Build Bot Platform
cd /opt/TigerWallet/mm_bot_platform
go build -o bot-platform .
```

### Step 4.4: Build C++ Components

```bash
# Install C++ build tools
sudo apt install -y build-essential cmake libssl-dev libcurl4-openssl-dev

# Build Logo Upload Service
cd /opt/TigerWallet/cpp/logo_upload_service
mkdir build && cd build
cmake ..
make -j$(nproc)

# Build DEX Connectors
cd /opt/TigerWallet/cpp/dex_connectors
mkdir build && cd build
cmake ..
make -j$(nproc)
```

### Step 4.5: Build Rust Components

```bash
# Install Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source ~/.cargo/env

# Build Trading Bots
cd /opt/TigerWallet/rust/trading_bots
cargo build --release

# Build Auto Optimizer
cd /opt/TigerWallet/rust/trading_bots/src
cargo build --release
```

### Step 4.6: Create Systemd Services

```bash
# Create systemd service for API Gateway
sudo vim /etc/systemd/system/tigerwallet-api.service
```

**Add this content:**

```ini
[Unit]
Description=TigerWallet API Gateway
After=network.target postgresql.service redis-server.service

[Service]
Type=simple
User=deploy
WorkingDirectory=/opt/TigerWallet/api_gateway
ExecStart=/opt/TigerWallet/api_gateway/api-gateway
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

```bash
# Create similar services for other services
# listing-service, fiat-onramp, fiat-offramp, bot-platform

# Reload systemd
sudo systemctl daemon-reload

# Enable services
sudo systemctl enable tigerwallet-api
sudo systemctl enable tigerwallet-listing
sudo systemctl enable tigerwallet-fiat-onramp
sudo systemctl enable tigerwallet-fiat-offramp
sudo systemctl enable tigerwallet-bot

# Start services
sudo systemctl start tigerwallet-api
```

---

## 5. Frontend Deployment

### Step 5.1: Build Frontend

```bash
# Navigate to frontend
cd /opt/TigerWallet/frontend/web_nextjs

# Install dependencies
npm install

# Build for production
npm run build
```

### Step 5.2: Configure Nginx

```bash
# Install Nginx
sudo apt install -y nginx

# Create Nginx config
sudo vim /etc/nginx/sites-available/tigerwallet
```

**Add this configuration:**

```nginx
server {
    listen 80;
    server_name your-domain.com www.your-domain.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com www.your-domain.com;

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Frontend
    root /opt/TigerWallet/frontend/web_nextjs/out;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    # API Proxy
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # WebSocket Support
    location /ws {
        proxy_pass http://localhost:8095;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }

    # Static files caching
    location ~* \.(jpg|jpeg|png|gif|ico|css|js|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/tigerwallet /etc/nginx/sites-enabled/

# Test Nginx
sudo nginx -t

# Restart Nginx
sudo systemctl restart nginx
```

---

## 6. Domain & SSL

### Step 6.1: Point Domain to Server

1. Go to your domain registrar (GoDaddy, Namecheap, etc.)
2. Create A Record:
   - Type: A
   - Name: @ or your subdomain
   - Value: Your server IP
3. Create CNAME:
   - Type: CNAME
   - Name: www
   - Value: your-domain.com

### Step 6.2: Install SSL Certificate

```bash
# Install Certbot
sudo apt install -y certbot python3-certbot-nginx

# Get SSL certificate
sudo certbot --nginx -d your-domain.com -d www.your-domain.com

# Auto-renewal
sudo certbot renew --dry-run
```

---

## 7. Launch Checklist

### Pre-Launch Verification

```bash
# Check all services
sudo systemctl status tigerwallet-api
sudo systemctl status tigerwallet-listing
sudo systemctl status tigerwallet-fiat-onramp
sudo systemctl status tigerwallet-fiat-offramp
sudo systemctl status tigerwallet-bot

# Test database connection
psql -h localhost -U tigerwallet -d tigerwallet -c "SELECT 1;"

# Test Redis connection
redis-cli -a your_redis_password ping

# Test API endpoint
curl -k https://localhost/api/health

# Test frontend
curl -k https://your-domain.com
```

### DNS Verification

```bash
# Check DNS propagation
dig your-domain.com
dig www.your-domain.com
```

### SSL Verification

```bash
# Check SSL certificate
curl -kI https://your-domain.com | grep -i ssl
```

### Performance Testing

```bash
# Test with hey (install first)
hey -n 1000 -c 10 https://your-domain.com/

# Test with ab
ab -n 1000 -c 10 https://your-domain.com/
```

---

## 🚀 Launch Commands

### Start All Services

```bash
# Start backend services
sudo systemctl start tigerwallet-api
sudo systemctl start tigerwallet-listing
sudo systemctl start tigerwallet-fiat-onramp
sudo systemctl start tigerwallet-fiat-offramp
sudo systemctl start tigerwallet-bot

# Restart Nginx
sudo systemctl restart nginx
```

### Stop All Services

```bash
sudo systemctl stop tigerwallet-api
sudo systemctl stop tigerwallet-listing
sudo systemctl stop tigerwallet-fiat-onramp
sudo systemctl stop tigerwallet-fiat-offramp
sudo systemctl stop tigerwallet-bot
sudo systemctl stop nginx
```

### View Logs

```bash
# View API logs
sudo journalctl -u tigerwallet-api -f

# View Nginx logs
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log
```

---

## 📞 Post-Launch Monitoring

### Health Check Endpoints

| Service | Endpoint |
|---------|-----------|
| API Gateway | `https://your-domain.com/api/health` |
| Listing Service | `http://localhost:8097/health` |
| Fiat On-Ramp | `http://localhost:8451/health` |
| Fiat Off-Ramp | `http://localhost:8452/health` |

### Monitoring URLs

| Service | URL |
|---------|-----|
| Frontend | `https://your-domain.com` |
| Admin Panel | `https://your-domain.com/admin` |
| API Docs | `https://your-domain.com/api/docs` |

---

## ⚠️ Important Security Notes

1. **Change all default passwords**
2. **Use strong JWT_SECRET** (minimum 32 characters)
3. **Enable firewall** after installation:
   ```bash
   sudo ufw allow 22/tcp   # SSH
   sudo ufw allow 80/tcp   # HTTP
   sudo ufw allow 443/tcp  # HTTPS
   sudo ufw enable
   ```
4. **Regular backups** of database
5. **Monitor logs** for suspicious activity

---

## 🎉 Your TigerWallet is Ready!

After completing all steps:

1. **Visit:** `https://your-domain.com`
2. **Admin Login:** `https://your-domain.com/admin`
3. **Default Admin:** `admin@tigerwallet.com`
4. **Change password** immediately after first login!

---

For support, check the repository at: https://github.com/meghlabd275-byte/TigerWallet
