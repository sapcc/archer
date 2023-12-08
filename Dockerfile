# Create a base layer with linkerd-await from a recent release.
FROM docker.io/curlimages/curl:latest as linkerd
ARG LINKERD_AWAIT_VERSION=v0.2.7
RUN curl -sSLo /tmp/linkerd-await https://github.com/linkerd/linkerd-await/releases/download/release%2F${LINKERD_AWAIT_VERSION}/linkerd-await-${LINKERD_AWAIT_VERSION}-amd64 && \
    chmod 755 /tmp/linkerd-await

################################################################################


FROM golang:1.21-alpine as builder

ENV CGO_ENABLED=0
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache make git
COPY . /src
RUN make -C /src

################################################################################

FROM alpine:3.19
LABEL source_repository="https://github.com/sapcc/archer"

RUN apk add --no-cache ca-certificates haproxy
COPY --from=builder /src/bin/ /usr/bin/
COPY --from=linkerd /tmp/linkerd-await /linkerd-await
ENTRYPOINT [ "/usr/bin/archer-server" ]
