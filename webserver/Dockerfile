# Build stage
FROM golang:1.26rc2@sha256:6ea9974028bfaaa0251e42468ec037fe21e49275a874914f3b42f58beabc84bf AS builder

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
FROM golang:1.26rc2@sha256:6ea9974028bfaaa0251e42468ec037fe21e49275a874914f3b42f58beabc84bf
COPY --from=builder /go/bin/romeo /romeo
ENTRYPOINT [ "/romeo" ]
