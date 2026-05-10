# AGENTS.md — Project Conventions for new-api

> 注意：本文件是 AI 助手在本仓库内协作开发的最高优先级项目规则。执行任务前，应先阅读并遵守本文件。

## 1. 协作总则

### 语言策略

- 默认使用简体中文进行交流、说明、提交备注和问题反馈。
- 代码实现中的变量名、函数名、类型名、配置键、协议字段、日志原文保持英文。
- 面向用户的文案应优先与现有界面语言风格保持一致；如果现有页面已接入 i18n，新增文案必须同步接入 i18n。

### 修改策略

- 默认直接按当前项目的最优方案修改，不为了兼容旧写法额外保留冗余分支。
- 若需求明确要求兼容旧行为、旧接口、旧数据，再单独设计兼容方案。
- 不要把无关重构、顺手格式化、批量命名调整混入当前任务。

### 方案确认条件

满足以下任一条件时，开始写代码前应先给出简短方案，待用户确认后再实施：

- 跨前后端或跨两个及以上核心模块。
- 涉及数据模型、接口结构、鉴权/计费/权限策略调整。
- 预估修改文件数大于等于 3，或明显存在回归风险。
- 涉及生产部署流程、对外端口、容器编排、反向代理边界调整。

### 禁止事项

- 禁止修改运行时数据、临时数据或备份产物，除非用户明确要求。
- 禁止把密钥、令牌、数据库连接串、生产环境凭据写入仓库。
- 禁止未经用户明确授权操作非当前目标环境，尤其是生产环境。

## 2. Tech Stack

- **Backend**: Go 1.22+, Gin web framework, GORM v2 ORM
- **Frontend**: React 19, TypeScript, Rsbuild, Radix UI, Tailwind CSS
- **Databases**: SQLite, MySQL, PostgreSQL (all three must be supported)
- **Cache**: Redis (go-redis) + in-memory cache
- **Auth**: JWT, WebAuthn/Passkeys, OAuth (GitHub, Discord, OIDC, etc.)
- **Frontend package manager**: Bun (preferred over npm/yarn/pnpm)

## 3. Architecture

Layered architecture: Router -> Controller -> Service -> Model

```
router/        — HTTP routing (API, relay, dashboard, web)
controller/    — Request handlers
service/       — Business logic
model/         — Data models and DB access (GORM)
relay/         — AI API relay/proxy with provider adapters
  relay/channel/ — Provider-specific adapters (openai/, claude/, gemini/, aws/, etc.)
middleware/    — Auth, rate limiting, CORS, logging, distribution
setting/       — Configuration management (ratio, model, operation, system, performance)
common/        — Shared utilities (JSON, crypto, Redis, env, rate-limit, etc.)
dto/           — Data transfer objects (request/response structs)
constant/      — Constants (API types, channel types, context keys)
types/         — Type definitions (relay formats, file sources, errors)
i18n/          — Backend internationalization (go-i18n, en/zh)
oauth/         — OAuth provider implementations
pkg/           — Internal packages (cachex, ionet)
web/             — Frontend themes container
  web/default/   — Default frontend (React 19, Rsbuild, Radix UI, Tailwind)
  web/classic/   — Classic frontend (React 18, Vite, Semi Design)
  web/default/src/i18n/ — Frontend internationalization (i18next, zh/en/fr/ru/ja/vi)
```

## 4. Internationalization

### Backend (`i18n/`)

- Library: `nicksnyder/go-i18n/v2`
- Languages: en, zh

### Frontend (`web/default/src/i18n/`)

- Library: `i18next` + `react-i18next` + `i18next-browser-languagedetector`
- Languages: en (base), zh (fallback), fr, ru, ja, vi
- Translation files: `web/default/src/i18n/locales/{lang}.json` — flat JSON, keys are English source strings
- Usage: `useTranslation()` hook, call `t('English key')` in components
- CLI tools: `bun run i18n:sync` (from `web/default/`)

## 5. 开发与验证流程

### 基本流程

