# 私聊系统设计与 API（给 iOS 接入）

本文整理私聊的核心设计、数据模型、HTTP API、WebSocket 事件，以及示例，方便 iOS 端快速对接。

## 设计概览
- 私聊是双人会话（thread）+ 消息（message）模型。
- 会话通过两位用户 ID 生成唯一对话。
- 未读数通过最后已读时间与消息时间计算。
- WebSocket 负责实时推送，HTTP 作为兜底。

## 数据模型（服务端）
chat_threads 字段：
- `id` BIGSERIAL
- `user_low` TEXT
- `user_high` TEXT
- `created_at` TIMESTAMPTZ
- `last_message` TEXT
- `last_message_at` TIMESTAMPTZ

chat_messages 字段：
- `id` BIGSERIAL
- `thread_id` BIGINT
- `sender_id` TEXT
- `content` TEXT
- `created_at` TIMESTAMPTZ
- `deleted_at` TIMESTAMPTZ (可空)
- `deleted_by` TEXT (可空)

chat_reads 字段：
- `thread_id` BIGINT
- `user_id` TEXT
- `last_read_at` TIMESTAMPTZ

## 认证方式
- 使用登录后 `SessionCookieName`（HTTP Cookie）认证。
- iOS 端需保存并回传 Cookie。

## HTTP API

### 1. 会话列表
`GET /api/chats?limit=20&offset=0`

返回字段
- `chats`: 会话数组
- `has_more`, `next_offset`

示例响应
```json
{
  "chats": [
    {
      "id": 12,
      "other_user_id": "abc123",
      "other_username": "Alice",
      "last_message": "在吗？",
      "last_message_at": "2026-03-19T09:15:00+08:00",
      "created_at": "2026-03-19T09:00:00+08:00",
      "unread_count": 2
    }
  ],
  "has_more": false,
  "next_offset": 1
}
```

### 2. 创建/获取会话
`POST /api/chats/start`

请求
```json
{ "user_id": "abc123" }
```

响应
```json
{
  "chat": {
    "id": 12,
    "other_user_id": "abc123",
    "other_username": "Alice",
    "last_message": "",
    "last_message_at": null,
    "created_at": "2026-03-19T09:00:00+08:00",
    "unread_count": 0
  }
}
```

### 3. 获取消息列表（并自动标记已读）
`GET /api/chats/:id/messages?limit=50&offset=0`

响应
```json
{
  "messages": [
    {
      "id": 88,
      "thread_id": 12,
      "sender_id": "me123",
      "sender_username": "Me",
      "content": "你好",
      "created_at": "2026-03-19T09:10:00+08:00",
      "deleted": false
    },
    {
      "id": 89,
      "thread_id": 12,
      "sender_id": "abc123",
      "sender_username": "Alice",
      "content": "消息已撤回",
      "created_at": "2026-03-19T09:11:00+08:00",
      "deleted": true,
      "deleted_at": "2026-03-19T09:12:00+08:00",
      "deleted_by": "abc123"
    }
  ],
  "has_more": false,
  "next_offset": 2
}
```

### 4. 发送消息
`POST /api/chats/:id/messages`

请求
```json
{ "content": "你好" }
```

响应
```json
{ "message": "发送成功", "id": 88 }
```

### 5. 撤回消息
`DELETE /api/chats/:id/messages/:messageId`

响应
```json
{ "message": "已撤回" }
```

## WebSocket 实时推送

### 连接
`GET /ws/chat`（WebSocket 升级）

iOS 端连接时需带 Cookie（Session）。

### 事件结构
所有事件为 JSON：
```json
{
  "type": "message | read | revoke",
  "chat_id": 12,
  "message": { ... },
  "message_id": 88,
  "user_id": "me123",
  "read_at": "2026-03-19T09:12:00+08:00",
  "deleted_at": "2026-03-19T09:12:00+08:00"
}
```

### 事件示例

#### 新消息
```json
{
  "type": "message",
  "chat_id": 12,
  "message": {
    "id": 88,
    "thread_id": 12,
    "sender_id": "me123",
    "sender_username": "Me",
    "content": "你好",
    "created_at": "2026-03-19T09:10:00+08:00"
  }
}
```

#### 已读
```json
{
  "type": "read",
  "chat_id": 12,
  "user_id": "me123",
  "read_at": "2026-03-19T09:12:00+08:00"
}
```

#### 撤回
```json
{
  "type": "revoke",
  "chat_id": 12,
  "message_id": 88,
  "deleted_at": "2026-03-19T09:12:00+08:00"
}
```

## iOS 接入建议
- 先拉会话列表，再进入会话请求消息列表。
- 进入会话后建立 WebSocket 长连，订阅消息与已读事件。
- WS 断开时回退到轮询 `GET /api/chats` 与 `GET /api/chats/:id/messages`。
