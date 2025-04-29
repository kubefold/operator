#!/bin/bash

export AWS_PAGER=""
aws fsx describe-file-systems --region eu-central-1 --query "FileSystems[?FileSystemType=='LUSTRE'].FileSystemId" --output text | xargs -n1 -I {} aws fsx delete-file-system --region eu-central-1 --file-system-id {}
eksctl delete cluster -f eks/cluster.yaml --disable-nodegroup-eviction