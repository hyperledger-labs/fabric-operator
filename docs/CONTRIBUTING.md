# Contributing to this repository

## Tips:
After changed module define at `/api/v1beta1/*.go` run the following command
```
make generate
make manifests
```
to make `crd` files up to date.

## Guide for operator Development
https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/

## Fabric env
for any fabric configuration as core.yaml for peer and orderer.yaml for orderer,  please considering check existing structure defined in configoverride. 

## TODO
