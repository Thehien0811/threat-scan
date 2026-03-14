# Deployment Guide

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 1.29+
- 4GB RAM minimum
- 10GB disk space for virus definitions
- Internet access (for initial setup)

## Quick Deployment

### 1. Clone and Setup

```bash
git clone <repository> threat-scan
cd threat-scan

# Run setup script
chmod +x setup.sh
./setup.sh
```

### 2. Configure (Optional)

Edit `config/config.yaml` to customize:
- Port bindings
- Timeout values
- AV engines
- NSQ settings

### 3. Start Services

```bash
docker-compose up -d
```

Wait for services to be healthy:
```bash
docker-compose ps
```

All services should show "Up".

### 4. Verify Health

```bash
# Check gRPC service
docker-compose logs threat-scan-service | grep -i "health\|started\|error"

# Check ClamAV
docker-compose exec clamav clamdscan --version

# Check NSQ
curl -s http://localhost:4161/api/nodes | jq .
```

## Environment-Specific Deployment

### Development Environment

Uses local file volumes and no resource limits:
```bash
docker-compose up -d
```

### Staging Environment

```bash
docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d
```

Staging version adds:
- Resource limits: 2 CPU, 2GB RAM
- Health checks enabled
- Logging aggregation

### Production Environment

```bash
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

Production version adds:
- Auto-restart policies
- Resource limits: 4 CPU, 4GB RAM
- Persistent volumes on NFS
- SSL/TLS for gRPC
- Monitoring/alerting integration

## Kubernetes Deployment

### 1. Create Namespace

```bash
kubectl create namespace threat-scan
```

### 2. Deploy Services

```bash
# ConfigMap from config.yaml
kubectl create configmap threat-scan-config \
  --from-file=config/config.yaml \
  -n threat-scan

# Secrets for any credentials
kubectl create secret generic threat-scan-secrets \
  --from-literal=api-key=xxxxx \
  -n threat-scan

# Deploy via manifests
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/nsqlookupd.yaml
kubectl apply -f k8s/nsqd.yaml
kubectl apply -f k8s/clamav.yaml
kubectl apply -f k8s/threat-scan-service.yaml
```

### 3. Verify Deployment

```bash
kubectl get pods -n threat-scan
kubectl logs -f deployment/threat-scan-service -n threat-scan
```

## AWS ECS Deployment

### 1. Create ECR Repositories

```bash
aws ecr create-repository --repository-name threat-scan-service
aws ecr create-repository --repository-name clamav
aws ecr create-repository --repository-name nsqd
```

### 2. Push Images

```bash
# Get login token
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin ${AWS_ACCOUNT}.dkr.ecr.us-east-1.amazonaws.com

# Tag and push
docker tag threat-scan-service \
  ${AWS_ACCOUNT}.dkr.ecr.us-east-1.amazonaws.com/threat-scan-service:latest
