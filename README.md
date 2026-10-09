# How to run the backend server for tests

The Go backend under `server/` is a drop-in replacement for the Postman local mock
(`postman/mocks/inventory-tracker-mock/default.js`) — same routes, same auth
(`x-api-key`), same seeded data, same business logic. It can run three ways: a bare
Docker container, inside Docker Desktop's local Kubernetes, or on the shared
`se-kube-demo` EKS cluster. All three expose the same API on port `4500`.

> **State is in-memory and single-instance.** Every method below resets all data
> (back to the seeded item/location) on restart/redeploy, and must never run with
> more than 1 replica — multiple replicas would each hold independent, diverging data
> behind the same Service.

## 1. Local docker container

**Run:**
```
docker build -t inventory-tracker-backend ./server
docker run --rm -d --name inventory-tracker-backend -p 4500:4500 inventory-tracker-backend
```
(or `docker compose up -d` using the root `docker-compose.yml`, which does the same thing)

**Verify:**
```
curl http://localhost:4500/health
```

**Run tests:** in Postman, select the **Dev** environment, then run the collection
(baseUrl `http://localhost:4500`).

**Tear down:**
```
docker rm -f inventory-tracker-backend
```
(or `docker compose down` if you used compose)

## 2. Local k8s (Docker Desktop)

Prereq: Docker Desktop's built-in Kubernetes is enabled and `docker-desktop` is an
available `kubectl` context.

**Deploy:**
```
docker build -t inventory-tracker-backend:local ./server
kubectl --context docker-desktop apply -k k8s/overlays/local
kubectl --context docker-desktop -n inventory-tracker wait --for=condition=Ready pod -l app=inventory-tracker-backend --timeout=60s
```

**Verify:**
```
kubectl --context docker-desktop -n inventory-tracker get pods,svc
curl http://localhost:4500/health
```
The Service is `type: LoadBalancer`; Docker Desktop auto-maps it to `localhost`, so
it's reachable on the same port as the bare container above.

**Run tests:** in Postman, select the **Staging** environment (baseUrl
`http://localhost:4500`), then run the collection.

**Tear down:**
```
kubectl --context docker-desktop delete -k k8s/overlays/local
```

## 3. AWS EKS cluster (se-kube-demo)

Prereq: authenticated AWS SSO session (`aws sts get-caller-identity` succeeds) and the
`arn:aws:eks:us-east-2:005904641462:cluster/se-kube-demo` context in `~/.kube/config`.

**Build & push the image** (must be `linux/amd64` — EKS nodes are amd64, regardless of
what architecture you're building on):
```
aws ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin 005904641462.dkr.ecr.us-east-2.amazonaws.com
docker buildx build --platform linux/amd64 -t 005904641462.dkr.ecr.us-east-2.amazonaws.com/se-challenge/gcho-inventory-tracker:v1 --push ./server
```

**Deploy:**
```
kubectl --context arn:aws:eks:us-east-2:005904641462:cluster/se-kube-demo apply -k k8s/overlays/eks
kubectl --context arn:aws:eks:us-east-2:005904641462:cluster/se-kube-demo -n gcho wait --for=condition=Ready pod -l app=inventory-tracker-backend --timeout=90s
```

**Verify:**
```
kubectl --context arn:aws:eks:us-east-2:005904641462:cluster/se-kube-demo -n gcho get pods,svc
```

The Service is `type: ClusterIP` (no public endpoint on purpose). Reach it with a
port-forward, on the same port `4500` used everywhere else above:
```
kubectl --context arn:aws:eks:us-east-2:005904641462:cluster/se-kube-demo -n gcho port-forward svc/inventory-tracker-backend 4500:4500
```
Leave that running, then in a separate terminal:
```
curl http://localhost:4500/health
```

**Run tests:** with the port-forward above still running, in Postman select the
**Staging** environment (baseUrl `http://localhost:4500` — same environment used for
local k8s; just don't run local k8s and this port-forward at the same time, since
they'd both claim port 4500), then run the collection.

**Tear down:**
```
kubectl --context arn:aws:eks:us-east-2:005904641462:cluster/se-kube-demo delete -k k8s/overlays/eks
aws ecr delete-repository --repository-name se-challenge/gcho-inventory-tracker --region us-east-2 --force
```
(stop the port-forward with `Ctrl-C` first, or via k9s's PortForwards view / `Ctrl-D`
if you started it there instead)

The `aws ecr delete-repository` step removes the pushed image entirely (ECR bills for
storage regardless of use, however small) — the next deploy will need to recreate the
repository and re-push the image from scratch (see the build & push step above).
