FROM --platform=$BUILDPLATFORM golang AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /w
COPY . .
ENV CGO_ENABLED=0
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go build -o /tmp/cu ./cmd/cu

FROM scratch
COPY --from=builder /tmp/cu .
