FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=none
ARG BUILTAT=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X github.com/shoresh319/firefly/pkg/version.Version=${VERSION} -X github.com/shoresh319/firefly/pkg/version.Commit=${COMMIT} -X github.com/shoresh319/firefly/pkg/version.BuiltAt=${BUILTAT}" -o /firefly ./cmd/firefly

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /firefly /firefly
EXPOSE 8080
ENV PORT=8080
USER nonroot:nonroot
ENTRYPOINT ["/firefly"]


