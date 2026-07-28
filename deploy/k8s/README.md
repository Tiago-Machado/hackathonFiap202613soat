# Kubernetes — overlay de demonstração

Este diretório é o **artefato bônus de escalabilidade**. A fundação de
desenvolvimento continua sendo o Docker Compose; aqui a mesma topologia é
reencenada em Kubernetes para demonstrar como o sistema atende aos requisitos de
*"processar vários vídeos ao mesmo tempo"*, *"em picos não perder requisição"* e
*"arquitetura que permita escalar"*.

## Topologia

Uma imagem, dois binários. API e worker rodam a **mesma imagem** (`./api` e
`./worker`), como no Dockerfile — um build, dois Deployments.

```
                        ┌── api (Deployment, HPA por CPU) ──┐
  browser ── Ingress ──►│                                   │──► RabbitMQ ──┐
                        └───────────────────────────────────┘               │
                                                                            fila
  postgres (STS+PVC)   rabbitmq (STS+PVC)   minio (STS+PVC)   redis         │
        ▲                     ▲                    ▲                         ▼
        └─────────── worker (Deployment) ◄── KEDA escala pelo tamanho da fila
```

## Pré-requisitos

- Um cluster local: **kind** ou **minikube**.
- `metrics-server` (para os HPAs). No minikube: `minikube addons enable metrics-server`.
- (Opcional) **KEDA**, para o autoescalonamento por fila — ver abaixo.
- (Opcional) um ingress controller, se for usar o `42-ingress.yaml`.

## Passo a passo

### 1. Buildar e carregar a imagem no cluster

```bash
docker build -t video-processor:local .

# kind:
kind load docker-image video-processor:local
# minikube:
# minikube image load video-processor:local
```

### 2. Config base + serviços de apoio

```bash
kubectl apply -k deploy/k8s
```

Isso cria namespace, ConfigMap/Secret, Postgres, RabbitMQ, Redis, MinIO, o Job de
setup do MinIO (bucket + credencial escopada), a API e o worker.

### 3. Rodar as migrations

As migrations entram via ConfigMap gerado da sua pasta `migrations/`:

```bash
kubectl create configmap fiapx-migrations \
  --from-file=migrations/ -n fiapx \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f deploy/k8s/30-migrations-job.yaml
kubectl -n fiapx wait --for=condition=complete job/db-migrations --timeout=120s
```

### 4. Escolher o autoescalonamento do worker (um OU o outro)

**Opção A — KEDA, por tamanho de fila (recomendado, é o destaque da apresentação):**

```bash
helm repo add kedacore https://kedacore.github.io/charts && helm repo update
helm install keda kedacore/keda -n keda --create-namespace
kubectl apply -f deploy/k8s/51-worker-scaler-keda.yaml
```

**Opção B — HPA por CPU (plano B, sem dependência externa):**

```bash
kubectl apply -f deploy/k8s/51-worker-scaler-hpa.yaml
```

> Nunca aplique as duas — o KEDA gerencia seu próprio HPA e elas conflitam.

### 5. Acessar

```bash
# API:
kubectl -n fiapx port-forward svc/api 8080:8080
# MinIO (necessário para as presigned URLs abrirem no browser):
kubectl -n fiapx port-forward svc/minio 9000:9000
```

Com o port-forward do MinIO ativo, `S3_PUBLIC_ENDPOINT=http://localhost:9000` (o
default do ConfigMap) funciona direto. Se preferir Ingress, aplique o
`42-ingress.yaml`, aponte `s3.local` no `/etc/hosts` e troque o
`S3_PUBLIC_ENDPOINT` para `http://s3.local`.

## Demonstrar a escalabilidade (o momento do vídeo)

```bash
# Numa aba, observe os workers nascendo e morrendo:
kubectl -n fiapx get pods -l app=worker -w

# Noutra, dispare vários uploads em paralelo. A fila enche,
# o KEDA sobe réplicas de worker, a fila drena, ele desce de volta a zero.
```

A frase que amarra tudo: *o RabbitMQ garante que nenhuma requisição se perde num
pico (ela só espera na fila), e o KEDA transforma o tamanho dessa fila em número
de workers — escalando pela causa, o trabalho pendente, e não por um sintoma
indireto como CPU.*

## Três pontos a conferir antes de aplicar

1. **DSNs** (`11-secret.yaml`): `DATABASE_URL` e `RABBITMQ_URL` foram montadas na
   convenção padrão. Confira `sslmode` e vhost contra o seu `docker-compose.yml`,
   que é o seu ambiente comprovadamente funcional.
2. **Nome da fila** (`51-worker-scaler-keda.yaml`): `queueName: video.process` é um
   placeholder. Coloque o nome exato que o `rabbitmq.NewConsumer` declara.
3. **Migrations** (`30-migrations-job.yaml`): assume `golang-migrate`. Se o
   migration-runner do seu compose usa outro mecanismo, alinhe imagem/comando.

## Nota de produção (decisões conscientes)

Postgres, RabbitMQ, MinIO e Redis aqui rodam **dentro** do cluster como
StatefulSets, o que mantém o `apply -k` autocontido e demoável. Numa entrega real
esses seriam serviços gerenciados ou operators, e o Secret viria de
SealedSecrets / External Secrets / Vault em vez de texto puro versionado.
