# Build stage
FROM golang:1.27rc2@sha256:d901fb8d9f6754d898c85bb9331ad76dbe039103b01cf0c068bb31f92e019788 AS builder

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
FROM golang:1.27rc2@sha256:d901fb8d9f6754d898c85bb9331ad76dbe039103b01cf0c068bb31f92e019788
COPY --from=builder /go/bin/romeo /romeo
ENTRYPOINT [ "/romeo" ]
