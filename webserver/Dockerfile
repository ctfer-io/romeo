# Build stage
FROM golang:1.24.3@sha256:86b4cff66e04d41821a17cea30c1031ed53e2635e2be99ae0b4a7d69336b5063 AS builder

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
FROM golang:1.24.3@sha256:86b4cff66e04d41821a17cea30c1031ed53e2635e2be99ae0b4a7d69336b5063
COPY --from=builder /go/bin/romeo /romeo
ENTRYPOINT [ "/romeo" ]
