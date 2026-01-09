#!/bin/bash
export NS=test-network
kubectl -n $NS delete deployment explorer-db explorer --ignore-not-found=true
kubectl -n $NS delete service explorerdb-service explorer --ignore-not-found=true
kubectl -n $NS delete configmap explorer-config --ignore-not-found=true
kubectl -n $NS delete secret my-secret --ignore-not-found=true
kubectl -n $NS delete ingress explorer --ignore-not-found=true
kubectl -n $NS delete pvc mypvc  --ignore-not-found=true
kubectl  delete pv mypv  --ignore-not-found=true