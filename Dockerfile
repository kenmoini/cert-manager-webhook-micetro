FROM registry.access.redhat.com/ubi10/go-toolset:1787531096 AS build_deps

WORKDIR /workspace
USER 0

RUN dnf install -y git

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM build_deps AS build

COPY . .

RUN CGO_ENABLED=0 go build -o cert-manager-webhook-micetro -buildvcs=false -ldflags '-w -extldflags "-static"' .

FROM registry.access.redhat.com/ubi10/ubi-minimal:1786398666
USER 0
RUN microdnf install -y ca-certificates && \
    update-ca-trust && \
    microdnf clean all && \
    rm -rf /var/cache/dnf

COPY --from=build /workspace/cert-manager-webhook-micetro /usr/local/bin/cert-manager-webhook-micetro

USER 1001

ENTRYPOINT ["cert-manager-webhook-micetro"]