#!/bin/bash

export AWS_PAGER=""
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
eksctl create cluster -f eks/cluster.yaml
aws eks update-kubeconfig --name alphafold-cluster --region eu-central-1
kubectl config use-context arn:aws:eks:eu-central-1:"$ACCOUNT_ID":cluster/alphafold-cluster
kubectl apply -k "github.com/kubernetes-sigs/aws-fsx-csi-driver/deploy/kubernetes/overlays/stable/?ref=release-1.3"
kubectl apply -f eks/fsx.storageclass.k8s.yaml

make install
kubectl apply -f config/samples/data_v1_proteindatabase.yaml
make run