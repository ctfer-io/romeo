# Build stage
FROM golang:1.24.4@sha256:10c131810f80a4802c49cab0961bbe18a16f4bb2fb99ef16deaa23e4246fc817 AS builder

WORKDIR /go/src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ARG VERSION="dev"
ARG COMMIT
ARG DATE
RUN go build -cover \
    -ldflags="-s -w -X 'main.version="$VERSION"' -X 'main.commit="$COMMIT"' -X 'main.date="$DATE"' -X 'main.builtBy=docker'" \
    -o /go/bin/romeo \
    cmd/main.go



# Prod stage
FROM golang:1.24.4@sha256:10c131810f80a4802c49cab0961bbe18a16f4bb2fb99ef16deaa23e4246fc817
COPY --from=builder /go/bin/romeo /romeo
ENTRYPOINT [ "/romeo" ]
