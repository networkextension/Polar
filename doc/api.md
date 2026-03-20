# API 文档

基础路径：`/api`  
认证方式：基于 Cookie 的 Session，登录成功后服务端会下发 `session_id`。

## 通用返回

- 成功：HTTP 2xx + JSON
- 失败：HTTP 4xx/5xx + `{ "error": "..." }`

## 注册

**POST** `/api/register`

请求体：
```json
{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "password123"
}
```

成功响应：
```json
{
  "message": "注册成功",
  "user_id": "xxxx",
  "username": "johndoe"
}
```

## 登录

**POST** `/api/login`

请求体：
```json
{
  "email": "john@example.com",
  "password": "password123"
}
```

成功响应：
```json
{
  "message": "登录成功",
  "user_id": "xxxx",
  "username": "johndoe"
}
```

## 登出

**POST** `/api/logout`

成功响应：
```json
{
  "message": "已成功退出登录"
}
```

## 获取当前用户

**GET** `/api/me`

成功响应：
```json
{
  "user_id": "xxxx",
  "username": "johndoe",
  "role": "admin"
}
```

## 用户组说明

用户组通过 `role` 字段表示：
- `user`：普通用户组
- `admin`：管理用户组（可管理标签）

## 标签（Tag）

### 获取标签列表

**GET** `/api/tags?limit=20&offset=0`

成功响应：
```json
{
  "tags": [
    {
      "id": 1,
      "name": "Go语言",
      "slug": "golang",
      "description": "Go 相关讨论区",
      "sort_order": 10,
      "created_at": "2026-03-19T08:00:00Z",
      "updated_at": "2026-03-19T08:00:00Z"
    }
  ],
  "has_more": false,
  "next_offset": 1
}
```

### 创建标签（仅管理员）

**POST** `/api/tags`

请求体：
```json
{
  "name": "Go语言",
  "slug": "golang",
  "description": "Go 相关讨论区",
  "sort_order": 10
}
```

成功响应：
```json
{
  "message": "创建成功",
  "tag": {
    "id": 1,
    "name": "Go语言",
    "slug": "golang",
    "description": "Go 相关讨论区",
    "sort_order": 10,
    "created_at": "2026-03-19T08:00:00Z",
    "updated_at": "2026-03-19T08:00:00Z"
  }
}
```

### 更新标签（仅管理员）

**PUT** `/api/tags/:id`

请求体：
```json
{
  "name": "Go 语言",
  "slug": "go",
  "description": "Go 语言讨论区",
  "sort_order": 20
}
```

成功响应：
```json
{
  "message": "更新成功",
  "id": 1
}
```

### 删除标签（仅管理员）

**DELETE** `/api/tags/:id`

成功响应：
```json
{
  "message": "删除成功"
}
```

## 帖子（Posts）

### 发帖（图片/视频可选）

**POST** `/api/posts`

请求类型：`multipart/form-data`

字段：
- `content`（必填）：帖子内容
- `images`（可选，可多张）：图片文件（字段名固定 `images`）
- `videos`（可选，可多条）：视频文件（字段名固定 `videos`）
- `tag_id`（可选）：标签 ID

成功响应：
```json
{
  "message": "发布成功",
  "id": 12,
  "images": ["/uploads/20260319_120000_abcd1234.png"],
  "videos": ["/uploads/20260319_120001_efgh5678.mp4"],
  "content": "今天分享一个 Go 小技巧。",
  "tag_id": 1,
  "created": "2026-03-19T12:00:00Z"
}
```

### 获取帖子列表

**GET** `/api/posts?limit=10&offset=0`

