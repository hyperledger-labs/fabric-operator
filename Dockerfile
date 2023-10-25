ARG GO_VER

########## Build operator binary ##########
FROM registry.access.redhat.com/ubi8/go-toolset:$GO_VER as builder

COPY . /go/src/github.com/IBM-Blockchain/fabric-operator
WORKDIR /go/src/github.com/IBM-Blockchain/fabric-operator
RUN GOOS=linux GOARCH=${ARCH} CGO_ENABLED=1 go build -mod=vendor -tags "pkcs11" -gcflags all=-trimpath=${GOPATH} -asmflags all=-trimpath=${GOPATH} -o /tmp/build/_output/bin/ibp-operator

########## Final Image ##########
FROM registry.access.redhat.com/ubi8/ubi-minimal

ENV OPERATOR=/usr/local/bin/ibp-operator

COPY --from=builder /tmp/build/_output/bin/ibp-operator ${OPERATOR}
COPY build/ /usr/local/bin
COPY definitions /definitions
COPY config/crd/bases /deploy/crds
COPY defaultconfig /defaultconfig
COPY docker-entrypoint.sh .

RUN microdnf update \
    && microdnf install -y \
    shadow-utils \
    iputils \
    && groupadd -g 7051 fabric-user \
    && useradd -u 7051 -g fabric-user -s /bin/bash fabric-user \
    && mkdir /licenses \
    && microdnf remove shadow-utils \
    && microdnf clean all \
    && chown -R fabric-user:fabric-user licenses \
    && /usr/local/bin/user_setup

USER fabric-user
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/usr/local/bin/entrypoint"]