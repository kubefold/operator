#!/bin/bash

export AWS_PAGER=""
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGION="eu-central-1"
CLUSTER_NAME="alphafold-cluster"
VPC_ID="vpc-0a4d6ce6217cd12cb"
FSX_SG_NAME="alphafold-fsx-lustre-sg"

eksctl create cluster -f eks/cluster.yaml
aws eks update-kubeconfig --name "$CLUSTER_NAME" --region "$REGION"
kubectl config use-context arn:aws:eks:"$REGION":"$ACCOUNT_ID":cluster/"$CLUSTER_NAME"

# --- Upsert FSx for Lustre security group ---
aws_sg_rule() {
  local err
  err=$(aws ec2 "$@" 2>&1) || {
    echo "$err" | grep -q "InvalidPermission.Duplicate" || { echo "ERROR: $err"; return 1; }
  }
}

FSX_SG_ID=$(aws ec2 describe-security-groups \
  --region "$REGION" \
  --filters "Name=group-name,Values=$FSX_SG_NAME" "Name=vpc-id,Values=$VPC_ID" \
  --query "SecurityGroups[0].GroupId" --output text)

if [ "$FSX_SG_ID" = "None" ] || [ -z "$FSX_SG_ID" ]; then
  FSX_SG_ID=$(aws ec2 create-security-group \
    --region "$REGION" \
    --group-name "$FSX_SG_NAME" \
    --description "Security group for FSx for Lustre ($CLUSTER_NAME)" \
    --vpc-id "$VPC_ID" \
    --query "GroupId" --output text)
  echo "Created FSx SG: $FSX_SG_ID"
else
  echo "Using existing FSx SG: $FSX_SG_ID"
fi

# Self-referencing inbound rules (inter-server Lustre traffic)
aws_sg_rule authorize-security-group-ingress --region "$REGION" --group-id "$FSX_SG_ID" \
  --ip-permissions "IpProtocol=tcp,FromPort=988,ToPort=988,UserIdGroupPairs=[{GroupId=$FSX_SG_ID,Description='Lustre traffic between FSx file servers'}]"
aws_sg_rule authorize-security-group-ingress --region "$REGION" --group-id "$FSX_SG_ID" \
  --ip-permissions "IpProtocol=tcp,FromPort=1018,ToPort=1023,UserIdGroupPairs=[{GroupId=$FSX_SG_ID,Description='Lustre traffic between FSx file servers'}]"

# Self-referencing outbound rules (inter-server Lustre traffic)
aws_sg_rule authorize-security-group-egress --region "$REGION" --group-id "$FSX_SG_ID" \
  --ip-permissions "IpProtocol=tcp,FromPort=988,ToPort=988,UserIdGroupPairs=[{GroupId=$FSX_SG_ID,Description='Allow Lustre traffic between FSx file servers'}]"
aws_sg_rule authorize-security-group-egress --region "$REGION" --group-id "$FSX_SG_ID" \
  --ip-permissions "IpProtocol=tcp,FromPort=1018,ToPort=1023,UserIdGroupPairs=[{GroupId=$FSX_SG_ID,Description='Allow Lustre traffic between FSx file servers'}]"

# Rules allowing EKS worker nodes (Lustre clients) to communicate with FSx
NODE_SGS=$(aws ec2 describe-security-groups \
  --region "$REGION" \
  --filters "Name=tag:aws:eks:cluster-name,Values=$CLUSTER_NAME" \
  --query "SecurityGroups[*].GroupId" --output text)

for NODE_SG in $NODE_SGS; do
  aws_sg_rule authorize-security-group-ingress --region "$REGION" --group-id "$FSX_SG_ID" \
    --ip-permissions "IpProtocol=tcp,FromPort=988,ToPort=988,UserIdGroupPairs=[{GroupId=$NODE_SG,Description='Lustre traffic from EKS worker nodes'}]"
  aws_sg_rule authorize-security-group-ingress --region "$REGION" --group-id "$FSX_SG_ID" \
    --ip-permissions "IpProtocol=tcp,FromPort=1018,ToPort=1023,UserIdGroupPairs=[{GroupId=$NODE_SG,Description='Lustre traffic from EKS worker nodes'}]"
  aws_sg_rule authorize-security-group-egress --region "$REGION" --group-id "$FSX_SG_ID" \
    --ip-permissions "IpProtocol=tcp,FromPort=988,ToPort=988,UserIdGroupPairs=[{GroupId=$NODE_SG,Description='Lustre traffic to EKS worker nodes'}]"
  aws_sg_rule authorize-security-group-egress --region "$REGION" --group-id "$FSX_SG_ID" \
    --ip-permissions "IpProtocol=tcp,FromPort=1018,ToPort=1023,UserIdGroupPairs=[{GroupId=$NODE_SG,Description='Lustre traffic to EKS worker nodes'}]"
done

# Write FSx SG ID into StorageClass manifest
sed "s|  securityGroupIds:.*|  securityGroupIds: $FSX_SG_ID|" \
  eks/fsx.storageclass.k8s.yaml > eks/fsx.storageclass.k8s.yaml.tmp \
  && mv eks/fsx.storageclass.k8s.yaml.tmp eks/fsx.storageclass.k8s.yaml
# --- End FSx security group ---

kubectl apply -k "github.com/kubernetes-sigs/aws-fsx-csi-driver/deploy/kubernetes/overlays/stable/?ref=release-1.3"
kubectl apply -f eks/fsx.storageclass.k8s.yaml

kubectl apply -f https://raw.githubusercontent.com/kubefold/operator/refs/heads/main/dist/install.yaml
kubectl apply -f config/samples/data_v1_proteindatabase.yaml

echo 'Done'