# SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
#
# SPDX-License-Identifier: Apache-2.0

# Create a base layer with linkerd-await from a recent release.
FROM docker.io/curlimages/curl:latest AS linkerd
ARG LINKERD_AWAIT_VERSION=v0.2.9
RUN curl -sSLo /tmp/linkerd-await https://github.com/linkerd/linkerd-await/releases/download/release%2F${LINKERD_AWAIT_VERSION}/linkerd-await-${LINKERD_AWAIT_VERSION}-amd64 && \
    chmod 755 /tmp/linkerd-await

################################################################################


FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache make git
COPY . /src
RUN make -C /src

################################################################################

FROM alpine:3.22
LABEL source_repository="https://github.com/sapcc/archer"

# upgrade all installed packages to fix potential CVEs in advance
RUN apk upgrade --no-cache --no-progress \
  && apk add --no-cache ca-certificates haproxy \
  && wget https://cacerts.digicert.com/DigiCertGlobalG2TLSRSASHA2562020CA1-1.crt.pem -O /usr/local/share/ca-certificates/zDigiCertGlobalG2TLSRSASHA2562020CA1-1.crt.pem \
  && update-ca-certificates
COPY --from=builder /src/build/ /usr/bin/
COPY --from=linkerd /tmp/linkerd-await /linkerd-await
ENTRYPOINT [ "/usr/bin/archer-server" ]
