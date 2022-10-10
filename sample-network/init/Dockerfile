
FROM registry.access.redhat.com/ubi8/ubi-minimal

ARG K8S_BUILDER_TAG

RUN microdnf update \
	&& microdnf install -y \
		shadow-utils \
		iputils \
		tar \
		wget \
	&& groupadd -g 7051 ibp-user \
	&& useradd -u 7051 -g ibp-user -s /bin/bash ibp-user \
	&& microdnf remove shadow-utils \
	&& microdnf clean all

WORKDIR /opt/hyperledger/k8s_builder/bin

RUN wget https://github.com/hyperledger-labs/fabric-builder-k8s/releases/download/${K8S_BUILDER_TAG}/fabric-builder-k8s-${K8S_BUILDER_TAG}-Linux-X64.tgz \
    && tar -xzf fabric-builder-k8s-${K8S_BUILDER_TAG}-Linux-X64.tgz \
	&& rm -rf fabric-builder-k8s-${K8S_BUILDER_TAG}-Linux-X64.tgz