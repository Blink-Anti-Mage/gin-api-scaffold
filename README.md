# gin-api-scaffold

一个基于 Gin + PostgreSQL/pgx 的 API scaffold。当前示例 app 采用清晰分层、统一响应 envelope、统一错误码、请求 ID、业务校验和 cursor 分页约定。

## 功能概览

- Gin HTTP API
- PostgreSQL / pgx
- 分层架构：handler / service / repository / types
- Config loader
- Request ID
- Logger
- CORS
- Body size limit
- Rate limit
- JWT middleware based on `golang-jwt/jwt/v5`
- Cursor pagination middleware
- Health check
- Standard response envelope

## 目录规矩

- `app/example/handler`：只处理 HTTP 输入输出，包括 path/query/body 读取、调用 service、返回 `httpx` 响应。
- `app/example/service`：放业务规则、输入 normalize、业务校验、分页 cursor 编解码。
- `app/example/repository`：只处理数据库访问和数据库错误映射，不处理 HTTP。
- `app/example/types`：放 request/input/filter/response domain types。
- `app/example/sql`：放 schema 初始化 SQL。
- `internal/httpx`：统一响应 envelope、JSON binding、字段校验错误输出。
- `internal/apperr`：统一业务错误结构和错误码。
- `internal/middleware`：通用中间件，例如 request id、logger、JWT、rate limit、cursor pagination。

新增模块时优先复制这个分层方式，不要把 SQL、业务校验、HTTP response 混在同一层。

## 路由规矩

路由统一在 `app/example/route.go` 注册。

全局 middleware 顺序：

```go
router.Use(middleware.RequestID())
router.Use(middleware.Logger(deps.Logger))
router.Use(gin.Recovery())
router.Use(middleware.CORS(deps.Config.CORS))
router.Use(middleware.BodySizeLimit(deps.Config.HTTP.MaxBodyBytes))
```

`/api/v1` 组内统一挂：

```go
v1.Use(middleware.RateLimit(deps.Config.RateLimit))
if deps.Config.Auth.Enabled {
	v1.Use(middleware.JWT(deps.Config.Auth))
}
```

List 接口需要 cursor 分页时，在具体 GET 路由上挂分页中间件：

```go
users.GET("", middleware.CursorPagination(middleware.CursorPaginationConfig{
	DefaultLimit: service.DefaultUsersListLimit,
	MaxLimit:     service.MaxUsersListLimit,
}), usersHandler.List)
```

## JWT 规矩

JWT middleware 使用 `github.com/golang-jwt/jwt/v5`，不要手写 base64、claims 解析或 HMAC 验签。

当前约定：

- token 从 `Authorization: Bearer <token>` 读取。
- 只接受 `HS256`。
- 签名 secret 来自 `config.Auth.Secret`。
- 必须包含 `sub`。
- 必须包含 `exp`。
- 会校验 `exp`、`nbf`、`iat`、`iss`、`aud`。
- `iss` 和 `aud` 只有在配置中非空时才要求匹配。
- `clock_skew` 通过 jwt parser leeway 处理。
- 解析后的 claims 存入 Gin context：
  - `middleware.CurrentJWTClaims(c)`
  - `middleware.CurrentSubject(c)`

认证失败统一返回：

```json
{
  "success": false,
  "error": {
    "code": "invalid_token",
    "message": "invalid or expired token"
  },
  "request_id": "..."
}
```

没有 Bearer token 时返回 `missing_token`。

## 响应规矩

所有 JSON 响应都走 `internal/httpx`，不要在业务 handler 里直接 `c.JSON`。

成功响应 envelope：

```json
{
  "success": true,
  "data": {},
  "request_id": "..."
}
```

错误响应 envelope：

```json
{
  "success": false,
  "error": {
    "code": "invalid_email",
    "message": "invalid email",
    "details": []
  },
  "request_id": "..."
}
```

约定：

- `httpx.OK(c, data)` 返回 `200`。
- `httpx.Created(c, data)` 返回 `201`。
- `httpx.NoContent(c)` 返回 `204`，无 body。
- `httpx.Error(c, err)` 统一转换 `apperr.Error`。
- 成功和失败响应都必须带 `request_id`。
- 字段级错误放在 `error.details`，结构为 `{ "field": "...", "reason": "..." }`。

## 错误规矩

业务错误使用 `internal/apperr`：

```go
return types.User{}, apperr.BadRequest("invalid_email", "invalid email")
```

需要字段详情时：

```go
return apperr.BadRequestWithDetails("validation_failed", "validation failed", details)
```

常用错误：

