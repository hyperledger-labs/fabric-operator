module github.com/IBM-Blockchain/fabric-operator

go 1.20

require (
	github.com/docker/docker v20.10.12+incompatible
	k8s.io/api v0.24.13
	k8s.io/apimachinery v0.24.13
)

require (
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/go-logr/logr v1.2.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/klog/v2 v2.60.1 // indirect
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9 // indirect
	sigs.k8s.io/json v0.0.0-20211208200746-9f7c6b3444d2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)

replace (
	github.com/go-kit/kit => github.com/go-kit/kit v0.8.0 // Needed for fabric-ca
	github.com/gorilla/handlers => github.com/gorilla/handlers v1.4.0 // Needed for fabric-ca
	github.com/gorilla/mux => github.com/gorilla/mux v1.7.3 // Needed for fabric-ca
	github.com/hyperledger/fabric => github.com/hyperledger/fabric v0.0.0-20191027202024-115c7a2205a6
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.0 // Needed for fabric-ca
)
