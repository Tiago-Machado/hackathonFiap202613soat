# Deploy em Kubernetes

Artefato de demonstração da capacidade de escalar o sistema em um orquestrador.
Testado em cluster local (kind ou minikube). O ambiente de desenvolvimento
principal continua sendo o Docker Compose.

## Pré-requisitos

- Um cluster local: `kind` ou `minikube`
- `kubectl` configurado apontando para o cluster
- Metrics Server instalado (necessário para o HorizontalPodAutoscaler)

## Passo a passo

1. Construa a imagem da aplicação e disponibilize-a ao cluster.

   ```bash
   docker build -t fiapx-video:local .
   # kind:
   kind load docker-image fiapx-video:local
   # minikube:
   minikube image load fiapx-video:local
   ```

2. Crie os ConfigMaps com as migrations e a policy do MinIO (a partir dos
   arquivos do repositório).

   ```bash
   kubectl apply -f k8s/00-namespace.yaml
   kubectl -n fiapx create configmap migrations --from-file=migrations/
   kubectl -n fiapx create configmap minio-policy --from-file=deploy/minio/app-policy.json
   ```

3. Aplique todos os manifests.

   ```bash
   kubectl apply -f k8s/
   ```

4. Acompanhe a subida.

   ```bash
   kubectl -n fiapx get pods -w
   ```

5. Acesse a API.

   ```bash
   kubectl -n fiapx port-forward svc/api 8080:8080
   # ou, via NodePort do cluster, na porta 30080
   ```

## Demonstração de escalabilidade

O `worker` possui um HorizontalPodAutoscaler (2 a 5 réplicas, alvo de 70% de CPU).
Sob carga de processamento, o número de pods do worker cresce automaticamente:

```bash
kubectl -n fiapx get hpa worker -w
kubectl -n fiapx get pods -l app=worker
```

## Observação sobre segredos

O `k8s/01-config.yaml` traz um `Secret` com valores de demonstração. Em um
ambiente real, esses valores viriam de um gerenciador de segredos (Sealed
Secrets, External Secrets ou Vault) e nunca seriam versionados.
