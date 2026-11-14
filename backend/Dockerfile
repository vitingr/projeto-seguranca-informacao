# syntax=docker/dockerfile:1

# ---- build -------------------------------------------------------------------
# Mantenha a versão alinhada ao go.mod e ao CI (ver README, "Versão do Go").
FROM golang:1.27 AS build

WORKDIR /src

# Binário estático: sem cgo, sem dependência de libc na imagem final.
ENV CGO_ENABLED=0 \
    GOFLAGS=-trimpath

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

COPY cmd ./cmd
COPY internal ./internal

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -o /out/backend_si ./cmd/server

# ---- runtime -----------------------------------------------------------------
# distroless/static: sem shell, sem gerenciador de pacotes, apenas certificados
# raiz e tzdata. A tag :nonroot já define o usuário 65532.
FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /out/backend_si /backend_si

USER 65532:65532
EXPOSE 8443

# A imagem não tem curl: o próprio binário faz o healthcheck via HTTPS,
# validando o certificado do servidor.
HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/backend_si", "healthcheck"]

ENTRYPOINT ["/backend_si"]
CMD ["serve"]
