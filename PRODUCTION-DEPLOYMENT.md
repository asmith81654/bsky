# 🚀 Bluesky Automation Platform - Production Deployment Guide

This guide provides comprehensive instructions for deploying the Bluesky Automation Platform to a production environment.

## 📋 Prerequisites

### System Requirements
- **OS**: Ubuntu 20.04+ / CentOS 8+ / RHEL 8+
- **RAM**: Minimum 8GB, Recommended 16GB+
- **CPU**: Minimum 4 cores, Recommended 8+ cores
- **Storage**: Minimum 100GB SSD
- **Network**: Static IP address, Domain name configured

### Software Requirements
- Docker 20.10+
- Docker Compose 2.0+
- Git
- curl
- openssl

## 🔧 Pre-Deployment Setup

### 1. Server Preparation

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Logout and login again for group changes to take effect
```

### 2. Clone Repository

```bash
git clone https://github.com/yourusername/bsky-automation.git
cd bsky-automation
```

### 3. SSL Certificate Setup

#### Option A: Let's Encrypt (Recommended)
```bash
# Install Certbot
sudo apt install certbot

# Generate certificates
sudo certbot certonly --standalone -d yourdomain.com -d api.yourdomain.com -d dashboard.yourdomain.com

# Copy certificates
sudo cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem configs/ssl/cert.pem
sudo cp /etc/letsencrypt/live/yourdomain.com/privkey.pem configs/ssl/key.pem
sudo chown $USER:$USER configs/ssl/*.pem
```

#### Option B: Self-Signed (Development Only)
```bash
# Generate self-signed certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout configs/ssl/key.pem \
  -out configs/ssl/cert.pem \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=yourdomain.com"
```

### 4. Environment Configuration

```bash
# Copy and edit production environment file
cp .env.prod .env.prod.local

# Edit the file with your actual values
nano .env.prod.local
```

**Important**: Update these critical values in `.env.prod.local`:
- `POSTGRES_PASSWORD`: Strong database password
- `REDIS_PASSWORD`: Strong Redis password  
- `JWT_SECRET`: 32+ character secret key
- `GRAFANA_PASSWORD`: Grafana admin password
- Domain names and email settings

## 🚀 Deployment Process

### 1. Run Deployment Script

```bash
# Make script executable
chmod +x scripts/deploy-prod.sh

# Run deployment
./scripts/deploy-prod.sh
```

### 2. Manual Deployment (Alternative)

```bash
# Create necessary directories
mkdir -p backups/{postgres,redis} logs/{nginx,account-manager,proxy-manager,api-gateway}

# Deploy services
docker-compose -f docker-compose.prod.yml --env-file .env.prod.local up -d

# Check service status
docker-compose -f docker-compose.prod.yml ps
```

## 🔍 Post-Deployment Verification

### 1. Health Checks

```bash
# Check all services are running
docker-compose -f docker-compose.prod.yml ps

# Test API endpoints
curl -k https://api.yourdomain.com/health
curl -k https://dashboard.yourdomain.com/health

# Check logs
docker-compose -f docker-compose.prod.yml logs -f
```

### 2. Service Verification

```bash
# Test Account Manager
curl -k -X POST https://api.yourdomain.com/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{"handle":"test.bsky.social","password":"test","host":"bsky.social"}'

# Test Proxy Manager
curl -k https://api.yourdomain.com/api/v1/proxies

# Access monitoring
# Grafana: https://dashboard.yourdomain.com/grafana
# Prometheus: https://dashboard.yourdomain.com/prometheus
```

## 📊 Monitoring Setup

### 1. Grafana Configuration

1. Access Grafana at `https://dashboard.yourdomain.com/grafana`
2. Login with admin/[GRAFANA_PASSWORD]
3. Import dashboards from `configs/grafana/dashboards/`
4. Configure alerts and notifications

### 2. Prometheus Targets

Verify all targets are up in Prometheus:
- API Gateway: `:8000/metrics`
- Account Manager: `:8001/metrics`
- Proxy Manager: `:8002/metrics`
- PostgreSQL: `:9187`
- Redis: `:9121`
- Nginx: `:9113`

## 🔒 Security Hardening

### 1. Firewall Configuration

```bash
# Configure UFW
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

### 2. SSL/TLS Configuration

- Ensure SSL certificates are properly configured
- Verify HTTPS redirects are working
- Check SSL rating at SSL Labs

### 3. Database Security

```bash
# Secure PostgreSQL
docker-compose -f docker-compose.prod.yml exec postgres psql -U postgres -c "ALTER USER postgres PASSWORD 'new_strong_password';"
```

## 💾 Backup Strategy

### 1. Automated Backups

```bash
# Setup daily backups via cron
crontab -e

# Add this line for daily backups at 2 AM
0 2 * * * /path/to/bsky-automation/scripts/backup.sh
```

### 2. Manual Backup

```bash
# Create immediate backup
./scripts/backup.sh

# Verify backup
./scripts/backup.sh verify /path/to/backup
```

## 🔄 Maintenance

### 1. Updates

```bash
# Pull latest changes
git pull origin main

# Rebuild and restart services
docker-compose -f docker-compose.prod.yml build --no-cache
docker-compose -f docker-compose.prod.yml up -d
```

### 2. Log Rotation

```bash
# Setup logrotate for Docker logs
sudo nano /etc/logrotate.d/docker

# Add configuration for log rotation
```

## 🚨 Troubleshooting

### Common Issues

1. **Services not starting**: Check logs with `docker-compose logs`
2. **SSL errors**: Verify certificate paths and permissions
3. **Database connection issues**: Check environment variables
4. **High memory usage**: Monitor with `docker stats`

### Emergency Procedures

```bash
# Stop all services
docker-compose -f docker-compose.prod.yml down

# Restore from backup
./scripts/restore.sh /path/to/backup.tar.gz

# Restart services
docker-compose -f docker-compose.prod.yml up -d
```

## 📞 Support

For production support:
- Check logs: `docker-compose -f docker-compose.prod.yml logs`
- Monitor metrics in Grafana
- Review system resources with `htop` and `df -h`

## 🎯 Performance Optimization

### 1. Database Tuning
- Configure PostgreSQL settings in `configs/postgres/postgresql.conf`
- Monitor query performance
- Set up connection pooling

### 2. Redis Optimization
- Configure Redis memory settings
- Enable persistence if needed
- Monitor memory usage

### 3. Application Scaling
- Scale worker services: `docker-compose up -d --scale worker=5`
- Configure load balancing
- Monitor response times

---

**🎉 Congratulations!** Your Bluesky Automation Platform is now running in production!