成功响应：
```json
{
  "posts": [
    {
      "id": 12,
      "user_id": "u123",
      "username": "johndoe",
      "user_icon": "/uploads/avatar_u123.png",
      "tag_id": 1,
      "content": "今天分享一个 Go 小技巧。",
      "created_at": "2026-03-19T12:00:00Z",
      "like_count": 3,
      "reply_count": 2,
      "liked_by_me": true,
      "images": ["/uploads/20260319_120000_abcd1234.png"],
      "videos": ["/uploads/20260319_120001_efgh5678.mp4"]
    }
  ],
  "has_more": false,
  "next_offset": 1
}
```

### 获取帖子详情

**GET** `/api/posts/:id`

成功响应：
```json
{
  "post": {
    "id": 12,
    "user_id": "u123",
    "username": "johndoe",
    "user_icon": "/uploads/avatar_u123.png",
    "tag_id": 1,
    "content": "今天分享一个 Go 小技巧。",
    "created_at": "2026-03-19T12:00:00Z",
    "like_count": 3,
    "reply_count": 2,
    "liked_by_me": true,
    "images": ["/uploads/20260319_120000_abcd1234.png"],
    "videos": ["/uploads/20260319_120001_efgh5678.mp4"]
  }
}
```

### 点赞

**POST** `/api/posts/:id/like`

成功响应：
```json
{
  "message": "已点赞"
}
```

### 取消点赞

**DELETE** `/api/posts/:id/like`

成功响应：
```json
{
  "message": "已取消点赞"
}
```

### 发表回复

**POST** `/api/posts/:id/replies`

请求体：
```json
{
  "content": "写得很好，感谢分享！"
}
```

成功响应：
```json
{
  "message": "回复成功",
  "id": 101
}
```

### 获取回复列表

**GET** `/api/posts/:id/replies?limit=50&offset=0`

成功响应：
```json
{
  "replies": [
    {
      "id": 101,
      "post_id": 12,
      "user_id": "u456",
      "username": "alice",
      "user_icon": "/uploads/avatar_u456.png",
      "content": "写得很好，感谢分享！",
      "created_at": "2026-03-19T12:10:00Z"
    }
  ],
  "has_more": false,
  "next_offset": 1
}
```

### iOS 对接提示（Posts）

- 发帖请使用 `multipart/form-data`，媒体字段名固定为 `images` 和 `videos`。
- `images`、`videos` 都允许为空数组或不传；仅 `content` 必填。
- 帖子列表/详情中的媒体 URL 为相对路径（如 `/uploads/xxx.mp4`），iOS 侧请拼接服务端域名后再加载。

## 新建 Markdown 记录

**POST** `/api/markdown`

请求体：
```json
{
  "title": "Demo Note",
  "content": "# 标题\\n\\n正文内容"
}
```

成功响应：
```json
{
  "message": "保存成功",
  "id": 1,
  "file": "data/markdown/xxx.md",
  "username": "johndoe"
}
```

## 分页获取 Markdown 记录

**GET** `/api/markdown?limit=10&offset=0`

成功响应：
```json
{
  "entries": [
    {
      "id": 1,
      "user_id": "xxxx",
      "title": "Demo Note",
      "file_path": "data/markdown/xxx.md",
      "uploaded_at": "2026-03-18T00:00:00Z"
    }
  ],
  "has_more": true,
  "next_offset": 10
}
```

## 读取 Markdown 记录

**GET** `/api/markdown/:id`

成功响应：
```json
{
  "entry": {
    "id": 1,
    "user_id": "xxxx",
    "title": "Demo Note",
    "file_path": "data/markdown/xxx.md",
    "uploaded_at": "2026-03-18T00:00:00Z"
  },
  "content": "# 标题\\n\\n正文内容"
}
```

## 更新 Markdown 记录

**PUT** `/api/markdown/:id`

请求体：
```json
{
  "title": "新标题",
  "content": "# 新标题\\n\\n更新内容"
}
```

成功响应：
```json
{
  "message": "更新成功",
  "id": 1
}
```

## 删除 Markdown 记录

**DELETE** `/api/markdown/:id`

成功响应：
```json
{
  "message": "删除成功"
}
```
