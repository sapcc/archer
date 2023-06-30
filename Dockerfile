FROM golang:1.20-alpine as builder

ENV CGO_ENABLED=0
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache make
COPY . /src
RUN make -C /src

################################################################################

FROM alpine:3.18
LABEL source_repository="https://github.com/sapcc/archer"

RUN apk add --no-cache ca-certificates haproxy
COPY --from=builder /src/bin/ /usr/bin/
ENTRYPOINT [ "/usr/bin/archer-server" ]
