FROM golang:1.25-alpine AS build-env

ARG COMMIT_SHA=dev
ENV CGO_ENABLED=0

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/project-00 project-00/
COPY internal/ internal/
COPY VERSION ./
RUN VERSION=$(cat VERSION) && \
    go build -o app -ldflags "-X main.AppVersion=${VERSION}-${COMMIT_SHA}" ./project-00

FROM alpine:3.22

WORKDIR /go/bin
RUN apk add --no-cache ca-certificates
COPY application.yml /go/bin/application.yml
COPY --from=build-env /build/app .

ENTRYPOINT ["/go/bin/app"]
