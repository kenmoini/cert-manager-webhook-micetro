FROM registry.access.redhat.com/ubi10/go-toolset:1786496358 AS build_deps

WORKDIR /workspace

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM build_deps AS build

COPY . .

RUN CGO_ENABLED=0 go build -o cert-manager-webhook-micetro -ldflags '-w -extldflags "-static"' .

FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk add --no-cache ca-certificates

COPY --from=build /workspace/cert-manager-webhook-micetro /usr/local/bin/cert-manager-webhook-micetro

USER 1001

ENTRYPOINT ["cert-manager-webhook-micetro"]