1. 先确认需求范围和影响面。
2. 必要时先给方案，再开始改代码。
3. 按当前技术栈和本文件规则实现。
4. 修改完成后执行最小必要验证。
5. 若本次任务包含代码或文档修改，默认需要整理提交并推送，除非用户明确要求不要提交或不要 push。

### 后端最小验证

涉及 Go 后端代码时，至少执行与改动直接相关的以下检查：

- `gofmt` 或等效格式化检查。
- `go build`，至少保证受影响入口可编译。
- 若存在对应测试，优先运行受影响包的 `go test`。

若改动影响主程序入口、HTTP 接口、配置解析、数据库访问、计费、认证或 relay 逻辑，应优先执行：

- `go test ./...` 或最小等价范围测试。

### 前端最小验证

涉及 `web/default/` 时，优先执行：

- `cd web/default && bun install`
- `cd web/default && bun run build`

涉及 `web/classic/` 时，优先执行：

- `cd web/classic && bun install`
- `cd web/classic && bun run build`

如仅改动局部前端文件，也至少保证对应主题可构建通过。

### 部署与运行验证

如果任务包含部署、重启、端口切换、Compose 变更、Nginx 反代变更，完成后至少验证：

- 目标端口进程或容器已按预期启动。
- `curl -fsS http://127.0.0.1:${PORT}/health` 返回成功。
- 新启动时间、容器状态或监听端口与本次部署匹配。

## 6. 生产部署边界与环境约束

### 10011 环境约定

- 当前项目内隔离部署默认端口为 `10011`。
- `10011` 对应 `docker-compose.prod.yml` + `scripts/deploy_prod_10011.sh`。
- 涉及 `10011` 的重建、重启、部署、健康检查，应优先复用现有脚本与 Compose 配置，不要绕开既有部署入口另起一套流程。

### 生产环境边界

- 未经用户在当前任务中明确授权，不要执行会影响其他生产站点或宿主机全局 Nginx 的操作。
- 不要修改宿主机全局 Nginx 配置、`systemctl reload nginx`、或直接接管 `80/443` 的全局行为，除非用户明确要求。
- 项目内 Nginx 与容器边界应遵守 `docs/installation/isolated-nginx-prod.md`。

### 环境变量与密钥

- 用户可能已在 shell 初始化脚本中预先配置部署、构建、联调所需环境变量；执行前应先检查变量是否存在，而不是默认未配置。
- 允许检查变量是否存在，禁止回显密钥明文、完整 DSN、Token 或私钥内容。
- `prod.env` 一类文件可能包含敏感配置，禁止提交到仓库，除非用户明确要求且已完成脱敏。

### 出站网络与代理

- 对于 Docker 拉取镜像、Git 访问海外仓库、访问海外网站/API/文档等出站网络操作，默认必须走代理，禁止直连。
- 统一代理地址为 `http://127.0.0.1:7890`。
- Shell 中执行海外网络命令时，优先使用 `with_proxy <command>`。
- Docker 和 Git 相关操作优先复用系统已配置代理；需要手动指定时，统一设置 `HTTP_PROXY`、`HTTPS_PROXY`、`ALL_PROXY` 及其小写变量。

## 7. Rules

### Rule 1: JSON Package — Use `common/json.go`

All JSON marshal/unmarshal operations MUST use the wrapper functions in `common/json.go`:

- `common.Marshal(v any) ([]byte, error)`
- `common.Unmarshal(data []byte, v any) error`
- `common.UnmarshalJsonStr(data string, v any) error`
- `common.DecodeJson(reader io.Reader, v any) error`
- `common.GetJsonType(data json.RawMessage) string`

Do NOT directly import or call `encoding/json` in business code. These wrappers exist for consistency and future extensibility (e.g., swapping to a faster JSON library).

Note: `json.RawMessage`, `json.Number`, and other type definitions from `encoding/json` may still be referenced as types, but actual marshal/unmarshal calls must go through `common.*`.

### Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6

All database code MUST be fully compatible with all three databases simultaneously.

**Use GORM abstractions:**

- Prefer GORM methods (`Create`, `Find`, `Where`, `Updates`, etc.) over raw SQL.
- Let GORM handle primary key generation — do not use `AUTO_INCREMENT` or `SERIAL` directly.

