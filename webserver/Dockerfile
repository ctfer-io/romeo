# Build stage
FROM golang:1.27rc3@sha256:6a284ec7a8c67eff5882aaa53b07a57ee397553f8c85a92e5fc97c517f12201f AS builder

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
FROM golang:1.27rc3@sha256:6a284ec7a8c67eff5882aaa53b07a57ee397553f8c85a92e5fc97c517f12201f
COPY --from=builder /go/bin/romeo /romeo
ENTRYPOINT [ "/romeo" ]
