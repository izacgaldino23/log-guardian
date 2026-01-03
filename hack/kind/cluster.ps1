param(
  [string]$Action = "up"
)

$CLUSTER_NAME = "log-agent-dev"

if ($Action -eq "up") {
  kind create cluster `
    --name $CLUSTER_NAME `
    --config kind-config.yaml
}
elseif ($Action -eq "down") {
  kind delete cluster --name $CLUSTER_NAME
}
else {
  Write-Host "Usage: ./cluster.ps1 up|down"
}