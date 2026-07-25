FROM node:24-bookworm-slim AS frontend
WORKDIR /src

COPY frontend/package*.json ./frontend/
RUN cd frontend && npm install

COPY frontend ./frontend
RUN mkdir -p core/admin && cd frontend && npm run build

FROM golang:1.25-bookworm AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /src/core/admin ./core/admin

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w" \
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
    && pnpm add --ignore-scripts @grpc/grpc-js@^1.8.18 express@^4.21.2 google-protobuf@^3.21.2 \
    && mkdir -p /data/plugins /data/conf \
    && ln -s /data/plugins /app/plugins \
    && ln -s /data/conf /app/conf

COPY --from=builder /out/shaniu /app/shaniu
COPY --from=builder /src/proto3/shaniu.js /app/proto3/shaniu.js
COPY --from=builder /src/proto3/shaniu.d.ts /app/proto3/shaniu.d.ts
COPY --from=builder /src/proto3/srpc.js /app/proto3/srpc.js

ENV TZ=Asia/Shanghai \
    SHANIU_DATA_PATH=/data \
    SHANIU_NODE_PATH=/app/node-runtime/node_modules \
    NODE_PATH=/app/node-runtime/node_modules
EXPOSE 8686 50051
VOLUME ["/data"]

ENTRYPOINT ["sh", "-c", "mkdir -p /data/plugins /data/conf && rm -rf /data/language/node && rmdir /data/language 2>/dev/null || true; exec /app/shaniu \"$@\"", "--"]
