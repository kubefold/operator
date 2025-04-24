#!/bin/bash

export AWS_PAGER=""
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
eksctl create cluster -f cluster.yaml
aws eks update-kubeconfig --name alphafold-cluster --region eu-central-1
kubectl config use-context arn:aws:eks:eu-central-1:"$ACCOUNT_ID":cluster/alphafold-cluster
kubectl apply -k "github.com/kubernetes-sigs/aws-fsx-csi-driver/deploy/kubernetes/overlays/stable/?ref=release-1.3"

SUBNET_ID=$(aws eks describe-cluster --name alphafold-cluster --query "cluster.resourcesVpcConfig.subnetIds[0]" --output text)
SECURITY_GROUP_ID=$(aws eks describe-cluster --name alphafold-cluster --query "cluster.resourcesVpcConfig.clusterSecurityGroupId" --output text)

LUSTRE_SG_ID=$(aws ec2 create-security-group --group-name LustreSG --description "Security group for Lustre" --vpc-id $(aws eks describe-cluster --name alphafold-cluster --query "cluster.resourcesVpcConfig.vpcId" --output text) --query GroupId --output text)
aws ec2 authorize-security-group-ingress --group-id $LUSTRE_SG_ID --protocol tcp --port 988 --cidr $(aws ec2 describe-subnets --subnet-ids $SUBNET_ID --query "Subnets[0].CidrBlock" --output text)
aws ec2 authorize-security-group-ingress --group-id $LUSTRE_SG_ID --protocol tcp --port 2049 --cidr $(aws ec2 describe-subnets --subnet-ids $SUBNET_ID --query "Subnets[0].CidrBlock" --output text)
aws ec2 authorize-security-group-ingress --group-id $LUSTRE_SG_ID --protocol tcp --port 80 --cidr $(aws ec2 describe-subnets --subnet-ids $SUBNET_ID --query "Subnets[0].CidrBlock" --output text)
aws ec2 authorize-security-group-ingress --group-id $LUSTRE_SG_ID --protocol tcp --port 2049 --source-group $SECURITY_GROUP_ID
aws ec2 authorize-security-group-ingress --group-id $LUSTRE_SG_ID --protocol tcp --port 988 --source-group $SECURITY_GROUP_ID
aws ec2 authorize-security-group-ingress --group-id $LUSTRE_SG_ID --protocol tcp --port 80 --source-group $SECURITY_GROUP_ID
aws ec2 authorize-security-group-ingress --group-id $LUSTRE_SG_ID --protocol tcp --port 2049 --source-group $LUSTRE_SG_ID
aws ec2 authorize-security-group-ingress --group-id $LUSTRE_SG_ID --protocol tcp --port 988 --source-group $LUSTRE_SG_ID
aws ec2 authorize-security-group-ingress --group-id $LUSTRE_SG_ID --protocol tcp --port 80 --source-group $LUSTRE_SG_ID
aws ec2 authorize-security-group-ingress --group-id $LUSTRE_SG_ID --protocol tcp --port 1018-1023 --cidr $(aws ec2 describe-subnets --subnet-ids $SUBNET_ID --query "Subnets[0].CidrBlock" --output text)
aws ec2 authorize-security-group-ingress --group-id $LUSTRE_SG_ID --protocol tcp --port 1018-1023 --source-group $SECURITY_GROUP_ID
aws ec2 authorize-security-group-ingress --group-id $LUSTRE_SG_ID --protocol tcp --port 1018-1023 --source-group $LUSTRE_SG_ID

aws ec2 create-tags --resources $LUSTRE_SG_ID --tags Key=Name,Value=LustreSG
aws ec2 create-tags --resources $LUSTRE_SG_ID --tags Key=Purpose,Value=LustreSG
aws ec2 create-tags --resources $LUSTRE_SG_ID --tags Key=Owner,Value=alphafold
aws ec2 create-tags --resources $LUSTRE_SG_ID --tags Key=Project,Value=alphafold
aws ec2 create-tags --resources $LUSTRE_SG_ID --tags Key=Environment,Value=production

cp fsx.storageclass.k8s.templ.yaml fsx.storageclass.k8s.yaml
sed -i -e "s/{SECURITY_GROUP_ID}/$LUSTRE_SG_ID/g" fsx.storageclass.k8s.yaml
sed -i -e "s/{SUBNET_ID}/$SUBNET_ID/g" fsx.storageclass.k8s.yaml
rm fsx.storageclass.k8s.yaml-e

kubectl apply -f fsx.storageclass.k8s.yaml