docker push ${AWS_ACCOUNT}.dkr.ecr.us-east-1.amazonaws.com/threat-scan-service:latest
```

### 3. Create ECS Task Definition

```json
{
  "family": "threat-scan",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "2048",
  "memory": "4096",
  "containerDefinitions": [
    {
      "name": "threat-scan-service",
      "image": "${AWS_ACCOUNT}.dkr.ecr.us-east-1.amazonaws.com/threat-scan-service:latest",
      "portMappings": [{"containerPort": 50051, "protocol": "tcp"}],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/threat-scan",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

### 4. Create Service

```bash
aws ecs create-service \
  --cluster threat-scan \
  --service-name threat-scan-service \
  --task-definition threat-scan \
  --desired-count 2 \
  --launch-type FARGATE
```

## Azure Container Instances Deployment

```bash
# Create resource group
az group create --name threat-scan --location eastus

# Deploy container
az container create \
  --resource-group threat-scan \
  --name threat-scan-service \
  --image threat-scan-service:latest \
  --ports 50051 4150 4160 \
  --cpu 4 --memory 4 \
  --registry-login-server acr.azurecr.io \
  --registry-username username \
  --registry-password password
```

## GCP Cloud Run Deployment

Note: Cloud Run requires stateless services. For NSQ/persistent state, use GKE.

```bash
# Build and push
gcloud builds submit --tag gcr.io/PROJECT_ID/threat-scan-service

# Deploy
gcloud run deploy threat-scan-service \
  --image gcr.io/PROJECT_ID/threat-scan-service \
  --platform managed \
  --region us-central1 \
  --port 50051 \
  --memory=4Gi \
  --cpu=2
```

## SSL/TLS Configuration

### Generate Certificates

```bash
# Self-signed (development)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365

# Production: Use proper CA certificates
```

### Configure gRPC with TLS

Update service code:
```go
creds, err := credentials.NewServerTLSFromFile("cert.pem", "key.pem")
s := grpc.NewServer(grpc.Creds(creds))
```

Update docker-compose:
```yaml
services:
  threat-scan-service:
    volumes:
      - ./certs/cert.pem:/certs/cert.pem:ro
      - ./certs/key.pem:/certs/key.pem:ro
    environment:
      CERT_FILE: /certs/cert.pem
      KEY_FILE: /certs/key.pem
```

## Persistent Storage

### NFS Setup

```bash
# Create NFS share for uploads
sudo mkdir -p /exports/threat-scan-uploads
sudo chown nobody:nogroup /exports/threat-scan-uploads
echo "/exports/threat-scan-uploads *(rw,sync,subtree_check)" | sudo tee -a /etc/exports
sudo exportfs -a
```

### Docker Volume Configuration

```yaml
volumes:
  uploads:
    driver: local
    driver_opts:
      type: nfs
      o: addr=nfs-server:10.0.0.1,vers=4,soft,timeo=180,bg,tcp,rw
      device: ":/exports/threat-scan-uploads"
```

## Backup and Recovery

### Backup Strategy

```bash
#!/bin/bash

BACKUP_DIR=/backups/threat-scan
mkdir -p $BACKUP_DIR

# Backup configuration
docker cp threat-scan-service:/etc/threat-scan $BACKUP_DIR/config

# Backup virus definitions
docker run --rm \
  -v threat-scan_clamav-db:/data \
  -v $BACKUP_DIR:/backup \
  alpine tar czf /backup/clamav-db-$(date +%Y%m%d).tar.gz /data

# Backup NSQ state
docker run --rm \
  -v threat-scan_nsqd-data:/data \
  -v $BACKUP_DIR:/backup \
  alpine tar czf /backup/nsqd-state-$(date +%Y%m%d).tar.gz /data

# Upload to S3
aws s3 sync $BACKUP_DIR s3://threat-scan-backups/
```

### Recovery Procedure

```bash
# Stop services
docker-compose down

# Restore volumes
docker run --rm \
  -v threat-scan_clamav-db:/data \
  -v $BACKUP_DIR:/backup \
  alpine tar xzf /backup/clamav-db-latest.tar.gz -C /

# Restart
docker-compose up -d
```

## Monitoring and Logging

### Prometheus Integration

```yaml
services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
```

### ELK Stack Integration

```yaml
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.0.0
  
  kibana:
    image: docker.elastic.co/kibana/kibana:8.0.0
    ports:
      - "5601:5601"

  logstash:
    image: docker.elastic.co/logstash/logstash:8.0.0
    volumes:
      - ./logstash.conf:/usr/share/logstash/pipeline/logstash.conf
```

### Sending Logs to CloudWatch

```yaml
logging:
  driver: awslogs
  options:
    awslogs-group: /ecs/threat-scan
    awslogs-region: us-east-1
    awslogs-stream-prefix: service
```

## Scaling Strategies

### Horizontal Scaling

Deploy multiple replicas:
```bash
# Docker Compose
docker-compose up -d --scale threat-scan-service=3

# Kubernetes
kubectl scale deployment threat-scan-service --replicas=5

# AWS ECS
aws ecs update-service --cluster threat-scan \
  --service threat-scan-service --desired-count 5
```

### Auto-scaling Configuration

Kubernetes HPA:
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: threat-scan-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: threat-scan-service
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## Troubleshooting Deployment

### Container won't start

```bash
docker-compose logs threat-scan-service
docker inspect threat-scan-service
```

### ClamAV virus definitions not updating

```bash
docker-compose exec clamav freshclam -v
docker-compose logs clamav
```

### NSQ connectivity issues

```bash
docker-compose exec nsqd nc -zv localhost 4150
docker-compose logs nsqd
```

### High memory usage

Check ClamAV settings:
```yaml
av_engines:
  clamav:
    timeout: 60
```

Reduce max concurrent scans in config.

## Performance Tuning

### Resource Allocation

```yaml
services:
  threat-scan-service:
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8G
        reservations:
          cpus: '2'
          memory: 4G
```

### ClamAV Performance

```dockerfile
RUN cat >> /etc/clamav/clamd.conf << EOF
MaxScanSize 500M
MaxFileSize 400M
MaxRecursion 16
MaxFiles 10000
ThreadPool 4
EOF
```

### NSQ Queue Size

```yaml
environment:
  - NSQ_MAX_MSG_SIZE=5242880
performance_tuning:
  nsq_buffer: 256
```
