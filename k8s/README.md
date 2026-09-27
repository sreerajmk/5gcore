# Kubernetes migration notes

This directory contains the Kubernetes equivalents of the Docker Compose services.

## Build the image first

```bash
docker build -t 5gcore:latest .
```

## Apply the manifests

```bash
kubectl apply -f k8s/nrf.yaml
kubectl apply -f k8s/amf.yaml
kubectl apply -f k8s/gnb.yaml
kubectl apply -f k8s/ue-sim.yaml
```

The remaining NFs can be added in the same pattern: SMF, AUSF, UDM, and UPF.

## Step-by-step design understanding

1. NRF acts as the service registry and is exposed as a Kubernetes Service.
2. AMF runs as a Deployment and discovers the other services through the NRF URL.
3. gNB and UE simulator are separate Deployments with their own Services.
4. Traffic uses Kubernetes DNS names like `http://amf:8001` and `http://gnb:8010`.
5. This is the Kubernetes equivalent of your Compose networking model.
