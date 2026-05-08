#!/bin/bash

clear

# Colors
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
WHITE='\033[1;37m'
GRAY='\033[0;90m'
NC='\033[0m' # No Color

echo -e "${GREEN}Hello! Following video is the demo of KubeFold Project on real Kubernetes cluster.${NC}" | pv -qL 20
echo -e "${GREEN}It will show how easy is to schedule protein conformation prediction job on GPU cluster.${NC}" | pv -qL 20
echo -e "${GREEN}Keep in mind that some parts of the process may be sped up in video editing.${NC}\n\n" | pv -qL 20

echo -e "${CYAN}First of all, we need to install KubeFold Operator${NC}\n" | pv -qL 20
echo -e "${YELLOW}$ kubectl apply -f https://raw.githubusercontent.com/kubefold/operator/refs/heads/main/dist/install.yaml${NC}" | pv -qL 20

kubectl apply -f https://raw.githubusercontent.com/kubefold/operator/refs/heads/main/dist/install.yaml

echo -e "\n\n${CYAN}Then, we can apply ProteinDatabase Resource that will install proteins databases used for inference.${NC}\n\n" | pv -qL 20
echo -e "${YELLOW}$ cat config/samples/data_v1_proteindatabase.yaml${NC}" | pv -qL 20
cat config/samples/data_v1_proteindatabase.yaml

echo ""
echo -e "${YELLOW}$ kubectl apply -f config/samples/data_v1_proteindatabase.yaml${NC}" | pv -qL 20
kubectl apply -f config/samples/data_v1_proteindatabase.yaml

echo -e "\n\n${MAGENTA}Now we should wait some time to allow initialization of database.${NC}"

sleep 900 # 15 minutes
#sleep 1

echo -e "\n\n${CYAN}After some time administrator can check progress of downloads${NC}" | pv -qL 20
echo -e "${YELLOW}$ kubectl get proteindatabase${NC}" | pv -qL 20

kubectl get proteindatabase

echo -e "\n\n${CYAN}The ProteinDatabase resource creates pvc with selected storage class:${NC}" | pv -qL 20
echo -e "${YELLOW}$ kubectl get pvc${NC}" | pv -qL 20

kubectl get pvc

echo -e "\n\n${CYAN}Then it starts few pods to download resources concurrently:${NC}" | pv -qL 20
echo -e "${YELLOW}$ kubectl get pods${NC}" | pv -qL 20

kubectl get pods

echo -e "\n\n${MAGENTA}At this stage, we should wait until downloads finish${NC}" | pv -qL 20

sleep 2100 # 35 minutes
#sleep 1

echo -e "\n\n${CYAN}After some time we can check status of downloads:${NC}" | pv -qL 20
echo -e "${YELLOW}$ kubectl get proteindatabase${NC}" | pv -qL 20

kubectl get proteindatabase

echo -e "\n\n${GREEN}Seems that resources has been downloaded. We can proceed to schedule prediction task.${NC}" | pv -qL 20

echo -e "${YELLOW}$ cat config/samples/data_v1_proteinconformationprediction.yaml${NC}" | pv -qL 20
cat config/samples/data_v1_proteinconformationprediction_redacted.yaml

echo ""
echo -e "${YELLOW}$ kubectl apply -f config/samples/data_v1_proteinconformationprediction.yaml${NC}" | pv -qL 20
kubectl apply -f config/samples/data_v1_proteinconformationprediction.yaml

sleep 1500 # 25 minutes
#sleep 1

echo -e "\n\n${CYAN}After some time administrator can check progress of task${NC}" | pv -qL 20
echo -e "${YELLOW}$ kubectl get proteinconformationprediction${NC}" | pv -qL 20

kubectl get proteinconformationprediction

echo -e "\n\n${CYAN}Under the hood, KubeFold manages pods that executes aligning and ML inference phases.${NC}" | pv -qL 20
echo -e "${CYAN}Those phases runs on different node groups (CPU or GPU oriented).${NC}" | pv -qL 20
echo -e "${YELLOW}$ kubectl get pods | grep proteinconformationprediction${NC}" | pv -qL 20

kubectl get pods | grep proteinconformationprediction

echo -e "\n\n${MAGENTA}Now we should wait until job finishes.${NC}" | pv -qL 20

sleep 5400 # 1.5h
#sleep 1

echo -e "\n\n${CYAN}We can check status of prediction task${NC}" | pv -qL 20
echo -e "${YELLOW}$ kubectl get proteinconformationprediction${NC}" | pv -qL 20

kubectl get proteinconformationprediction

echo -e "\n\n${CYAN}And also explore artifacts:${NC}" | pv -qL 20
echo -e "${YELLOW}$ aws s3 ls s3://kubefold-artifacts-example/${NC}" | pv -qL 20

aws s3 ls s3://kubefold-artifacts-example/

echo -e "\n\n${GREEN} As you can see artifacts has been uploaded to remote storage.${NC}" | pv -qL 20