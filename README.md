# WebSocket Game Server on Kubernetes 

## Overview

This is a project to test new architecture appliable to 'https://github.com/iErcann/Notblox'

Sample game server made to test perfs (Golang), and how scalable it can be to create new instance for each player. Right now this is a relay server, which could be highly scalable, let's see!


A scalable WebSocket-based game server that:
- Creates isolated game sessions
- Runs in Docker containers
- Deploys on Kubernetes
- Auto-scales with player demand

## Project Structure
```
game-server/
└── src  
    ├── cmd
    │   ├── client
    │   │   └── main.go
    │   │   └── .env.example
    │   └── server
    │       └── main.go
    │   │   └── .env.example
    ├── go.mod
    ├── go.sum
    ├── internal
    │   └── shared
    │       └── types.go
├── Dockerfile           # Server build
├── k8s/
│   ├── deployment.yaml  # K8s pod configuration
│   ├── service.yaml     # Network exposure
│   └── ingress.yaml     # External access
└── README.md
```

## Getting Started

### Environment Configuration
Create a `.env` file in client or server:
```env
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
TICK_RATE=20
```

### Running the Game Server
```bash
cd src
go run cmd/server/main.go
```

### Running the Test Client
```bash
cd src
go run cmd/client/main.go
```

### Building the Dockerfile
docker build -t mygame/game-server:latest .
docker run mygame/game-server

## 3. Kubernetes Manifests

### `k8s/deployment.yaml`
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: game-server
spec:
  replicas: 3  # Initial instances
  selector:
    matchLabels:
      app: game-server
  template:
    metadata:
      labels:
        app: game-server
    spec:
      containers:
      - name: game-server
        image: your-registry/game-server:v1
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
      autoscaling:
        minReplicas: 2
        maxReplicas: 50
        targetCPUUtilizationPercentage: 80
```

### `k8s/service.yaml`
```yaml
apiVersion: v1
kind: Service
metadata:
  name: game-service
spec:
  selector:
    app: game-server
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
  type: ClusterIP
```

### `k8s/ingress.yaml`
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: game-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /$1
spec:
  rules:
  - http:
      paths:
      - path: /ws/(.*)
        pathType: Prefix
        backend:
          service:
            name: game-service
            port:
              number: 80
```

## Deployment Workflow

### 1. Build & Push Docker Image
```bash
docker build -t your-registry/game-server:v1 .
docker push your-registry/game-server:v1
```

### 2. Deploy to Kubernetes
```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
```

### 3. Verify Deployment
```bash
kubectl get pods -l app=game-server
kubectl get svc game-service
kubectl get ingress game-ingress
```

## Client Connection
Players connect using:
```javascript
const sessionId = "game-" + Date.now();
const socket = new WebSocket(
  `wss://your-domain.com/ws/${sessionId}`
);

socket.onmessage = (event) => {
  console.log("Game update:", event.data);
};
```

## Scaling Features
- **Auto-scaling:** Pods scale from 2 to 50 based on CPU
- **Rolling Updates:** 
  ```bash
  kubectl rollout restart deployment/game-server
  ```
- **Monitoring:**
  ```bash
  kubectl top pods -l app=game-server
  ```

## Dependencies
- Go 1.21+
- Docker 20.10+
- Kubernetes 1.25+
- NGINX Ingress Controller

---

**Note:** For production use:
1. Add TLS to Ingress
2. Implement session persistence
3. Add Redis for shared state
4. Set resource limits based on load testing
