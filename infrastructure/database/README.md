# TigerWallet Production Database Configuration
# PostgreSQL + Redis Production Setup

## PostgreSQL Configuration

```yaml
# postgresql.conf
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 4MB
min_wal_size = 1GB
max_wal_size = 4GB
max_worker_processes = 8
max_parallel_workers_per_gather = 4
max_parallel_workers = 8
max_parallel_maintenance_workers = 4
wal_level = replica
max_wal_senders = 10
wal_keep_size = 1GB
hot_standby = on
```

## Redis Configuration

```yaml
# redis.conf
maxmemory 4gb
maxmemory-policy allkeys-lru
save 900 1
save 300 10
save 60 10000
appendonly yes
appendfsync everysec
tcp-keepalive 300
```

## Docker Compose

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: tigerwallet-postgres
    environment:
      POSTGRES_DB: tigerwallet
      POSTGRES_USER: tigeruser
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./postgresql.conf:/etc/postgresql/postgresql.conf
    ports:
      - "5432:5432"
    networks:
      - tigerwallet-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U tigeruser"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: tigerwallet-redis
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
      - ./redis.conf:/usr/local/etc/redis/redis.conf
    ports:
      - "6379:6379"
    networks:
      - tigerwallet-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis-cluster:
    image: redis:7-alpine
    container_name: tigerwallet-redis-cluster
    command: |
      redis-server --cluster-enabled yes --cluster-config-file nodes.conf
      --cluster-node-timeout 5000 --appendonly yes
    volumes:
      - redis_cluster_data:/data
    ports:
      - "7001-7006:7001-7006"
    networks:
      - tigerwallet-network

  pgadmin:
    image: dpage/pgadmin4
    container_name: tigerwallet-pgadmin
    environment:
      PGADMIN_DEFAULT_EMAIL: admin@tigerwallet.io
      PGADMIN_DEFAULT_PASSWORD: ${PGADMIN_PASSWORD}
    volumes:
      - pgadmin_data:/var/lib/pgadmin
    ports:
      - "8080:80"
    networks:
      - tigerwallet-network
    depends_on:
      - postgres

volumes:
  postgres_data:
  redis_data:
  redis_cluster_data:
  pgadmin_data:

networks:
  tigerwallet-network:
    driver: bridge

# Environment Variables (.env)
# DB_PASSWORD=your_secure_password
# REDIS_PASSWORD=your_redis_password
# PGADMIN_PASSWORD=your_pgadmin_password
```

## Connection Pool (PgBouncer)

```yaml
# pgbouncer.ini
[databases]
tigerwallet = host=postgres port=5432 dbname=tigerwallet

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 25
min_pool_size = 10
reserve_pool_size = 5
reserve_pool_timeout = 5
max_db_connections = 100
max_user_connections = 100
log_connections = 0
log_disconnections = 0
log_pooler_errors = 1
```

## Database Schema

```sql
-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    wallet_address VARCHAR(66) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    kyc_level INTEGER DEFAULT 0,
    status VARCHAR(50) DEFAULT 'active'
);

-- Wallets table
CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    chain_id VARCHAR(50) NOT NULL,
    address VARCHAR(66) NOT NULL,
    private_key_encrypted BYTEA,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Transactions table
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    hash VARCHAR(66) UNIQUE NOT NULL,
    from_address VARCHAR(66),
    to_address VARCHAR(66),
    amount DECIMAL(78, 0),
    token_address VARCHAR(66),
    chain_id VARCHAR(50),
    status VARCHAR(50),
    type VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Listings table
CREATE TABLE listings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_name VARCHAR(255) NOT NULL,
    token_symbol VARCHAR(50) NOT NULL,
    token_address VARCHAR(66) NOT NULL,
    chain_id VARCHAR(50) NOT NULL,
    tier INTEGER,
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_wallet ON users(wallet_address);
CREATE INDEX idx_wallets_user ON wallets(user_id);
CREATE INDEX idx_transactions_user ON transactions(user_id);
CREATE INDEX idx_transactions_hash ON transactions(hash);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_listings_status ON listings(status);

-- Partitioning for transactions (by month)
CREATE TABLE transactions (
    LIKE transactions_template INCLUDING ALL
) PARTITION BY RANGE (created_at);

CREATE TABLE transactions_2024_01 PARTITION OF transactions
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

## Backup Strategy

```bash
#!/bin/bash
# backup.sh
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups/postgres"

# Full backup
pg_dump -h postgres -U tigeruser -Fc tigerwallet > $BACKUP_DIR/full_$DATE.dump

# Incremental backup (WAL)
pg_basebackup -h postgres -U replicator -D /backups/wal -X stream -P

# Retention
find $BACKUP_DIR -name "full_*.dump" -mtime +7 -delete
```

## Monitoring Queries

```sql
-- Slow queries
SELECT query, calls, mean_time, total_time 
FROM pg_stat_statements 
ORDER BY mean_time DESC LIMIT 20;

-- Connection status
SELECT datname, numbackends, xact_commit, xact_rollback, blks_read, blks_hit
FROM pg_stat_database WHERE datname = 'tigerwallet';

-- Table sizes
SELECT relname, pg_size_pretty(pg_total_relation_size(relid))
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC;
```
