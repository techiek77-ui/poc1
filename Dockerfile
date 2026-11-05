# Linux pod image: pure-Go scanner (rules + Tier A) by default; add the ONNX tier
# with --build-arg ONNX=1 (CGO build + ONNX Runtime shared library).
#
#   docker build -t pii-poc .                                                          # rules + Tier A, static binary, distroless
#   docker build -t pii-poc:onnx --build-arg ONNX=1 --build-arg RUNTIME=runtime-onnx .  # + Tier B via ONNX Runtime (piiscan -onnx <dir>)
ARG ONNX=0
ARG ORT_VERSION=1.31.1
# RUNTIME selects the final image: runtime-static (default) or runtime-onnx
ARG RUNTIME=runtime-static

FROM golang:1.24-bookworm AS build
ARG ONNX
ARG ORT_VERSION
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN if [ "$ONNX" = "1" ]; then \
      CGO_ENABLED=1 go build -tags onnx -trimpath -ldflags="-s -w" -o /out/piiscan ./cmd/piiscan && \
      CGO_ENABLED=1 go build -tags onnx -trimpath -ldflags="-s -w" -o /out/piibench ./cmd/piibench && \
      curl -sSL -o /tmp/ort.tgz https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/onnxruntime-linux-x64-${ORT_VERSION}.tgz && \
      mkdir -p /out/lib && tar xzf /tmp/ort.tgz -C /tmp && cp /tmp/onnxruntime-linux-x64-${ORT_VERSION}/lib/libonnxruntime.so* /out/lib/ ; \
    else \
      CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/piiscan ./cmd/piiscan ; \
    fi

# Runtime images; the final stage picks one by the RUNTIME build arg.
FROM debian:bookworm-slim AS runtime-onnx
COPY --from=build /out/ /app/
COPY models/tier_a_lr.json /models/tier_a_lr.json
COPY models/tier_b /models/tier_b
ENV PII_ORT_LIB=/app/lib/libonnxruntime.so PII_ORT_THREADS=1
ENTRYPOINT ["/app/piiscan", "-model", "/models/tier_a_lr.json", "-onnx", "/models/tier_b"]

FROM gcr.io/distroless/static-debian12 AS runtime-static
COPY --from=build /out/piiscan /piiscan
COPY models/tier_a_lr.json /models/tier_a_lr.json
ENTRYPOINT ["/piiscan", "-model", "/models/tier_a_lr.json"]

# Final image = the selected runtime stage.
FROM ${RUNTIME} AS final
