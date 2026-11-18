# openhongguo-signer 部署

openhongguo-signer 是签名服务。它在 unidbg 里运行 App 自带的 `libmetasec_ml.so`，为请求生成 X-Gorgon、X-Helios、X-Medusa 等签名头。本文说明怎样用 Docker 部署它。

部署包含两个容器：

- `openhongguo-signer`：签名服务，在容器内监听 `0.0.0.0:19830`；
- `caddy`：反向代理，为 `SIGNER_DOMAIN` 自动申请证书，对外提供 HTTPS（443 端口）。

```text
远程调用方 --HTTPS--> caddy :443 --> openhongguo-signer :19830
本机调用方 ---------------------------> 127.0.0.1:19830
```

签名服务自身的端口在宿主机上映射到哪个地址，由 `SIGNER_BIND` 决定：

| `SIGNER_BIND` | 谁能访问 | 适用场景 |
|---|---|---|
| `127.0.0.1`（默认） | 只有这台服务器 | 远程调用方走 caddy 的 HTTPS，或者通过 SSH 隧道访问 |
| `0.0.0.0` | 所有能连到这台服务器的机器 | 不用 caddy、直接用 HTTP 调用 |

caddy 通过 compose 内部网络访问签名服务，不受 `SIGNER_BIND` 影响。

> **警告**：
>
> - 能访问服务的人都能签任意请求。对外开放时（经过 caddy 或者 `SIGNER_BIND=0.0.0.0`），建议设置 `SIGNER_TOKEN`。
> - 映射到 `0.0.0.0` 时，请用云服务器的安全组限制来源 IP。Docker 写入的 iptables 规则会绕过 ufw，ufw 挡不住这个端口。

## 准备

需要一台 Linux 服务器，已安装 Docker 和 Docker Compose 插件。镜像支持 amd64 和 arm64。

使用 caddy 时，还需要一个域名，A 记录指向这台服务器，并且放行 80 和 443 端口（80 端口用于申请证书）。

## 镜像

每次发布 SDK 新版本（GitHub Release 发布）时，`.github/workflows/signer.yml` 会构建镜像，推送到 Docker Hub：

```text
<Docker Hub 用户名>/openhongguo-signer:<版本 tag>   例如 v0.3.1
<Docker Hub 用户名>/openhongguo-signer:latest
```

已发布的版本可以在 Actions 页面手动运行 signer 工作流，填入 tag 重新构建。

工作流需要在仓库设置里配置：

| 名称 | 类型 | 说明 |
|---|---|---|
| `DOCKERHUB_USERNAME` | Variable | Docker Hub 用户名，也是镜像的命名空间 |
| `DOCKERHUB_TOKEN` | Secret | Docker Hub 访问令牌，需要 Read & Write 权限 |

镜像仓库第一次推送时自动创建。是否公开，在 Docker Hub 的仓库设置里修改。

## 部署

1. 把 `deploy` 目录复制到服务器：

   ```sh
   rsync -a tools/openhongguo-signer/deploy/ server:/opt/openhongguo-signer/deploy/
   ```

2. 在服务器上写配置：

   ```sh
   cd /opt/openhongguo-signer/deploy
   cp .env.example .env
   ```

   需要修改的项：

   - `SIGNER_IMAGE`：要运行的镜像，例如 `<Docker Hub 用户名>/openhongguo-signer:v0.3.1`；
   - `SIGNER_DOMAIN`：caddy 使用的域名；
   - `SIGNER_TOKEN`：对外开放时建议设置，可以这样生成：

     ```sh
     sed -i "s/^SIGNER_TOKEN=.*/SIGNER_TOKEN=$(openssl rand -hex 32)/" .env
     ```

   Docker Hub 仓库是私有的话，先在服务器上运行 `docker login`。

3. 启动：

   ```sh
   docker compose up -d
   ```

   服务启动大约需要 10～20 秒。caddy 第一次申请证书也需要几秒钟。

   不需要 caddy 时，只启动签名服务：

   ```sh
   docker compose up -d openhongguo-signer
   ```

### 从源码构建

不使用 Docker Hub 上的镜像时，把整个 `tools/openhongguo-signer` 复制到服务器，`SIGNER_IMAGE` 留空，然后构建并启动：

```sh
rsync -a --exclude build --exclude .gradle --exclude 'hs_err*' \
  tools/openhongguo-signer/ server:/opt/openhongguo-signer/
ssh server 'cd /opt/openhongguo-signer/deploy && docker compose up -d --build'
```

第一次构建要下载 Gradle 和依赖，需要几分钟。

如果服务器不方便访问 Maven Central，可以在本机构建镜像，再传到服务器：

```sh
cd tools/openhongguo-signer
docker buildx build --platform linux/amd64 -f deploy/Dockerfile -t openhongguo-signer:latest --load .
docker save openhongguo-signer:latest | gzip | ssh server 'gunzip | docker load'
```

传过去以后，在服务器上用 `docker compose up -d` 启动，不加 `--build`。