- `400 invalid_query`：query 参数格式错误。
- `400 invalid_request`：JSON body 格式错误。
- `400 validation_failed`：binding/validator 字段校验失败，带 `details`。
- `400 invalid_name`：用户名称业务校验失败。
- `400 invalid_email`：用户邮箱业务校验失败。
- `400 invalid_cursor`：cursor 无法解析。
- `404 not_found`：资源不存在。
- `409 user_email_exists`：用户邮箱唯一性冲突。
- `413 payload_too_large`：请求体过大。
- `429 rate_limited`：触发限流。
- `500 internal_error`：未识别错误。

Repository 负责把数据库错误映射成业务错误，例如 `pgx.ErrNoRows` 映射为 `apperr.NotFound("user")`，唯一索引冲突映射为 `apperr.Conflict(...)`。

## 请求校验规矩

`httpx.BindJSON` 负责 JSON 结构和 Gin validator 错误：

- 请求体过大返回 `payload_too_large`。
- validator 错误返回 `validation_failed`，并输出字段级 `details`。
- JSON 格式错误返回 `invalid_request`。

Service 负责业务校验和 normalize。不要只依赖数据库约束暴露业务错误。

用户模块当前规则：

- `name` 会 `strings.TrimSpace`。
- `name` 不能为空。
- `name` 最长 100 个字符，按 rune 计数。
- `email` 会 `strings.TrimSpace` 并转小写。
- `email` 不能为空。
- `email` 最长 255 字节。
- `email` 使用 `net/mail` 校验格式。
- `Create` 和 `Update` 共用同一套用户输入校验。

## Cursor 分页规矩

List 接口使用 cursor 模式，不使用 `offset`。

请求参数：

- `limit`：可选；空值或 `<=0` 使用默认值；超过 `MaxLimit` 会截断；非整数返回 `400 invalid_query`。
- `cursor`：可选；作为 opaque string 传给 service。

Handler 从 Gin context 读取中间件解析后的分页参数：

```go
pagination, _ := middleware.CurrentCursorPagination(c)

filter := types.ListUsersFilter{
	Search: strings.TrimSpace(c.Query("search")),
	Limit:  pagination.Limit,
	Cursor: pagination.Cursor,
}
```

Service 负责具体业务 cursor 的编解码，因为每个列表的排序键可能不同。

用户列表当前排序键：

```sql
ORDER BY created_at ASC, id ASC
```

用户列表 cursor 内容：

- 上一页最后一条的 `created_at`
- 上一页最后一条的 `id`

Repository 查询下一页时使用 keyset 条件：

```sql
(created_at, id) > ($cursor_created_at, $cursor_id)
```

Service 查询时多取一条：

- repository limit = `limit + 1`
- 如果多取到第 `limit + 1` 条，说明还有下一页
- 返回前截断到 `limit`
- 用当前页最后一条生成 `next_cursor`

第一页：

```http
GET /api/v1/users?search=ada&limit=10
```

下一页：

```http
GET /api/v1/users?search=ada&limit=10&cursor=<next_cursor>
```

响应示例：

```json
{
  "success": true,
  "data": {
    "items": [],
    "limit": 10,
    "next_cursor": "opaque-cursor"
  },
  "request_id": "..."
}
```

当 `next_cursor` 省略时，表示没有下一页。

## 用户模块 API 规矩

当前用户模块提供完整 CRUD + List + Stats：

- `POST /api/v1/users`：创建用户，成功返回 `201`。
- `GET /api/v1/users`：cursor 分页列表。
- `GET /api/v1/users/:id`：获取用户详情。
- `PUT /api/v1/users/:id`：完整更新用户，成功返回更新后的 user。
- `DELETE /api/v1/users/:id`：删除用户，成功返回 `204`。
- `GET /api/v1/users/stats`：用户统计。

行为约定：

- 不存在的用户返回 `404 not_found`。
- 邮箱重复返回 `409 user_email_exists`。
- Create/Update 入库前先做 service 业务校验。
- Delete 成功不返回 body。

## Repository 规矩

- 所有数据库调用都带 `context.Context`。
- PostgreSQL 访问使用 `pgx` / `pgxpool`。
- `repository` 不直接返回 HTTP 响应。
- `repository` 可以返回 `apperr`，用于数据库错误到业务错误的映射。
- 查询结果扫描后必须检查 `rows.Err()`。
- `List` 查询必须有稳定排序；cursor 分页排序字段要和 cursor 内容一致。

## 测试和提交前检查

改 Go 代码后必须格式化：

```bash
gofmt -w <changed-go-files>
```

提交前至少跑：

```bash
go test ./...
go vet ./...
```

新增规则时要补对应测试：

- service 业务校验补 service test。
- middleware 行为补 middleware test。
- response/binding 结构补 httpx test。
- repository SQL 行为如果影响错误映射或分页条件，要补对应测试或至少通过集成验证。
