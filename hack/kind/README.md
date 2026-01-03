# Kind – Ambiente local de desenvolvimento

Este diretório contém tudo o que é necessário para rodar um cluster
Kubernetes local usando **kind**, focado em testar o agente de logs
como DaemonSet.

## Pré-requisitos

- Docker Desktop (Hyper-V backend)
- kind
- kubectl

## Criando o cluster

```powershell
cd hack/kind
.\cluster.ps1 up
```

### Verifique:

```powershell
kubectl get nodes
```

## Deploy de pod gerador de logs
```powershell
kubectl apply -f deploy-hello-log.yaml
```

### Verifique os logs:

```powershell
kubectl logs hello-log
```

## Removendo o cluster
```powershell
.\cluster.ps1 down
```

## Observações
Os logs dos pods ficam em `/var/log/pods` dentro dos nodes do kind

Este cluster é apenas para desenvolvimento local