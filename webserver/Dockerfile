# Build stage
FROM golang:1.24.1@sha256:52ff1b35ff8de185bf9fd26c70077190cd0bed1e9f16a2d498ce907e5c421268 AS builder

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
FROM golang:1.24.1@sha256:52ff1b35ff8de185bf9fd26c70077190cd0bed1e9f16a2d498ce907e5c421268
COPY --from=builder /go/bin/romeo /romeo
ENTRYPOINT [ "/romeo" ]
