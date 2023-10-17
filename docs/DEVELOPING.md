# Fabric Operator Development

## Prerequisites

- Golang 1.17+
- A good IDE, any will do.  VSCode and GoLand are great tools - use them!
- A healthy mix of patience, curiosity, and ingenuity.
- A strong desire to make Fabric ... _right_.  
- Check your ego at the door.


## Build the Operator

```shell
# Let's Go! 
make
```

```shell
# Build ghcr.io/ibm-blockchain/fabric-operator:latest-amd64 
make image
```

```shell
# Build Fabric CRDs
make manifests
```

## Unit Tests

```shell
# Just like it says: 
make test
```


## Integration Tests

Integration tests run the operator binary _locally_ as a native process, connecting to a "remote" Kube API 
controller.

```shell
# point the operator at a kubernetes 
export KUBECONFIG_PATH=$HOME/.kube/config

make integration-tests
```


Or focus on a targeted suite: 
```shell
INT_TEST_NAME=<folder under /integration> make integration-tests 
```


## Debug the Operator

Launch main.go with the following environment:
```shell
export KUBECONFIG=$HOME/.kube/config
export WATCH_NAMESPACE=test-network
export CLUSTERTYPE=k8s
export OPERATOR_LOCAL_MODE=true

go run .
```


## Local Kube Options:

### Rancher / k3s

[Rancher Desktop](https://rancherdesktop.io) is a _fantastic_ alternative for running a local Kubernetes on
_either_ containerd _or_ mobyd / Docker.

It's great.

Use it.

Learn to love typing `nerdctl --namespace k8s.io`, providing a direct line of sight for k3s to read directly from
the local image cache.


### KIND
```shell
# Create a KIND cluster - suitable for integration testing.
make kind

# Why?
make unkind
```

OR ... create a KIND cluster pre-configured with Nginx ingress and Fabric CRDs:
```shell
sample-network/network kind
sample-network/network cluster init
```

Note that KIND does not have [visibility to images](https://iximiuz.com/en/posts/kubernetes-kind-load-docker-image/) 
in the local Docker cache.  If you build an image, make sure to directly load it into the KIND image plane
(`kind load docker-image ...`) AND set `imagePullPolicy: IfNotPresent` in any Kube spec referencing the container.

Running `network kind` will deploy a companion, insecure Docker registry at `localhost:5000`.  This can be
_extremely useful_ for relaying custom images into the cluster when the imagePullPolicy can not be overridden.
If for some reason you can't seem to mangle an image into KIND, build, tag, and push the custom image over to
the `localhost:5000` container registry.  (Or use Rancher/k3s.)


## What's up with Ingress, localho.st, vcap.me, and nip.io domains?

Fabric Operator uses Kube Ingress to route traffic through a common, DNS wildcard domain (e.g. *.my-network.example.com.)
In cloud-native environments, where a DNS wildcard domain resolvers are readily available, it is possible to 
map a top-level A record to a single IP address bound to the cluster ingress.

Unfortunately it is _exceedingly annoying_ to emulate a top-level A wildcard DNS domain in a way that can be visible
to pods running in a Docker network (e.g. KIND) AND to the host OS using the same domain alias and IP.

Alternate solutions available:

- Use the `*.localho.st` domain alias for your Fabric network, mapping all sub-domains and hosts to 127.0.0.1.

- Use the `*.vcap.me` domain alias for your Fabric network, mapping to 127.0.0.1 in all cases.  This is convenient for
  scenarios where pods in the cluster will have no need to traverse the ingress (e.g. in integration testing).
  (Update: vcap.me stopped resolving host names some time in late 2022.)

- Use the [Dead simple wildcard DNS for any IP Address](https://nip.io) *.nip.io domain for the cluster, providing 
  full flexibility for the IP address of the ingress port.


## Commit Practices

- There is no "Q/A" team, other than the "A Team" : you.  
- When you write a new feature, develop BOTH unit tests and a functional / integration test.
- When you find a bug, write a regression/unit test to illuminate it, and step on it AFTER it's in the spotlight.
- Submit PRs in tandem with GitHub Issues describing the feature, fix, or enhancement.
- Don't allow PRs to linger.
- Ask your peers and maintainers to review PRs.  Be efficient by including solid test cases.
- Have fun, and learn something new from your peers.


## Pitfalls / Hacks / Tips / Tricks 

- On OSX, there is a bug in the Golang DNS resolver, causing the Fabric binaries to stall out when resolving DNS.
  See [Fabric #3372](https://github.com/hyperledger/fabric/issues/3372) and [Golang #43398](https://github.com/golang/go/issues/43398).
  Fix this by turning a build of [fabric](https://github.com/hyperledger/fabric) binaries and copying the build outputs
  from `fabric/build/bin/*` --> `sample-network/bin`


- ???