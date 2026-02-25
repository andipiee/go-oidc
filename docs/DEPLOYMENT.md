# Deployment Guide

## Docker Deployment

### Quick Start

```bash
# Clone and start
git clone https://github.com/andipiee/go-oidc.git
cd go-oidc
docker-compose up -d
```

### Production Deployment

#### 1. Build the image

```bash
docker build -t oauth2-server -f docker/Dockerfile .
```

#### 2. Run with Docker Compose

Create `docker-compose.prod.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: oauth2
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: oauth2
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: always

  oauth2:
    image: oauth2-server:latest
    ports:
      - "8080:8080"
    environment:
      - CONFIG_PATH=/app/configs/config.yaml
    volumes:
      - ./configs:/app/configs:ro
    depends_on:
      - postgres
    restart: always

volumes:
  postgres_data:
```

#### 3. Start services

```bash
docker-compose -f docker-compose.prod.yml up -d
```

## Manual Deployment

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Nginx (optional, for reverse proxy)

### Steps

1. **Build the binary**

```bash
go build -o server ./cmd/server
```

2. **Set up the database**

```bash
psql -U postgres -d oauth2 -f migrations/001_init.sql
```

3. **Configure the application**

Edit `configs/config.yaml` with production values.

4. **Run the server**

```bash
./server
```

## Reverse Proxy with Nginx

Example Nginx configuration:

```nginx
server {
    listen 443 ssl http2;
    server_name oauth.example.com;

    ssl_certificate /etc/letsencrypt/live/oauth.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/oauth.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Health Check

```bash
curl http://localhost:8080/health
```

Response:
```json
{"status": "ok"}
```

## Logging

Logs are written to stdout. In production, use Docker's logging drivers:

```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

## Monitoring

### Prometheus metrics (coming soon)

```bash
curl http://localhost:8080/metrics
```

## Backup and Recovery

### Backup database

```bash
docker exec oauth2-postgres pg_dump -U postgres oauth2 > backup.sql
```

### Restore database

```bash
docker exec -i oauth2-postgres psql -U postgres oauth2 < backup.sql
```