## 检查

在服务器上检查服务是否就绪：

```sh
docker compose ps                       # openhongguo-signer 显示 healthy
curl -s http://127.0.0.1:19830/ready    # {"service":"openhongguo-signer","ready":true}
```

经过 caddy 时，在任意机器上检查：

```sh
curl -s https://<SIGNER_DOMAIN>/ready
```

在本机跑对照测试，确认签名能被服务端接受。没有设置 token 时省略 `SIGNER_TOKEN`：

```sh
SIGNER_TOKEN=<token> tools/openhongguo-signer/scripts/compare.sh --relay https://<SIGNER_DOMAIN>
```

输出 `MATCH` 表示 Go 签名和签名服务的结果一致。详细说明见 `compare/main.go` 开头的注释。

## 接口

所有响应都带 `Server: openhongguo-signer` 头。

### `GET /ready`

不需要 token。签名句柄就绪后返回 `{"service":"openhongguo-signer","ready":true}`，供健康检查使用。

### `POST /feature-hash`

设置了 `SIGNER_TOKEN` 时，请求必须带 `Authorization: Bearer <token>`。

请求体是 JSON：

| 字段 | 说明 |
|---|---|
| `url` | 要发出的完整 URL，包括查询参数。服务从中读取 `device_id` 和 `iid`，作为签名里的设备身份 |
| `body_hex` | 请求体的十六进制编码，按实际发送的字节计算（例如 gzip 之后）。GET 请求省略 |

示例：

```sh
curl -s http://127.0.0.1:19830/feature-hash \
  -H "Authorization: Bearer $SIGNER_TOKEN" \
  -d '{"url":"https://api5-normal-lq.fqnovel.com/reading/bookapi/search/tab/v?device_id=...&iid=...&_rticket=..."}'
```

成功时返回：

```json
{"headers":{"X-Gorgon":"...","X-Helios":"...","X-Medusa":"...","X-Khronos":"...","X-Argus":"...","X-Ladon":"..."}}
```

调用方用这些值替换请求里的同名头，然后马上发出请求。X-Khronos 是签名时的时间戳。

失败时返回 `{"error":"openhongguo-signer: <原因>"}`：

| 状态码 | 原因 | 含义 |
|---|---|---|
| 400 | `url is required` | 缺少 `url` |
| 401 | `unauthorized` | token 缺失或错误 |
| 405 | `method not allowed` | 请求方法不对 |
| 422 | `signer refused the request` | SDK 拒绝签这个请求。`neptune` 字段是错误码，例如 `-7` 表示 URL 无效 |
| 503 | `signer restarting` | 服务连续签名失败，正在退出重启 |
| 500 | `internal error: ...` | 其他错误 |

服务一次只签一个请求，其余请求排队。在 Apple Silicon 上实测，一次签名约 35 毫秒；换了设备身份后的第一次约 100 毫秒，因为要先把新的 `device_id` 和 `iid` 设置进 SDK。

## 运维

| 操作 | 命令 |
|---|---|
| 看日志 | `docker compose logs -f openhongguo-signer` |
| 看 caddy 日志 | `docker compose logs -f caddy` |
| 重启 | `docker compose restart openhongguo-signer` |
| 升级到新版本 | 修改 `.env` 里的 `SIGNER_IMAGE`，然后运行 `docker compose pull && docker compose up -d` |
| 从源码重新部署 | 重新 rsync，然后运行 `docker compose up -d --build` |
| 修改配置 | 修改 `.env`，然后运行 `docker compose up -d` |

服务连续 3 次签不出来（返回空结果或 `X-Neptune -8`），进程会主动退出，由 `restart: unless-stopped` 重新拉起。可以用环境变量 `SIGNER_MAX_FAILURES` 修改次数。SDK 拒签的请求（422）不计入失败次数。unicorn 偶发的 JIT 崩溃也会让 JVM 退出，同样会被自动拉起。

## 环境变量

`.env` 中的变量：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `SIGNER_IMAGE` | `openhongguo-signer:latest` | 要运行的镜像。留空时使用本地构建的镜像 |
| `SIGNER_DOMAIN` | `localhost` | caddy 使用的域名。`localhost` 只签发本地自签证书，用于试用 |
| `SIGNER_BIND` | `127.0.0.1` | 签名服务端口在宿主机上的映射地址 |
| `SIGNER_PUBLISH_PORT` | `19830` | 宿主机上的端口 |
| `SIGNER_TOKEN` | 空 | 设置后 `/feature-hash` 要求 Bearer token；为空时不鉴权 |
| `SIGNER_MAX_FAILURES` | `3` | 连续签名失败多少次后退出重启 |

不用 Docker、直接运行 `build/install/openhongguo-signer/bin/openhongguo-signer` 时，还可以用以下变量：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `SIGNER_HOST` | `127.0.0.1`（镜像内为 `0.0.0.0`） | 监听地址 |
| `SIGNER_PORT` | `19830` | 监听端口 |
