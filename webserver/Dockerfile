# Build stage
FROM golang:1.23.6@sha256:927112936d6b496ed95f55f362cc09da6e3e624ef868814c56d55bd7323e0959 AS builder

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
FROM golang:1.23.6@sha256:927112936d6b496ed95f55f362cc09da6e3e624ef868814c56d55bd7323e0959
COPY --from=builder /go/bin/romeo /romeo
ENTRYPOINT [ "/romeo" ]
