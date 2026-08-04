# TigerWallet White Label - Complete Installation Guide

## Table of Contents
1. [System Requirements](#system-requirements)
2. [Prerequisites](#prerequisites)
3. [Database Setup](#database-setup)
4. [Redis Setup](#redis-setup)
5. [Cloud Deployment Options](#cloud-deployment-options)
6. [Service Installation](#service-installation)
7. [Environment Configuration](#environment-configuration)
8. [SSL/HTTPS Setup](#sslhttps-setup)
9. [Verification](#verification)
10. [Troubleshooting](#troubleshooting)

---

## System Requirements

### Minimum Requirements
- **CPU**: 4 cores
- **RAM**: 8 GB
- **Storage**: 100 GB SSD
- **Network**: 100 Mbps bandwidth

### Recommended Requirements
- **CPU**: 8+ cores
- **RAM**: 16+ GB
- **Storage**: 500 GB SSD
- **Network**: 1 Gbps bandwidth

### Supported Operating Systems
- Ubuntu 20.04 LTS / 22.04 LTS
- Debian 11+
- CentOS 8+
- macOS (development only)

---

## Prerequisites

### 1. Install Go (for Go services)
```bash
# Download and install Go 1.21+
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
```

### 2. Install Rust (for Rust services)
```bash
# Install Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source ~/.cargo/env
```

### 3. Install PostgreSQL 14+
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install postgresql postgresql-contrib

# Start PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database
sudo -u postgres psql
CREATE DATABASE tigerwallet;
CREATE USER tigerwallet WITH PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE tigerwallet TO tigerwallet;
```

### 4. Install Redis
```bash
# Ubuntu/Debian
sudo apt install redis-server

# Start Redis
sudo systemctl start redis-server
sudo systemctl enable redis-server

# Secure Redis (optional)
sudo redis-cli CONFIG SET requirepass your_redis_password
```

### 5. Install Node.js (for frontend)
```bash
# Install Node.js 18+
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install nodejs

# Install pnpm
npm install -g pnpm
```

---

## Database Setup

### PostgreSQL Configuration

1. Edit PostgreSQL config:
```bash
sudo nano /etc/postgresql/14/main/postgresql.conf
```

2. Update these settings:
```properties
max_connections = 200
shared_buffers = 2GB
effective_cache_size = 6GB
maintenance_work_mem = 512MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 10MB
min_wal_size = 1GB
max_wal_size = 4GB
```

3. Restart PostgreSQL:
```bash
sudo systemctl restart postgresql
```

### Create Database Schema
```bash
# Connect to database
sudo -u postgres psql -d tigerwallet

# Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
```

---

## Redis Setup

### Redis Configuration
```bash
sudo nano /etc/redis/redis.conf
```

Update these settings:
```properties
maxmemory 2gb
maxmemory-policy allkeys-lru
save 900 1
save 300 10
save 60 10000
```

Restart Redis:
```bash
sudo systemctl restart redis-server
```

---

## Cloud Deployment Options

### Option 1: AWS Deployment

#### 1. Launch EC2 Instance
- Instance Type: t3.large or larger
- Ubuntu 22.04 LTS
- Security Group: Open ports 80, 443, 22, 8080-8100

#### 2. Install Dependencies
```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
sudo apt install docker.io docker-compose
sudo systemctl start docker
sudo systemctl enable docker

# Install PostgreSQL (or use RDS)
# Install Redis (or use ElastiCache)
```

#### 3. Deploy Services
```bash
# Clone repository
git clone https://github.com/meghlabd275-byte/TigerWallet.git
cd TigerWallet

# Start all services
docker-compose up -d
```

#### AWS Services Recommended:
- RDS PostgreSQL
- ElastiCache Redis
- Route 53 for DNS
- ACM for SSL certificates
- CloudWatch for monitoring

---

### Option 2: Google Cloud Platform

#### 1. Create VM Instance
- Machine Type: e2-standard-4
- OS: Ubuntu 22.04 LTS

#### 2. Set up Cloud SQL
```bash
# Create Cloud SQL instance
gcloud sql instances create tigerwallet-db \
    --database-version=POSTGRES_14 \
    --tier=db-custom-2-4096 \
    --region=us-central1
```

#### 3. Set up Memorystore (Redis)
```bash
# Create Redis instance
gcloud redis instances create tigerwallet-redis \
    --size=2 \
    --region=us-central1
```

---

### Option 3: Azure Deployment

#### 1. Create Virtual Machine
- Size: Standard_D4s_v3
- OS: Ubuntu 22.04 LTS

#### 2. Azure Services
- Azure Database for PostgreSQL
- Azure Cache for Redis
- Azure DNS for domain

---

### Option 4: DigitalOcean

#### 1. Create Droplet
- Size: Premium Intel (4GB RAM)
- OS: Ubuntu 22.04 LTS

#### 2. Managed Databases
- PostgreSQL (4GB RAM)
- Redis (1GB)

---

## Service Installation

### All Services Overview

| Service | Port | Language | Description |
|---------|------|----------|-------------|
| White Label Marketplace | 8085 | Go | Partner marketplace |
| White Label Templates | 8086 | Rust | Template system |
| Auto-Approval Workflow | 8087 | Go | KYC/WL approval |
| Multi-Level White Label | 8088 | Go | Hierarchy management |
| Advanced Analytics AI | 8089 | Rust | AI analytics |
| Trading Charts | 8090 | Rust | TradingView charts |
| Master Admin Management | 8091 | Go | Master admin portal |
| Self-Hosted Master Wallet | 8092 | Rust | Master wallet hosting |
| White Label Admin | 8093 | Go | WL admin portal |

### Installation Steps

#### 1. Clone and Build
```bash
# Clone repository
git clone https://github.com/meghlabd275-byte/TigerWallet.git
cd TigerWallet

# Build all Go services
cd white_label_marketplace/go
go build -o tiger-wl-marketplace main.go

cd ../../auto_approval_workflow/go
go build -o tiger-auto-approval main.go

# Build all Rust services
cd ../../white_label_analytics_ai/rust
cargo build --release

# (Repeat for other services)
```

#### 2. Environment Configuration
Create `.env` file for each service:
```bash
# Example for white_label_marketplace
cat > white_label_marketplace/go/.env << EOF
PORT=8085
DATABASE_URL=postgres://user:password@host:5432/tigerwallet
REDIS_URL=redis://password@host:6379
JWT_SECRET=your-super-secret-jwt-key
LOG_LEVEL=info
EOF
```

#### 3. Run Services
```bash
# Using systemd (recommended)
sudo tee /etc/systemd/system/tiger-wl-marketplace.service << EOF
[Unit]
Description=TigerWallet White Label Marketplace
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/TigerWallet/white_label_marketplace/go
ExecStart=/home/ubuntu/TigerWallet/white_label_marketplace/go/tiger-wl-marketplace
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable tiger-wl-marketplace
sudo systemctl start tiger-wl-marketplace
```

#### 4. Using Docker
```bash
# Create docker-compose.yml
cat > docker-compose.yml << EOF
version: '3.8'

services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: tigerwallet
      POSTGRES_USER: tigerwallet
      POSTGRES_PASSWORD: password
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass password
    ports:
      - "6379:6379"

  wl-marketplace:
    build: ./white_label_marketplace/go
    ports:
      - "8085:8085"
    environment:
      - DATABASE_URL=postgres://tigerwallet:password@postgres:5432/tigerwallet
      - REDIS_URL=redis://password@redis:6379

volumes:
  pgdata:
EOF

docker-compose up -d
```

---

## Environment Configuration

### Required Environment Variables

```bash
# Database
DATABASE_URL=postgres://user:password@host:5432/tigerwallet

# Redis
REDIS_URL=redis://password@host:6379

# Security
JWT_SECRET=your-256-bit-secret-key
BCRYPT_ROUNDS=12

# API Keys (add your own)
ALCHEMY_API_KEY=your_alchemy_key
INFURA_API_KEY=your_infura_key
COINMARKETCAP_API_KEY=your_coinmarketcap_key

# Email (SMTP)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your_email@gmail.com
SMTP_PASSWORD=your_app_password

# SMS (Twilio)
TWILIO_ACCOUNT_SID=your_sid
TWILIO_AUTH_TOKEN=your_token
TWILIO_PHONE_NUMBER=+1234567890
```

---

## SSL/HTTPS Setup

### Using Let's Encrypt (Free)

```bash
# Install Certbot
sudo apt install certbot python3-certbot-nginx

# Generate certificate
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com

# Auto-renewal
sudo certbot renew --dry-run
```

### Using Cloudflare

1. Add domain to Cloudflare
2. Update nameservers
3. Create Origin Server certificate
4. Install certificate on server

---

## Verification

### 1. Check Service Status
```bash
# Check all services
sudo systemctl status tiger-*

# Or check with docker
docker-compose ps
```

### 2. Health Checks
```bash
# Check each service
curl http://localhost:8085/health
curl http://localhost:8086/health
curl http://localhost:8087/health
# ... etc
```

### 3. Check Logs
```bash
# View logs
sudo journalctl -u tiger-wl-marketplace -f

# Or docker logs
docker-compose logs -f
```

---

## Troubleshooting

### Common Issues

#### 1. Database Connection Failed
```bash
# Check PostgreSQL status
sudo systemctl status postgresql

# Test connection
psql -h localhost -U tigerwallet -d tigerwallet
```

#### 2. Redis Connection Failed
```bash
# Check Redis status
sudo systemctl status redis-server

# Test connection
redis-cli -h localhost -a your_password
```

#### 3. Port Already in Use
```bash
# Find process using port
sudo lsof -i :8085

# Kill process
sudo kill -9 <PID>
```

#### 4. Out of Memory
```bash
# Check memory usage
free -h

# Increase swap
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

---

## Next Steps

After installation, see:
- [API Documentation](./API_DOCS.md)
- [Admin Panel Guide](./ADMIN_GUIDE.md)
- [White Label Setup Guide](./WHITE_LABEL_SETUP.md)
- [Mobile App Build Guide](./MOBILE_BUILD.md)

---

## Support

- Documentation: https://docs.tigerwallet.io
- GitHub Issues: https://github.com/meghlabd275-byte/TigerWallet/issues
- Email: support@tigerwallet.io
