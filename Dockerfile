FROM golang:1.11 AS builder

# Install dep
RUN curl -L https://github.com/golang/dep/releases/download/v0.5.1/dep-linux-amd64 > /usr/local/bin/dep && \
  chmod a+x /usr/local/bin/dep

COPY . /go/src/github.com/govau/torque

# If we don't disable CGO, the binary won't work. Unsure why?
WORKDIR /go/src/github.com/govau/torque

RUN dep ensure && \
  go test ./... && \
  CGO_ENABLED=0 go install

FROM alpine:3.9

RUN apk add --update \
  bash \
  curl \
  git \
  jq \
  && \
  rm -rf /var/cache/apk/*

COPY --from=builder /go/bin/torque /usr/bin/torque

ENTRYPOINT [ "/usr/bin/torque"]
