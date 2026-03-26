# Passkey 工作文档

本文档记录 Gin Auth Demo 中 Passkey（WebAuthn）登录的实现、配置与常见问题排查。

## 目标
- 支持用户绑定 Passkey（注册凭据）。
- 支持用户使用 Passkey 登录。
- 兼容本地开发时 `localhost` / `127.0.0.1` / `::1` 的访问差异。
- 支持 UI 静态站（`localhost:3000`）跨域访问 API（`localhost:8080`）。

## 系统结构
- 后端：Gin + WebAuthn（`github.com/go-webauthn/webauthn`）
- 会话：Postgres 中存储 Session
- Passkey 凭据：Postgres 持久化 `webauthn_credentials`
- UI：
  - Gin 模板页：`/login`、`/dashboard`
  - 静态站：`ui/public/*.html`（`localhost:3000`）

## 主要接口
### 绑定 Passkey
- `POST /api/passkey/register/begin`（需要已登录）
  - 返回 `publicKey` 与 `session_id`
- `POST /api/passkey/register/finish`（需要已登录）
  - Header: `X-Passkey-Session: <session_id>`
  - Body: 浏览器 `navigator.credentials.create()` 的结果
  - 返回当前账户的 Passkey 列表、数量与是否已绑定 Passkey
- `GET /api/passkeys`（需要已登录）
  - 返回当前账户已绑定的 Passkey 列表与数量
- `DELETE /api/passkeys/:credentialId`（需要已登录）
  - 删除当前账户名下的某个 Passkey

### Passkey 登录
- `POST /api/passkey/login/begin`（访客）
  - Body: `{ "email": "user@example.com" }`
- `POST /api/passkey/login/finish`（访客）
  - Header: `X-Passkey-Session: <session_id>`
  - Body: 浏览器 `navigator.credentials.get()` 的结果

## 前端流程
### 注册（绑定）
1. 点击 “绑定 Passkey”
2. `register/begin` 返回 `publicKey`
3. 浏览器 `navigator.credentials.create({ publicKey })`
4. `register/finish` 校验并存储凭据
5. 个人中心“账户安全”即时刷新为“已绑定 N 个 Passkey”

### 已绑定列表与删除
1. 个人中心加载时调用 `GET /api/passkeys`
2. 前端展示已绑定状态、数量、创建时间和更新时间
3. 用户点击“删除”后调用 `DELETE /api/passkeys/:credentialId`
4. 删除成功后前端使用最新列表重新渲染

### 登录
1. 输入邮箱
2. 点击 “使用 Passkey 登录”
3. `login/begin` 返回 `publicKey`
4. 浏览器 `navigator.credentials.get({ publicKey })`
5. `login/finish` 校验并创建 Session

## 配置说明
### 环境变量
- `PASSKEY_ORIGIN`
  - 例：`http://localhost:8080`
  - 必须与浏览器地址栏中的 Origin 完全一致（协议 + 主机 + 端口）
- `PASSKEY_RP_ID`
  - 例：`localhost`
  - 必须与 `PASSKEY_ORIGIN` 的主机一致，且不带端口
- `PASSKEY_RP_NAME`
  - 例：`Gin Auth Demo`

### 自动对齐机制
当 `PASSKEY_ORIGIN` 与 `PASSKEY_RP_ID` 保持默认值时，系统会优先使用请求的 `Origin` 计算 `rpId` 和 `origin`，避免 `localhost` / `127.0.0.1` 不匹配导致校验失败。

## 数据存储
### 表结构
新增表：
- `webauthn_credentials`
  - `credential_id`：主键（base64url）
  - `user_id`：关联用户
  - `credential_json`：完整凭据 JSON
  - `created_at` / `updated_at`

### Session
Passkey 的中间会话（challenge 等）保存在内存 `passkeySessions` 中，默认 TTL 为 5 分钟。

## CORS 与跨域
静态站（`localhost:3000`）访问 API（`localhost:8080`）时：
- 需要后端允许跨域并允许携带 cookie
- 后端已允许 `http://localhost:3000`

## 常见问题
### 1. “网络错误，请重试”
通常是浏览器拦截跨域请求：
- 确认 API 启动地址
- 静态站需走 CORS 允许的 Origin
- 检查浏览器 Network 是否有 CORS 报错

### 2. “Passkey 校验失败”
通常是 RP 配置不匹配：
- 确认 `PASSKEY_ORIGIN` 与当前访问地址完全一致
- `PASSKEY_RP_ID` 必须是同一主机（不含端口）

### 3. Safari 无法使用
Safari 需要安全上下文：
- `https://` 或 `http://localhost`

