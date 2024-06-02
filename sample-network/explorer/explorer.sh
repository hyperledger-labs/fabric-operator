#!/bin/bash
# kubectl create namespace explorer
kubectl create secret generic explorer-secret --from-file=key.pem=${PWD}/../temp/enrollments/org1/users/org1admin/msp/keystore/key.pem \
--from-file=cert.pem=${PWD}/../temp/enrollments/org1/users/org1admin/msp/signcerts/cert.pem \
  --from-file=tlsca-signcert.pem=${PWD}/../temp/channel-msp/peerOrganizations/org1/msp/tlscacerts/tlsca-signcert.pem -n test-network
kubectl apply -f  . -n test-network
