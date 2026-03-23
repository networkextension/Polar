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
  "role": "admin",
  "icon_url": "/uploads/icon_user_xxx.png"
}
```

## 登录记录

**GET** `/api/login-history?limit=5`

权限要求：已登录用户

成功响应：
```json
{
  "records": [
    {
      "id": 1,
      "user_id": "u123",
      "ip_address": "127.0.0.1",
      "country": "China",
      "region": "Shanghai",
      "city": "Shanghai",
      "login_method": "password",
      "logged_in_at": "2026-03-22T08:00:00Z"
    }
  ]
}
```

## 用户头像

### 上传当前用户头像

**POST** `/api/user/icon`

权限要求：已登录用户

请求类型：`multipart/form-data`

字段：
- `icon`：图片文件，建议不超过 2MB

成功响应：
```json
{
  "message": "更新成功",
  "icon_url": "/uploads/icon_u123_20260322.png"
}
```

## 用户 Profile 与 Recommendation

说明：

- 更完整的 Profile 设计说明见 `doc/profile.md`
- Profile 页面地址：
  - `/profile.html`
  - `/profile.html?user_id=<targetUserId>`

### 获取用户 Profile

**GET** `/api/users/:id/profile`

权限要求：已登录用户

成功响应：
```json
{
  "profile": {
    "user_id": "u_018",
    "username": "Bob",
    "icon_url": "/uploads/icon_u_018.png",
    "bio": "做过活动执行、打扫卫生、搬运协助类零工，周末时间比较灵活。",
    "created_at": "2026-03-21T09:00:00+08:00",
    "is_me": false,
    "can_recommend": true,
    "recommendations": [
      {
        "id": 1,
        "target_user_id": "u_018",
        "author_user_id": "u_001",
        "author_username": "Alice",
        "author_user_icon": "/uploads/icon_u_001.png",
        "content": "做事很靠谱，现场响应很快，沟通也顺畅。",
        "created_at": "2026-03-23T10:00:00+08:00",
        "updated_at": "2026-03-23T10:00:00+08:00"
      }
    ]
  }
}
```

### 更新自己的 Profile

**PUT** `/api/users/me/profile`

权限要求：已登录用户

请求体：
```json
{
  "bio": "我擅长活动执行、临时搬运和打扫整理，周末全天可接单。"
}
```

说明：

- 当前接口仅更新 `bio`
- 头像继续通过 **POST** `/api/user/icon` 上传

成功响应：
```json
{
  "message": "保存成功",
  "profile": {
    "user_id": "u_018",
    "username": "Bob",
    "icon_url": "/uploads/icon_u_018.png",
    "bio": "我擅长活动执行、临时搬运和打扫整理，周末全天可接单。",
    "created_at": "2026-03-21T09:00:00+08:00",
    "is_me": true,
    "can_recommend": false,
    "recommendations": []
  }
}
```

### 写入或更新 Recommendation

**POST** `/api/users/:id/recommendations`

权限要求：已登录用户

请求体：
```json
{
  "content": "做事很认真，沟通及时，现场执行力不错。"
}
```

说明：

- `:id` 为被推荐用户 ID
- 不允许给自己写 Recommendation
- 同一作者再次提交时会覆盖之前的 Recommendation 内容

成功响应：
```json
{
  "message": "Recommendation 已保存",
  "profile": {
    "user_id": "u_018",
    "username": "Bob",
    "icon_url": "/uploads/icon_u_018.png",
    "bio": "做过活动执行、打扫卫生、搬运协助类零工，周末时间比较灵活。",
    "created_at": "2026-03-21T09:00:00+08:00",
    "is_me": false,
    "can_recommend": true,
    "recommendations": [
      {
        "id": 1,
        "target_user_id": "u_018",
        "author_user_id": "u_001",
        "author_username": "Alice",
        "author_user_icon": "/uploads/icon_u_001.png",
        "content": "做事很认真，沟通及时，现场执行力不错。",
        "created_at": "2026-03-23T10:00:00+08:00",
        "updated_at": "2026-03-23T12:00:00+08:00"
      }
    ]
  }
}
```

## 站点设置

### 获取站点设置

**GET** `/api/site-settings`

说明：公开接口，登录前后都可读取。

成功响应：
```json
{
  "site": {
    "name": "Polar-",
    "description": "AI-assisted product prototyping workspace",
    "icon_url": "/uploads/site_icon_20260322.png",
    "updated_at": "2026-03-22T08:00:00Z"
  }
}
```

### 更新站点设置（仅管理员）

**PUT** `/api/site-settings`

权限要求：管理员

请求体：
```json
{
  "name": "Polar-",
  "description": "AI-assisted product prototyping workspace"
}
```

成功响应：
```json
{
  "message": "保存成功",
  "site": {
    "name": "Polar-",
    "description": "AI-assisted product prototyping workspace",
    "icon_url": "/uploads/site_icon_20260322.png",
    "updated_at": "2026-03-22T08:00:00Z"
  }
}
```

### 上传站点图标（仅管理员）

**POST** `/api/site-settings/icon`

权限要求：管理员

请求类型：`multipart/form-data`

字段：
- `icon`：图片文件，建议不超过 2MB

成功响应：
```json
{
  "message": "更新成功",
  "icon_url": "/uploads/site_icon_20260322.png",
  "site": {
    "name": "Polar-",
    "description": "AI-assisted product prototyping workspace",
    "icon_url": "/uploads/site_icon_20260322.png",
    "updated_at": "2026-03-22T08:00:00Z"
  }
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

## 帖子与板块筛选

说明：

- 帖子支持绑定一个可选的 `tag_id`
- `tag_id` 可以理解为 BBS 风格的“板块”
- 帖子列表支持按 `post_type` 与 `tag_id` 组合筛选

### 发布帖子时指定板块

**POST** `/api/posts`

权限要求：已登录用户

请求类型：`multipart/form-data`

字段补充：

- `tag_id`：可选，指定帖子所属 Tag / 板块
- `post_type`：可选，默认 `standard`；任务帖传 `task`

示例：
```bash
curl -X POST http://localhost:3000/api/posts \
  -b cookie.txt \
  -F tag_id=3 \
  -F post_type=standard \
  -F content='这是一个 Go 讨论帖子'
```

成功响应：
```json
{
  "message": "发布成功",
  "id": 88,
  "post_type": "standard",
  "tag_id": 3,
  "images": [],
  "videos": [],
  "video_items": [],
  "content": "这是一个 Go 讨论帖子",
  "created": "2026-03-23T09:00:00+08:00"
}
```

### 获取帖子列表并筛选

**GET** `/api/posts?limit=10&offset=0&post_type=all&tag_id=3`

权限要求：已登录用户

支持查询参数：

- `limit`：分页大小
- `offset`：分页偏移
- `post_type`：`all | standard | task`
- `tag_id`：可选，按板块筛选

说明：

- 如果只传 `post_type=task`，可筛选零工任务帖
- 如果只传 `tag_id=3`，可筛选某个板块下的所有帖子
- 如果两个参数同时传，则表示“在指定板块中筛指定类型”

示例响应：
```json
{
  "posts": [
    {
      "id": 88,
      "user_id": "u_001",
      "username": "Alice",
      "tag_id": 3,
      "post_type": "standard",
      "content": "这是一个 Go 讨论帖子",
      "created_at": "2026-03-23T09:00:00+08:00",
      "like_count": 5,
      "reply_count": 2,
      "liked_by_me": false,
      "images": [],
      "videos": []
    }
  ],
  "has_more": false,
  "next_offset": 1
}
```

## 零工任务模块

说明：

- 零工任务复用帖子系统，任务帖通过 `post_type = "task"` 区分。
- 更完整的设计说明见 `doc/tasks.md`。

### 发布任务帖

**POST** `/api/posts`

权限要求：已登录用户

请求类型：`multipart/form-data`

字段：
- `post_type`：固定传 `task`
- `content`：任务描述，必填
- `task_location`：地理位置，可选
- `task_start_at`：任务开始时间，RFC3339，必填
- `task_end_at`：任务结束时间，RFC3339，必填
- `working_hours`：工作时长或班次说明，必填
- `apply_deadline`：申请截止时间，RFC3339，必填
- `images`：图片文件数组，可选
- `videos`：视频文件数组，可选

成功响应：
```json
{
  "message": "发布成功",
  "id": 101,
  "post_type": "task",
  "images": [],
  "videos": [],
  "video_items": [],
  "content": "周末商场活动需要 2 名兼职",
  "created": "2026-03-23T09:00:00+08:00"
}
```

### 获取任务帖列表/详情

任务帖仍然使用帖子接口：

- **GET** `/api/posts?limit=10&offset=0`
- **GET** `/api/posts/:id`

当帖子为任务帖时，返回对象中会附带 `task` 字段，例如：

```json
{
  "id": 101,
  "post_type": "task",
  "content": "周末商场活动需要 2 名兼职",
  "task": {
    "post_id": 101,
    "location": "上海徐汇",
    "start_at": "2026-03-28T09:00:00+08:00",
    "end_at": "2026-03-28T18:00:00+08:00",
    "working_hours": "9:00-18:00，共 8 小时",
    "apply_deadline": "2026-03-27T20:00:00+08:00",
    "application_status": "open",
    "applicant_count": 3,
    "applied_by_me": false,
    "can_apply": true,
    "can_manage": false
  }
}
```

### 申请任务

**POST** `/api/tasks/:id/apply`

权限要求：已登录用户

成功响应：
```json
{
  "message": "申请成功"
}
```

### 撤销申请

**DELETE** `/api/tasks/:id/apply`

权限要求：已登录用户

成功响应：
```json
{
  "message": "已撤销申请"
}
```

### 查看申请者列表

**GET** `/api/tasks/:id/applications`

权限要求：任务发布者

成功响应：
```json
{
  "applications": [
    {
      "id": 9001,
      "post_id": 101,
      "user_id": "u_018",
      "username": "Bob",
      "user_icon": "/uploads/icon_xxx.png",
      "applied_at": "2026-03-23T12:00:00+08:00"
    }
  ]
}
```

### 关闭申请

**POST** `/api/tasks/:id/close`

权限要求：任务发布者

成功响应：
```json
{
  "message": "已关闭申请"
}
```

### 选择候选人并发送私信

**POST** `/api/tasks/:id/select-candidate`

权限要求：任务发布者

请求体：
```json
{
  "applicant_user_id": "u_018",
  "message_template": "你好，你已被选为该零工任务候选人。如果确认参与，请直接回复我。"
}
```

说明：

- `applicant_user_id` 必填，且必须是当前有效申请者。
- `message_template` 可为空；为空时服务端会生成默认模板。
- 成功后任务会自动关闭申请，并通过现有私聊系统发送消息。

成功响应：
```json
{
  "message": "候选人已确认，私信已发送",
  "chat_id": 12,
  "message_id": 88,
  "message_template": "你好，你已被选为该零工任务候选人。如果确认参与，请直接回复我。"
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

权限要求：已登录用户

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
  "video_items": [
    {
      "url": "/uploads/20260319_120001_efgh5678.mp4",
      "poster_url": "/uploads/20260319_120001_efgh5678_poster.jpg"
    }
  ],
  "content": "今天分享一个 Go 小技巧。",
  "tag_id": 1,
  "created": "2026-03-19T12:00:00Z"
}
```

### 获取帖子列表

**GET** `/api/posts?limit=10&offset=0`

权限要求：已登录用户

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
      "videos": ["/uploads/20260319_120001_efgh5678.mp4"],
      "video_items": [
        {
          "url": "/uploads/20260319_120001_efgh5678.mp4",
          "poster_url": "/uploads/20260319_120001_efgh5678_poster.jpg"
        }
      ]
    }
  ],
  "has_more": false,
  "next_offset": 1
}
```

### 获取帖子详情

**GET** `/api/posts/:id`

权限要求：已登录用户

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
    "videos": ["/uploads/20260319_120001_efgh5678.mp4"],
    "video_items": [
      {
        "url": "/uploads/20260319_120001_efgh5678.mp4",
        "poster_url": "/uploads/20260319_120001_efgh5678_poster.jpg"
      }
    ]
  }
}
```

