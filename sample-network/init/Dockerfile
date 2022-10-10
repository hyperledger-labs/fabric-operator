ARG K8S_TAG
FROM ghcr.io/hyperledgendary/k8s-fabric-peer:${K8S_TAG} as builder

FROM registry.access.redhat.com/ubi8/ubi-minimal

RUN microdnf update \
	&& microdnf install -y \
		shadow-utils \
		iputils \
		tar \
	&& groupadd -g 7051 ibp-user \
	&& useradd -u 7051 -g ibp-user -s /bin/bash ibp-user \
	&& microdnf remove shadow-utils \
	&& microdnf clean all \
	&& mkdir -p /opt/hyperledger/k8s_builder/bin;

COPY --from=builder /opt/hyperledger/k8s_builder/bin/ /opt/hyperledger/k8s_builder/bin/