# gin-api-scaffold

Features:
- Gin HTTP API
- PostgreSQL / pgx
- Layered architecture: handler / service / repository
- Config loader
- Request ID
- Logger
- CORS
- Rate limit
- JWT middleware
- Cursor pagination middleware
- Health check
- Standard response envelope

## Cursor pagination middleware

Use `middleware.CursorPagination` on list routes that accept `limit` and `cursor` query parameters.

```go
users.GET("", middleware.CursorPagination(middleware.CursorPaginationConfig{
	DefaultLimit: service.DefaultUsersListLimit,
	MaxLimit:     service.MaxUsersListLimit,
}), usersHandler.List)
```

Read the parsed pagination parameters from the Gin context in the handler:

```go
pagination, _ := middleware.CurrentCursorPagination(c)

filter := types.ListUsersFilter{
	Search: strings.TrimSpace(c.Query("search")),
	Limit:  pagination.Limit,
	Cursor: pagination.Cursor,
}
```

The middleware handles the common request parsing:

- `limit` is optional.
- Empty or non-positive `limit` uses `DefaultLimit`.
- `limit` above `MaxLimit` is capped.
- Non-integer `limit` returns `400 invalid_query`.
- `cursor` is trimmed and passed through as an opaque string.

Each service still owns cursor encoding and decoding because each list endpoint can have different sort keys. The example users list encodes the last returned user's `created_at` and `id`, then queries the next page with `(created_at, id) > cursor`.

First page:

```http
GET /api/v1/users?search=ada&limit=10
```

Next page:

```http
GET /api/v1/users?search=ada&limit=10&cursor=<next_cursor>
```

Example response:

```json
{
  "success": true,
  "data": {
    "items": [],
    "limit": 10,
    "next_cursor": "opaque-cursor"
  }
}
```

When `next_cursor` is omitted, there are no more pages.
