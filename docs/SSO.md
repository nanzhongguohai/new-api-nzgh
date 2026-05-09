# SSO 配置指南 — new-api（OIDC Provider 端）

new-api 可作为 OIDC Provider（OpenID Connect 认证服务端），让 LobeChat 通过 OIDC 协议完成单点登录。

## 架构

```
用户浏览器 → LobeChat（RP / OIDC Client）
                ↓ OIDC Authorization Code Flow
           new-api（OP / OIDC Provider）
```

- **new-api**：OIDC Provider (OP)，负责用户认证、签发 ID Token。
- **LobeChat**：OIDC Relying Party (RP)，通过 Better-Auth 的 new-api provider 接入。

## 前置条件

1. new-api 服务正常运行（开发环境 `:3000`，生产环境按实际域名）。
2. new-api 数据库可访问（SQLite / MySQL / PostgreSQL 均可）。
3. 已完成 new-api 自身的管理员账号创建，OIDC 登录基于 new-api 的用户体系。

## 步骤一：设置环境变量

OIDC Provider 需要知道前端（LobeChat）的访问地址，以便生成正确的回调 URL。

### 开发环境

`docker-compose.dev.yml` 中已预置：

```yaml
- FRONTEND_BASE_URL=http://localhost:3001
- NODE_TYPE=slave
```

### 生产环境

在 `docker-compose.yml` 或 `prod.env` 中设置：

```bash
FRONTEND_BASE_URL=https://your-lobechat-domain.com
NODE_TYPE=master
```

| 变量 | 说明 |
|------|------|
| `FRONTEND_BASE_URL` | LobeChat 前端的公网访问地址，用于生成 OAuth 回调 URL。必填。 |
| `NODE_TYPE` | 节点类型。单节点部署设为 `master`，多节点 worker 设为 `slave`。 |

## 步骤二：启用 OIDC Provider

1. 登录 new-api 管理后台。
2. 进入 **系统设置 → 通用设置**。
3. 找到 `oidc_provider` 配置项，将 `enabled` 设为 `true`。
4. 保存配置。

或者通过 API 直接设置：

```bash
curl -X PUT http://localhost:3000/api/option \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"key": "oidc_provider", "value": "{\"enabled\":true}"}'
```

启动时 new-api 会自动生成 RSA 密钥对（存储在 `oidc_provider_private_key` 选项中）用于签发 JWT。

## 步骤三：注册 OIDC 客户端

在 new-api 数据库中注册 LobeChat 为受信任的 OIDC 客户端：

```sql
INSERT INTO oidc_clients (client_id, client_secret, redirect_uri, name, enabled)
VALUES (
  'lobehub',
  '<生成一个随机密钥>',
  'https://your-lobechat-domain.com/api/auth/callback/new-api',
  'LobeChat',
  true
);
```

**开发环境示例：**

```sql
INSERT INTO oidc_clients (client_id, client_secret, redirect_uri, name, enabled)
VALUES (
  'lobehub',
  'dev-secret',
  'http://localhost:3210/api/auth/callback/new-api',
  'LobeChat',
  true
);
```

| 字段 | 说明 |
|------|------|
| `client_id` | 客户端标识，需与 LobeChat 侧 `AUTH_NEWAPI_ID` 一致，建议填 `lobehub` |
| `client_secret` | 客户端密钥，需与 LobeChat 侧 `AUTH_NEWAPI_SECRET` 一致，生产环境务必使用随机强密码 |
| `redirect_uri` | 回调地址，格式为 `<LOBECHAT_URL>/api/auth/callback/new-api` |
| `enabled` | 设为 `true` 启用 |

## 步骤四：验证

1. 访问 new-api 的 OIDC Discovery 端点确认配置生效：

```bash
curl http://localhost:3000/.well-known/openid-configuration
```

2. 应返回包含 `issuer`、`authorization_endpoint`、`token_endpoint`、`jwks_uri` 等字段的 JSON。

3. 在 LobeChat 侧完成 SSO 配置后（参见 LobeChat `docs/SSO.md`），使用浏览器访问 LobeChat 登录页，应出现 "使用 new-api 登录" 的 SSO 按钮。

## 故障排查

| 现象 | 可能原因 | 解决方法 |
|------|----------|----------|
| Discovery 端点 404 | 未启用 OIDC Provider | 按步骤二启用 |
| 回调地址不匹配 | `FRONTEND_BASE_URL` 未正确设置 | 检查环境变量，确保与 LobeChat 实际访问地址一致 |
| 客户端认证失败 | `oidc_clients` 表中无对应记录或密码不一致 | 核对 `client_id` 和 `client_secret` |
| 签名验证失败 | RSA 密钥未生成 | 重启 new-api 触发 `InitOIDCProviderKeys` |
