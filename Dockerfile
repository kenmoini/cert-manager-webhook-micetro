# ===========================================================================
# Build Stage
# ===========================================================================
FROM registry.access.redhat.com/ubi10/go-toolset:1788213739 AS build_deps

WORKDIR /workspace
USER 0

RUN dnf install -y git openssl openssl-devel && \
    dnf clean all && \
    rm -rf /var/cache/dnf

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM build_deps AS build

COPY . .

#RUN CGO_ENABLED=0 go build -o cert-manager-webhook-micetro -buildvcs=false -ldflags '-w -extldflags "-static"' .
# The following build command is used to enable FIPS mode usage.
# https://developers.redhat.com/articles/2025/01/23/fips-mode-red-hat-go-toolset?source=sso
RUN CGO_ENABLED=1 go build -o cert-manager-webhook-micetro -buildvcs=false -ldflags '-w' .

# ===========================================================================
# Final Stage
# ===========================================================================
FROM registry.access.redhat.com/ubi10/ubi-minimal:1788137827

USER 0
RUN microdnf install -y ca-certificates openssl && \
    update-ca-trust && \
    microdnf clean all && \
    rm -rf /var/cache/dnf

COPY --from=build /workspace/cert-manager-webhook-micetro /usr/local/bin/cert-manager-webhook-micetro

USER 1001

ENTRYPOINT ["cert-manager-webhook-micetro"]