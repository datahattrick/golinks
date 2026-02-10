# Deployment

## Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| Go | 1.22+ | For building from source |
| PostgreSQL | 14+ | Database backend |
| Docker | 20.10+ | For containerized deployment |
| Docker Compose | 2.0+ | For local development |

## Docker

### Pre-built Images

Images are published to both GitHub Container Registry and DockerHub:

```bash
# DockerHub
docker pull qskhattrick/golinks:latest
docker pull qskhattrick/golinks:v1.0.0

# GitHub Container Registry
docker pull ghcr.io/datahattrick/golinks:latest
docker pull ghcr.io/datahattrick/golinks:v1.0.0
```

**Available Tags:** `latest`, `v1.0.0` / `v1.0` / `v1` (semver), `<sha>` (commit hash).

### Run

```bash
docker run -d \
  -p 3000:3000 \
  -e DATABASE_URL="postgres://user:pass@host:5432/golinks" \
  -e OIDC_ISSUER="https://accounts.google.com" \
  -e OIDC_CLIENT_ID="your-client-id" \
  -e OIDC_CLIENT_SECRET="your-client-secret" \
  -e SESSION_SECRET="your-32-char-secret-here" \
  -e LOG_LEVEL="info" \
  qskhattrick/golinks:latest
```

### Docker Compose

```bash
make docker-up        # Full stack (PostgreSQL + OIDC + App)
make docker-down      # Stop and clean up

# Or directly:
docker compose --profile full up -d
```

## Build from Source

```bash
git clone https://github.com/datahattrick/golinks.git
cd golinks
go mod download
go build -o golinks ./cmd/server

export DATABASE_URL="postgres://golinks:golinks@localhost:5432/golinks"
export SESSION_SECRET="your-32-character-secret-here"
./golinks
```

## Helm Chart (Kubernetes / OpenShift)

A Helm chart is provided under `chart/golinks/`. It is compatible with OpenShift restricted SCC.

### Install

```bash
# Create secrets
kubectl create secret generic golinks-secret \
  --from-literal=SESSION_SECRET=$(openssl rand -base64 32) \
  --from-literal=OIDC_CLIENT_SECRET=your-client-secret \
  --from-literal=DATABASE_URL=postgres://user:pass@host:5432/golinks \
  --from-literal=SMTP_PASSWORD=your-smtp-password

# Install chart
helm install golinks ./chart/golinks \
  --set existingSecret=golinks-secret \
  --set route.host=go.example.com \
  --set oidc.issuer=https://accounts.google.com \
  --set oidc.clientId=your-client-id \
  --set image.repository=qskhattrick/golinks \
  --set image.tag=v1.0.0

# With SMTP notifications
helm install golinks ./chart/golinks \
  --set existingSecret=golinks-secret \
  --set smtp.enabled=true \
  --set smtp.host=smtp.example.com \
  --set smtp.from=noreply@example.com
```

**OpenShift notes:** Security context is compatible with restricted SCC. Includes an OpenShift Route with TLS edge termination. Runs as non-root with arbitrary UID.

See `chart/golinks/values.yaml` for all available configuration options.

## Kubernetes (Manual)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: golinks-secrets
type: Opaque
stringData:
  database-url: "postgres://user:password@postgres:5432/golinks?sslmode=require"
  session-secret: "your-32-character-secret-here"
  oidc-client-secret: "your-oidc-client-secret"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: golinks
  labels:
    app: golinks
spec:
  replicas: 2
  selector:
    matchLabels:
      app: golinks
  template:
    metadata:
      labels:
        app: golinks
    spec:
      containers:
        - name: golinks
          image: ghcr.io/datahattrick/golinks:main
          ports:
            - containerPort: 3000
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: golinks-secrets
                  key: database-url
            - name: SESSION_SECRET
              valueFrom:
                secretKeyRef:
                  name: golinks-secrets
                  key: session-secret
            - name: OIDC_ISSUER
              value: "https://accounts.google.com"
            - name: OIDC_CLIENT_ID
              value: "your-client-id"
            - name: OIDC_CLIENT_SECRET
              valueFrom:
                secretKeyRef:
                  name: golinks-secrets
                  key: oidc-client-secret
            - name: OIDC_REDIRECT_URL
              value: "https://go.example.com/auth/callback"
            - name: OIDC_ORG_CLAIM
              value: "hd"
          resources:
            limits:
              memory: "256Mi"
              cpu: "500m"
            requests:
              memory: "128Mi"
              cpu: "100m"
          livenessProbe:
            httpGet:
              path: /healthz
              port: 3000
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /readyz
              port: 3000
            initialDelaySeconds: 5
            periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: golinks
spec:
  selector:
    app: golinks
  ports:
    - port: 80
      targetPort: 3000
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: golinks
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
    - hosts:
        - go.example.com
      secretName: golinks-tls
  rules:
    - host: go.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: golinks
                port:
                  number: 80
```
