FROM debian:11

RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list && \
    sed -i 's/security.debian.org/mirrors.aliyun.com\/debian-security/g' /etc/apt/sources.list

RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    php \
    php-json \
    php-xml \
    php-mbstring \
    php-curl \
    php-gd \
    php-zip \
    php-mysql \
    php-pgsql \
    php-redis \
    php-pear \
    php-dev \
    curl \
    wget \
    git

# 安装gRPC扩展
# RUN pecl install grpc

# 清理缓存和临时文件
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# 下载文件
RUN mkdir -p /usr/local/shaniu \
    && ARCH=$(uname -m) \
    && DOWNLOAD_URL="" \
    && if [ "$ARCH" = "x86_64" ]; then \
        DOWNLOAD_URL="/releases/download/main/shaniu_linux_amd64"; \
    elif [ "$ARCH" = "aarch64" ]; then \
        DOWNLOAD_URL="/releases/download/main/shaniu_linux_arm64"; \
    elif [ "$ARCH" = "armv7l" ]; then \
        DOWNLOAD_URL="/releases/download/main/shaniu_linux_armv7"; \
    else \
        echo "Unsupported architecture: $ARCH"; \
        exit 1; \
    fi \
    && curl -L -sSL -f "https://gitee.com/shaniubot/shaniu$DOWNLOAD_URL" -o /usr/local/shaniu/shaniu \
    || (echo "Download from original address failed, trying proxy address..." \
    && curl -sSL --connect-timeout 10 -f "https://github.com/smallfawn/shaniu$DOWNLOAD_URL" -o /usr/local/shaniu/shaniu) \
    && chmod +x /usr/local/shaniu/shaniu

# 设置工作目录
WORKDIR /usr/local/shaniu

ENV PATH="/usr/local/shaniu/language/node/yarn/bin:${PATH}"
ENV PATH="/usr/local/shaniu/language/node:${PATH}"
ENV SHANIU_DATA_PATH=/usr/local/shaniu/

# 指定容器启动时要运行的命令
# CMD ["/usr/local/shaniu/shaniu", "-t"]

CMD ["/usr/local/shaniu/shaniu -t"]

# docker build -t smallfawn/shaniu .
# docker run -d --restart always --name shaniu smallfawn/shaniu
