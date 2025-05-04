<div align="center">
  <img src="assets/logo.png" alt="Kubefold Logo" />
</div>

# 🧬 Kubefold

Kubefold is a Kubernetes operator for managing protein structure prediction workflows. It provides a declarative way to handle protein databases and run conformation predictions in a Kubernetes cluster.

## 🚀 Features

- **Protein Database Management** - Download and manage various protein databases (UniProt, PDB, BFD, etc.)
- **Conformation Prediction** - Run protein structure predictions with configurable parameters
- **Cloud Integration** - Store results in S3 and receive notifications via SMS
- **Scalable Architecture** - Built on Kubernetes for horizontal scaling and resource management

## 📋 Prerequisites

- Kubernetes cluster (v1.11.3+)
- kubectl (v1.11.3+)
- Access to an S3 bucket for storing results
- AWS credentials for S3 and SMS notifications (if using these features)

## 🛠️ Installation

```sh
kubectl apply -f https://raw.githubusercontent.com/kubefold/operator/main/dist/install.yaml
```

## 📝 Usage

### Protein Database

Create a ProteinDatabase resource to download and manage protein databases:

```yaml
apiVersion: data.kubefold.io/v1
kind: ProteinDatabase
metadata:
  name: my-database
spec:
  datasets:
    uniprot: true
    pdb: true
  volume:
    storageClassName: fsx-sc
```

### Protein Conformation Prediction

Run a protein structure prediction:

```yaml
apiVersion: data.kubefold.io/v1
kind: ProteinConformationPrediction
metadata:
  name: my-prediction
spec:
  database: my-database
  protein:
    id: ['A']
    sequence: "YOUR_PROTEIN_SEQUENCE"
  model:
    volume:
      storageClassName: fsx-sc
    weights:
      http: "https://your-model-weights.bin.zst"
  destination:
    s3:
      bucket: your-bucket
      region: your-region
  notify:
    region: your-region
    sms:
      - "+1234567890"
```

## 🧹 Cleanup

To uninstall the operator:

```sh
kubectl delete -f https://raw.githubusercontent.com/kubefold/operator/main/dist/install.yaml
```

