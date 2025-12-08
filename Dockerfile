# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.25.5-alpine3.23 AS builder

RUN apk add --no-cache --no-progress ca-certificates gcc git make musl-dev

COPY . /src
ARG BININFO_BUILD_DATE BININFO_COMMIT_HASH BININFO_VERSION # provided to 'make install'
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
  make -C /src install PREFIX=/pkg GOTOOLCHAIN=local

RUN wget https://cacerts.digicert.com/DigiCertGlobalG2TLSRSASHA2562020CA1-1.crt.pem -O /usr/local/share/ca-certificates/zDigiCertGlobalG2TLSRSASHA2562020CA1-1.crt.pem \
  && update-ca-certificates


################################################################################

# To only build the tests run: docker build . --target test
# We can't do `FROM builder AS test` here, as then make prepare-static-check would not be cached during interactive use when developing
# and caching all the tools, especially golangci-lint, takes a few minutes.
FROM golang:1.25.5-alpine3.23 AS test

COPY Makefile /src/Makefile

# used below by USER
RUN addgroup -g 4200 appgroup \
  && adduser -h /home/appuser -s /sbin/nologin -G appgroup -D -u 4200 appuser

RUN apk add --no-cache --no-progress git make py3-pip postgresql \
  && pip3 install --break-system-packages reuse \
  && make -C /src prepare-static-check


# We only copy here because we want the "prepare-static-check" to be cacheable.
# It is not a problem that we are overwriting the go cache from the earlier steps because we do not need to rebuild those tools.
COPY --from=builder /go /go
COPY --from=builder /src /src

RUN make -C /src static-check

# Some things like postgres do not like to run as root. For simplicity, just always run as an unprivileged user,
# but for it to be able to read the go cache, we need to allow it.
RUN chown -R 4200:4200 /src/ /go/
USER 4200:4200
RUN cd /src \
  && git config --global --add safe.directory /src \
  && make build/cover.out

################################################################################

FROM alpine:3.23

# upgrade all installed packages to fix potential CVEs in advance
# also remove apk package manager to hopefully remove dependency on OpenSSL 🤞
RUN apk upgrade --no-cache --no-progress \
  && apk add --no-cache --no-progress haproxy \
  && wget -qO /usr/bin/linkerd-await https://github.com/linkerd/linkerd-await/releases/download/release%2Fv0.2.7/linkerd-await-v0.2.7-amd64 \
  && chmod 755 /usr/bin/linkerd-await \
  && apk del --no-cache --no-progress apk-tools alpine-keys alpine-release libc-utils

COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=builder /etc/ssl/cert.pem /etc/ssl/cert.pem
COPY --from=builder /pkg/ /usr/
# make sure all binaries can be executed
RUN set -x \
  && archer-server --version 2>/dev/null \
  && archer-f5-agent --version 2>/dev/null \
  && archerctl --version 2>/dev/null \
  && archer-migrate --version 2>/dev/null \
  && archer-ni-agent --version 2>/dev/null

ARG BININFO_BUILD_DATE BININFO_COMMIT_HASH BININFO_VERSION
LABEL source_repository="https://github.com/sapcc/archer" \
  org.opencontainers.image.url="https://github.com/sapcc/archer" \
  org.opencontainers.image.created=${BININFO_BUILD_DATE} \
  org.opencontainers.image.revision=${BININFO_COMMIT_HASH} \
  org.opencontainers.image.version=${BININFO_VERSION}

WORKDIR /
ENTRYPOINT [ "/usr/bin/linkerd-await", "--shutdown", "--", "/usr/bin/archer-server" ]