### 删除帖子

**DELETE** `/api/posts/:id`

权限要求：管理员或帖子作者本人

成功响应：
```json
{
  "message": "帖子已删除"
}
```

### 点赞

**POST** `/api/posts/:id/like`

权限要求：已登录用户

成功响应：
```json
{
  "message": "已点赞"
}
```

### 取消点赞

**DELETE** `/api/posts/:id/like`

权限要求：已登录用户

成功响应：
```json
{
  "message": "已取消点赞"
}
```

### 发表回复

**POST** `/api/posts/:id/replies`

权限要求：已登录用户

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

权限要求：已登录用户

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
- `videos` 字段保留为纯地址数组以兼容旧客户端；新客户端应优先使用 `video_items[].poster_url` 展示统一封面。
- 删帖权限规则：管理员可删除任意帖子；普通用户只能删除自己发布的帖子。
- 帖子删除后，列表接口不会再返回该帖子；详情接口访问该帖子会返回 `404`。

## 新建 Markdown 记录

**POST** `/api/markdown`

请求体：
```json
{
  "title": "Demo Note",
  "content": "# 标题\\n\\n正文内容",
  "is_public": true
}
```

成功响应：
```json
{
  "message": "保存成功",
  "id": 1,
  "file": "data/markdown/xxx.md",
  "username": "johndoe",
  "is_public": true
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
      "is_public": true,
      "uploaded_at": "2026-03-18T00:00:00Z"
    }
  ],
  "has_more": true,
  "next_offset": 10
}
```

