# 项目内 Nginx 隔离部署

这套部署结构的目标是把 `new-api` 的请求缓冲、反代超时、热更新和临时目录全部收回到项目自己的容器里，避免继续占用或修改宿主机 `nginx.service` 的全局状态。

## 结构

- 宿主机 Nginx：只监听 `80/443`，只负责 TLS 和域名分发
- 项目内 Nginx：运行在 `docker-compose.prod.yml` 中，对宿主机仅暴露 `127.0.0.1:${PORT}`
- `new-api` 应用：仅在 Docker 网络内监听 `3000`

请求链路：

`client -> 宿主机 Nginx -> 127.0.0.1:${PORT} -> 项目内 Nginx -> new-api:3000`

## 为什么这样隔离

- 宿主机 `/var/lib/nginx` 不再承接本项目的大请求体缓冲
- 项目自己的 Nginx 重启、替换配置、临时目录异常，不会影响宿主机其他站点
- 宿主机只保留一条稳定反代，项目内部的 Nginx 配置归项目仓库管理

## 使用方法

1. 复制环境变量模板：

```bash
cp prod.env.example prod.env
```

2. 按实际环境修改 `prod.env`。

如果数据库或 Redis 仍然运行在宿主机，`prod.env` 里不要写 `localhost` 或 `127.0.0.1`，而是改成 `host.docker.internal`。例如：

```env
SQL_DSN=postgres://new_api:new_api_password@host.docker.internal:5432/new_api?sslmode=disable
LOG_SQL_DSN=postgres://new_api:new_api_password@host.docker.internal:5432/new_api?sslmode=disable
REDIS_CONN_STRING=redis://host.docker.internal:6379
```

3. 执行部署脚本：

```bash
./scripts/deploy_prod_10011.sh
```

该脚本会先在宿主机构建前端和 Go 二进制，然后再用 `Dockerfile.prod` 封装运行时镜像。这样可以避免 Docker 构建阶段再去访问前端依赖、Go 模块和系统包仓库。

4. 宿主机 Nginx 继续反代到 `127.0.0.1:${PORT}` 即可，不需要再把请求直接打到 Go 进程。

## 宿主机 Nginx 的边界

宿主机 Nginx 现在只需要保留这种最小职责：

- 证书与 `80/443`
- 基于域名转发到 `127.0.0.1:${PORT}`
- 基础访问日志和全局限流

不要再让项目脚本去修改：

- `/etc/nginx/sites-enabled/*`
- `/var/lib/nginx/*`
- `systemctl reload nginx`

## 项目内 Nginx 的特性

- 独立临时目录：`/tmp/nginx/*`
- `proxy_request_buffering off`
- `proxy_buffering off`
- 保留 `Upgrade` / `Connection` 头，兼容 SSE 和长连接场景
- 信任来自宿主机 `127.0.0.1` 的转发头，继续向应用透传真实客户端 IP

## 宿主机前置条件

- 已安装 Docker Compose v2
- 已安装 `bun`
- 已安装 Go（建议版本不低于项目要求）
