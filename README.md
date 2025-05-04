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

## ☁️ Running on AWS EKS

This project provides resources for easy deployment on Amazon EKS, including GPU and FSx Lustre support for high-performance workloads.

- `eks/cluster.yaml`: Example EKS cluster configuration (with CPU and GPU node groups, IAM policies, and VPC setup) for use with `eksctl`.
- `eks/fsx.storageclass.k8s.yaml`: StorageClass for FSx for Lustre, enabling fast, shared storage for protein data and models.
- `eks/sample-lustre-volume.k8s.yaml`: Sample PersistentVolumeClaim using the FSx StorageClass.
- `up.sh`: Automated script to provision the EKS cluster, install the FSx CSI driver, set up storage, deploy Kubefold, and apply a sample resource. Run this script for a one-command setup (requires AWS CLI, eksctl, and kubectl configured).

**Note:**
- The provided EKS configuration includes a GPU node group using `g5.xlarge` instances. You must request a quota increase for g5 instances in your AWS region before running the setup.

**Quick start:**
(Recommended) Run the automated setup script:
```sh
# Automatically:
# - creates EKS cluster with GPU nodegroup
# - installs AWS FSx CSI Driver for automated FSx for Lustre volume provisioning
# - create fsx-sc storage class
# - installs kubefold controller
# - deploys sample ProteinDatabase
./up.sh
```

These resources help you get started with scalable, cloud-native protein prediction workflows on AWS.