## 读取 Markdown 记录

**GET** `/api/markdown/:id`

权限要求：已登录用户  
说明：作者始终可读；如果文档设置为 `is_public=true`，其他已登录用户也可读。

成功响应：
```json
{
  "entry": {
    "id": 1,
    "user_id": "xxxx",
    "title": "Demo Note",
    "file_path": "data/markdown/xxx.md",
    "is_public": true,
    "uploaded_at": "2026-03-18T00:00:00Z"
  },
  "content": "# 标题\\n\\n正文内容",
  "can_edit": false
}
```

## 更新 Markdown 记录

**PUT** `/api/markdown/:id`

请求体：
```json
{
  "title": "新标题",
  "content": "# 新标题\\n\\n更新内容",
  "is_public": false
}
```

成功响应：
```json
{
  "message": "更新成功",
  "id": 1,
  "is_public": false
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

## 公开 Markdown

### 分页获取公开 Markdown 列表

**GET** `/api/public/markdowns?limit=10&offset=0`

说明：公开接口，返回所有 `is_public=true` 的 Markdown。

成功响应：
```json
{
  "entries": [
    {
      "id": 12,
      "user_id": "u123",
      "username": "johndoe",
      "user_icon": "/uploads/icon_u123.png",
      "title": "Demo Note",
      "uploaded_at": "2026-03-22T08:00:00Z"
    }
  ],
  "has_more": false,
  "next_offset": 1
}
```

### 读取公开 Markdown 详情

**GET** `/api/public/markdown/:id`

说明：
- 文档公开时，任何用户都可访问
- 如果当前请求携带有效登录态且该用户是文档作者，也可访问自己的非公开文档

成功响应：
```json
{
  "entry": {
    "id": 12,
    "user_id": "u123",
    "title": "Demo Note",
    "file_path": "data/markdown/demo_note.md",
    "is_public": true,
    "uploaded_at": "2026-03-22T08:00:00Z"
  },
  "content": "# Demo Note\\n\\n正文内容",
  "can_edit": false
}
```
