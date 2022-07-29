module github.com/IBM-Blockchain/fabric-operator

go 1.18

require (
	github.com/docker/docker v20.10.12+incompatible
	k8s.io/api v0.21.5
	k8s.io/apimachinery v0.21.5
)

require (
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	golang.org/x/net v0.0.0-20210917221730-978cfadd31cf // indirect
	golang.org/x/text v0.3.7 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/klog/v2 v2.8.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
)

replace (
	github.com/go-kit/kit => github.com/go-kit/kit v0.8.0 // Needed for fabric-ca
	github.com/gorilla/handlers => github.com/gorilla/handlers v1.4.0 // Needed for fabric-ca
	github.com/gorilla/mux => github.com/gorilla/mux v1.7.3 // Needed for fabric-ca
	github.com/hyperledger/fabric => github.com/hyperledger/fabric v0.0.0-20191027202024-115c7a2205a6
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.0 // Needed for fabric-ca
)
