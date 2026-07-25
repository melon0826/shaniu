# 部署指南

本文档介绍 Shaniu 的多种部署方式，包括二进制部署、Docker 部署、systemd 服务化以及反向代理配置。

## 目录

- [二进制部署](#二进制部署)
- [systemd 服务化（Linux）](#systemd-服务化linux)
- [Docker 部署](#docker-部署)
  - [单容器部署](#单容器部署)
  - [Docker Compose](#docker-compose)
- [反向代理](#反向代理)
  - [Nginx](#nginx)
  - [Caddy](#caddy)
- [高可用与集群](#高可用与集群)
- [备份与恢复](#备份与恢复)

## 二进制部署

### 下载与安装

```bash
# 创建应用目录
mkdir -p /opt/shaniu && cd /opt/shaniu

# 下载正式版本（以 Linux amd64 为例）
wget https://github.com/melon0826/shaniu/releases/download/v0.1.0/shaniu_v0.1.0_linux_amd64.tar.gz
tar -xzf shaniu_v0.1.0_linux_amd64.tar.gz
mv shaniu_v0.1.0_linux_amd64 shaniu
chmod +x shaniu

# 测试运行
./shaniu -t
```

### 生产环境运行

生产环境建议使用 `-d` 参数后台运行（不启用终端模式）：

```bash
./shaniu -d
```

或使用 nohup：

```bash
nohup ./shaniu -d > /var/log/shaniu.log 2>&1 &
```

## systemd 服务化（Linux）

创建 systemd 服务文件以实现开机自启和进程守护：

```bash
sudo tee /etc/systemd/system/shaniu.service > /dev/null << 'EOF'
[Unit]
Description=Shaniu Bot Framework
After=network.target

[Service]
Type=simple
User=shaniu
Group=shaniu
WorkingDirectory=/opt/shaniu
ExecStart=/opt/shaniu/shaniu -d
Restart=on-failure
RestartSec=5

# 资源限制
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

# 创建用户
sudo useradd -r -s /bin/false shaniu
sudo chown -R shaniu:shaniu /opt/shaniu

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable --now shaniu
sudo systemctl status shaniu
```

### 常用命令

```bash
sudo systemctl start shaniu    # 启动
sudo systemctl stop shaniu     # 停止
sudo systemctl restart shaniu  # 重启
sudo systemctl status shaniu   # 查看状态
sudo journalctl -u shaniu -f   # 查看日志
```

## Docker 部署

### 单容器部署

```bash
# 拉取镜像
docker pull melon0826/shaniu:latest

# 运行
docker run -d \
  --name shaniu \
  --restart unless-stopped \
  -p 8080:8080 \
  -e SHANIU_DATA_PATH=/data \
  -v $(pwd)/data:/data \
  -v /var/run/docker.sock:/var/run/docker.sock \
  melon0826/shaniu:latest
```

镜像默认使用 `/data` 作为数据目录，`/app/plugins`、`/app/conf` 会指向 `/data` 下的对应目录，所以只需要映射 `./data:/data` 即可持久化数据库、插件和配置。
如果需要通过后台或官方命令插件执行 Docker 自动更新，需要挂载 `/var/run/docker.sock`；不需要容器内更新时可以去掉这一行。

### Docker Compose

创建 `docker-compose.yml`：

```yaml
version: "3.8"

services:
  shaniu:
    image: melon0826/shaniu:latest
    container_name: shaniu
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
      - /var/run/docker.sock:/var/run/docker.sock
      - /etc/localtime:/etc/localtime:ro
    environment:
      - SHANIU_PORT=8080
      - SHANIU_DATA_PATH=/data
    # 如需使用 Redis，取消下面注释
    # depends_on:
    #   - redis

  # redis:
  #   image: redis:7-alpine
  #   container_name: shaniu-redis
  #   restart: unless-stopped
  #   volumes:
  #     - ./redis-data:/data
```

启动：

```bash
docker-compose up -d
docker-compose logs -f
```

### 环境变量

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `SHANIU_PORT` | HTTP 服务端口 | 8080 |
| `SHANIU_REDIS_ADDR` | Redis 地址 | `""` |
| `SHANIU_REDIS_PASSWORD` | Redis 密码 | `""` |

## 反向代理

### Nginx

```nginx
server {
    listen 80;
    server_name bot.example.com;

    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # WebSocket / 长轮询支持
    location /api/web_chat {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }

    # 普通 HTTP 请求
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### HTTPS（Let's Encrypt）

```nginx
server {
    listen 443 ssl http2;
    server_name bot.example.com;

    ssl_certificate /etc/letsencrypt/live/bot.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/bot.example.com/privkey.pem;
    ssl_trusted_certificate /etc/letsencrypt/live/bot.example.com/chain.pem;

    # 其他配置同上...
}

server {
    listen 80;
    server_name bot.example.com;
    return 301 https://$server_name$request_uri;
}
```

### Caddy

```caddy
bot.example.com {
    reverse_proxy localhost:8080
}
```

Caddy 会自动处理 HTTPS 证书申请和续期。

## 高可用与集群

Shaniu 本身是无状态的单机服务，但可以通过以下方式提升可用性：

### 使用 Redis 共享存储

当多个 Shaniu 实例共用同一个 Redis 后端时：
- 插件代码和配置在所有实例间同步
- 用户状态跨实例共享
- 消息路由仍需外部负载均衡器分发

```yaml
# docker-compose.yml 多实例示例
services:
  shaniu-1:
    image: melon0826/shaniu:latest
    environment:
      - SHANIU_REDIS_ADDR=redis:6379
    depends_on:
      - redis

  shaniu-2:
    image: melon0826/shaniu:latest
    environment:
      - SHANIU_REDIS_ADDR=redis:6379
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
```

### 使用外部负载均衡

```nginx
upstream shaniu {
    least_conn;
    server 192.168.1.10:8080;
    server 192.168.1.11:8080;
}

server {
    listen 80;
    location / {
        proxy_pass http://shaniu;
    }
}
```

## 备份与恢复

### BoltDB 备份

```bash
# 备份数据文件
cp /opt/shaniu/shaniu.db /backup/shaniu-$(date +%Y%m%d).db

# 或使用 systemd timer 自动备份
```

### Redis 备份

```bash
# 手动触发 RDB 保存
redis-cli SAVE

# 复制 RDB 文件
cp /var/lib/redis/dump.rdb /backup/redis-$(date +%Y%m%d).rdb
```

### 恢复

```bash
# 停止服务
sudo systemctl stop shaniu

# 恢复数据文件
cp /backup/shaniu-20231001.db /opt/shaniu/shaniu.db

# 重启服务
sudo systemctl start shaniu
```

### 插件代码导出

所有插件代码存储在 Bucket `plugins` 中，可以通过 Admin 面板批量导出，或通过脚本导出：

```bash
# 使用 gRPC 客户端或 REST API 导出
curl "http://localhost:8080/api/bucket/plugins/keys" | jq
```