**When raw SQL is unavoidable:**

- Column quoting differs: PostgreSQL uses `"column"`, MySQL/SQLite uses `` `column` ``.
- Use `commonGroupCol`, `commonKeyCol` variables from `model/main.go` for reserved-word columns like `group` and `key`.
- Boolean values differ: PostgreSQL uses `true`/`false`, MySQL/SQLite uses `1`/`0`. Use `commonTrueVal`/`commonFalseVal`.
- Use `common.UsingPostgreSQL`, `common.UsingSQLite`, `common.UsingMySQL` flags to branch DB-specific logic.

**Forbidden without cross-DB fallback:**

- MySQL-only functions (e.g., `GROUP_CONCAT` without PostgreSQL `STRING_AGG` equivalent)
- PostgreSQL-only operators (e.g., `@>`, `?`, `JSONB` operators)
- `ALTER COLUMN` in SQLite (unsupported — use column-add workaround)
- Database-specific column types without fallback — use `TEXT` instead of `JSONB` for JSON storage

**Migrations:**

- Ensure all migrations work on all three databases.
- For SQLite, use `ALTER TABLE ... ADD COLUMN` instead of `ALTER COLUMN` (see `model/main.go` for patterns).

### Rule 3: Frontend — Prefer Bun

Use `bun` as the preferred package manager and script runner for the frontend:

- `bun install` for dependency installation
- `bun run dev` for development server
- `bun run build` for production build
- `bun run i18n:*` for i18n tooling

### Rule 4: New Channel StreamOptions Support

When implementing a new channel:

- Confirm whether the provider supports `StreamOptions`.
- If supported, add the channel to `streamSupportedChannels`.

### Rule 5: Frontend i18n Discipline

When adding or changing user-facing text in `web/default/`:

- Prefer existing translation keys and conventions.
- If introducing new text, update the locale files under `web/default/src/i18n/locales/`.
- Do not leave newly added UI text hard-coded in only one language when the surrounding module already uses i18n.

### Rule 6: Upstream Relay Request DTOs — Preserve Explicit Zero Values

For request structs that are parsed from client JSON and then re-marshaled to upstream providers (especially relay/convert paths):

- Optional scalar fields MUST use pointer types with `omitempty` (e.g. `*int`, `*uint`, `*float64`, `*bool`), not non-pointer scalars.
- Semantics MUST be:
  - field absent in client JSON => `nil` => omitted on marshal;
  - field explicitly set to zero/false => non-`nil` pointer => must still be sent upstream.
- Avoid using non-pointer scalars with `omitempty` for optional request parameters, because zero values (`0`, `0.0`, `false`) will be silently dropped during marshal.

### Rule 7: Billing Expression System — Read `pkg/billingexpr/expr.md`

When working on tiered/dynamic billing (expression-based pricing), you MUST read `pkg/billingexpr/expr.md` first. It documents the design philosophy, expression language (variables, functions, examples), full system architecture (editor → storage → pre-consume → settlement → log display), token normalization rules (`p`/`c` auto-exclusion), quota conversion, and expression versioning. All code changes to the billing expression system must follow the patterns described in that document.

## 8. 提交规范

- 提交信息格式建议使用：`feat/fix/chore/docs: <description>`。
- 只提交本次任务直接相关文件，避免把无关改动混入同一提交。
- 默认情况下，只要发生代码或文档修改，任务结束时应执行 `git commit` 和 `git push`。
- 唯一例外是用户明确要求“不提交”或“不要 push”。
- 如果构建、测试、验证门禁失败，不允许强行提交和推送；应先修复或向用户说明阻塞点。
- 完成提交后，回复中应说明分支名、Commit Hash 和 push 结果。

## 9. 生产服务器

- newapi 项目的生产服务器是 `38.76.215.133`，用户名是 `root`，可以直接 ssh 访问。必须有明确授权才可登录生产服务器执行操作。
- 如果要更新生产服务器，应先在当前服务器完成编译并打包好安装包，再登录生产服务器执行更新；不要在生产服务器上执行编译构建操作。
