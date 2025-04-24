#!/bin/bash

aws fsx --region eu-central-1 describe-file-systems --query "FileSystems[?contains(Name, 'alphafold')].FileSystemId" --output text | xargs -I {} aws fsx delete-file-system --file-system-id {} --force
eksctl delete cluster -f cluster.yaml --disable-nodegroup-eviction