## API 样例
### 绑定 Passkey（注册）
#### begin
请求：
```bash
curl -i http://localhost:8080/api/passkey/register/begin \
  -H "Content-Type: application/json" \
  -H "Cookie: session_id=YOUR_SESSION_ID"
```
响应（示例）：
```json
{
  "publicKey": {
    "rp": { "name": "Gin Auth Demo", "id": "localhost" },
    "user": {
      "name": "user@example.com",
      "displayName": "user",
      "id": "c2Vzc2lvbi11c2VyLWlk"
    },
    "challenge": "r1qzq8Hn3U5YQwYzO3W7XvR1U0n8uI",
    "pubKeyCredParams": [
      { "type": "public-key", "alg": -7 },
      { "type": "public-key", "alg": -257 }
    ],
    "timeout": 300000,
    "authenticatorSelection": {}
  },
  "session_id": "PASSKEY_SESSION_ID"
}
```

#### finish
请求：
```bash
curl -i http://localhost:8080/api/passkey/register/finish \
  -H "Content-Type: application/json" \
  -H "X-Passkey-Session: PASSKEY_SESSION_ID" \
  -H "Cookie: session_id=YOUR_SESSION_ID" \
  --data @payload.json
```
`payload.json` 为浏览器 `navigator.credentials.create()` 的结果 JSON。

响应（示例）：
```json
{
  "message": "Passkey 绑定成功",
  "count": 1,
  "has_passkeys": true,
  "credentials": [
    {
      "credential_id": "credential_abc123",
      "created_at": "2026-03-26T20:15:00+08:00",
      "updated_at": "2026-03-26T20:15:00+08:00"
    }
  ]
}
```

### 查询已绑定 Passkey
请求：
```bash
curl -i http://localhost:8080/api/passkeys \
  -H "Cookie: session_id=YOUR_SESSION_ID"
```

响应（示例）：
```json
{
  "count": 2,
  "has_passkeys": true,
  "credentials": [
    {
      "credential_id": "credential_abc123",
      "created_at": "2026-03-26T20:15:00+08:00",
      "updated_at": "2026-03-26T20:15:00+08:00"
    }
  ]
}
```

### 删除已绑定 Passkey
请求：
```bash
curl -i -X DELETE http://localhost:8080/api/passkeys/credential_abc123 \
  -H "Cookie: session_id=YOUR_SESSION_ID"
```

响应（示例）：
```json
{
  "message": "Passkey 已删除",
  "count": 1,
  "has_passkeys": true,
  "credentials": []
}
```

### Passkey 登录
#### begin
请求：
```bash
curl -i http://localhost:8080/api/passkey/login/begin \
  -H "Content-Type: application/json" \
  --data '{"email":"user@example.com"}'
```
响应（示例）：
```json
{
  "publicKey": {
    "challenge": "A0a6c9xVt_AbQKfLwZBk",
    "allowCredentials": [
      { "type": "public-key", "id": "credential-id" }
    ],
    "timeout": 300000,
    "userVerification": "preferred"
  },
  "session_id": "PASSKEY_SESSION_ID"
}
```

#### finish
请求：
```bash
curl -i http://localhost:8080/api/passkey/login/finish \
  -H "Content-Type: application/json" \
  -H "X-Passkey-Session: PASSKEY_SESSION_ID" \
  --data @payload.json
```
`payload.json` 为浏览器 `navigator.credentials.get()` 的结果 JSON。

响应（示例）：
```json
{ "message": "登录成功", "user_id": "user-id", "username": "user" }
```

## HTTPS 部署注意事项
### 必须使用 HTTPS
Passkey 只能在安全上下文下使用：
- 生产环境必须是 `https://`
- 本地开发仅允许 `http://localhost`

### 证书与域名
- 证书域名必须和 `PASSKEY_ORIGIN` 一致
- `PASSKEY_RP_ID` 必须是 `PASSKEY_ORIGIN` 的主机名（无端口）

### 反向代理
若通过 Nginx/Ingress 代理，请确保：
- `Origin` 正常透传
- `X-Forwarded-Proto` 正确设置为 `https`

## Safari / iOS 说明
### 支持要求
- Safari 需要安全上下文
- iOS/iPadOS 16+ 才完整支持 Passkey

### 常见限制
- 仅支持平台认证器（Touch ID/Face ID）
- 低版本系统或禁用 iCloud Keychain 会导致创建失败

### 调试建议
- 优先在 macOS Safari 测试
- 使用 `http://localhost`，避免 `127.0.0.1`/`::1` 的 RP 不匹配

## 关键文件
- 后端配置：`internal/app/dock/config.go`
- WebAuthn 初始化：`internal/app/dock/app.go`
- Passkey 处理：`internal/app/dock/passkey.go`
- CORS：`internal/app/dock/cors.go`
- UI：`ui/public/login.html`、`ui/public/dashboard.html`
