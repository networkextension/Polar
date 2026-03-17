# Auth 模块总结

这个模块提供基于 Cookie + Session 的认证流程，服务于 `Gin` 应用的注册、登录、登出和用户信息读取。核心状态存储在 PostgreSQL 中，所有会话与用户信息由数据库持久化管理。

## 关键组件

- User 模型：包含 `id`、`username`、`email`、`password_hash`、`created_at` 字段，存储于 `users` 表。
- Session 模型：包含 `id`、`user_id`、`username`、`expires_at` 字段，存储于 `sessions` 表。
- 会话 Cookie：使用 `session_id` 作为 cookie 名称，默认有效期 24 小时。

## 数据库结构

- `users` 表：邮箱唯一约束，保存用户基础信息与密码哈希。
- `sessions` 表：保存会话与过期时间，关联用户 `id`，支持登出与过期清理。

## 认证流程

- 注册 `/api/register`：
  - 参数：`username`、`email`、`password`
  - 校验邮箱唯一性，密码使用 bcrypt 哈希
  - 创建用户与 session，写入 cookie
- 登录 `/api/login`：
  - 参数：`email`、`password`
  - 校验用户与密码，创建 session，写入 cookie
- 登出 `/api/logout`：
  - 删除 session 并清理 cookie
- 当前用户 `/api/me`：
  - 需要认证中间件，返回当前用户信息

## 中间件

- AuthMiddleware：
  - 验证 cookie 中的 session
  - 失效或过期则清理 cookie，并重定向到登录页
- GuestMiddleware：
  - 已登录用户访问登录/注册页时重定向到控制台

## Session 生命周期

- Session 创建时写入 `sessions` 表，并设置过期时间
- 定时协程每小时清理过期 session

## 运行配置

- 环境变量 `POSTGRES_DSN` 指定数据库连接
- 默认值：`postgres://postgres:postgres@localhost:5432/gin_auth?sslmode=disable`
