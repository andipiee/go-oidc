# Deployment Guide

## Docker Deployment

### Quick Start

```bash
git clone https://github.com/andipiee/go-oidc.git
cd go-oidc
docker-compose up -d
```

The default `docker-compose.yml` starts the application and an Adminer UI (port 8081). It expects an external PostgreSQL instance matching `configs/config.yaml`.

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

- Go 1.24+
- PostgreSQL 15+
- Nginx (optional, for reverse proxy)

### Steps

1. **Build the binary**

```bash
go build -o server ./cmd/server
```

2. **Set up the database**

```bash
createdb -U postgres oauth2
psql -U postgres -d oauth2 -f migrations/001_init.sql
```

3. **Configure the application**

Edit `configs/config.yaml` with production values. At minimum:
- Set a strong database password
- Set `sslmode: "require"` for the database
- Change the admin password
- Set the `issuer` to your public URL (e.g., `https://auth.example.com`)

4. **Run the server** (from the project root directory)

```bash
./server
```

## Reverse Proxy with Nginx

Example Nginx configuration:

```nginx
server {
    listen 443 ssl http2;
    server_name auth.example.com;

    ssl_certificate /etc/letsencrypt/live/auth.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/auth.example.com/privkey.pem;

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

Logs are written to stdout in the format `[STATUS] METHOD PATH DURATION`. In production, use Docker's logging drivers:

```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

## Backup and Recovery

### Backup database

```bash
pg_dump -U postgres oauth2 > backup.sql
```

### Restore database

```bash
psql -U postgres oauth2 < backup.sql
```
