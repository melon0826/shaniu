FROM node:24-bookworm-slim AS frontend
WORKDIR /src

COPY frontend/package*.json ./frontend/
RUN --mount=type=cache,target=/root/.npm \
    cd frontend && if [ -f package-lock.json ]; then npm ci; else npm install; fi

COPY frontend ./frontend
RUN mkdir -p core/admin && cd frontend && npm run build

FROM golang:1.26.5-bookworm AS builder
WORKDIR /src

ARG VERSION=dev

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
COPY --from=frontend /src/core/admin ./core/admin

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X github.com/melon0826/shaniu/core.compiled_at=${VERSION}" \
    -o /out/shaniu .

FROM node:24-bookworm-slim
WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && corepack enable \
    && corepack prepare pnpm@11.16.0 --activate \
    && mkdir -p /app/node-runtime \
    && cd /app/node-runtime \
    && printf '{"name":"shaniu-node-runtime","private":true,"version":"1.0.0"}\n' > package.json \
    && printf 'packages:\n  - .\nallowBuilds:\n  protobufjs: true\n' > pnpm-workspace.yaml \
    && pnpm --allow-build=protobufjs add @grpc/grpc-js@^1.8.18 express@^4.21.2 google-protobuf@^3.21.2 \
    && mkdir -p /data/plugins /data/conf \
    && ln -s /data/plugins /app/plugins \
    && ln -s /data/conf /app/conf

COPY --from=builder /out/shaniu /app/shaniu
COPY --from=builder /src/proto3/shaniu.js /app/proto3/shaniu.js
COPY --from=builder /src/proto3/shaniu.d.ts /app/proto3/shaniu.d.ts
COPY --from=builder /src/proto3/srpc.js /app/proto3/srpc.js
COPY --from=builder /src/proto3/shaniu.py /app/proto3/shaniu.py
COPY --from=builder /src/proto3/srpc_pb2.py /app/proto3/srpc_pb2.py
COPY --from=builder /src/proto3/srpc_pb2_grpc.py /app/proto3/srpc_pb2_grpc.py

ENV TZ=Asia/Shanghai \
    SHANIU_DATA_PATH=/data \
    SHANIU_NODE_PATH=/app/node-runtime/node_modules \
    NODE_PATH=/app/node-runtime/node_modules
EXPOSE 8686 50051
VOLUME ["/data"]

ENTRYPOINT ["sh", "-c", "mkdir -p /data/plugins /data/conf && rm -rf /data/language/node && rmdir /data/language 2>/dev/null || true; exec /app/shaniu \"$@\"", "--"]
