# API 文档

基础路径：`/api`  
认证方式：基于 Cookie 的 Session，登录成功后服务端会下发 `session_id`。

## 通用请求头

以下请求头当前主要用于登录态建立、设备登记与后续 Push 能力预埋：

- `X-Device-Type`：可选，登录/注册/Passkey 登录时上报设备类型。
  - 允许值：`browser`、`ios`、`android`
  - 未传时默认按 `browser` 处理
- `X-Push-Token`：可选，登录/注册/Passkey 登录时上报当前设备的 Push Token
  - 服务端会按 `user_id + device_type` 维度保存最近一次 token
  - 当前版本仅保存，不会直接触发推送

## 通用返回

- 成功：HTTP 2xx + JSON
- 失败：HTTP 4xx/5xx + `{ "error": "..." }`

## 注册

**POST** `/api/register`

可选请求头：

- `X-Device-Type`
- `X-Push-Token`

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

可选请求头：

- `X-Device-Type`
- `X-Push-Token`

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

说明：

- 当前接口仅清理 Session Cookie
- 不会主动清空 `user_devices.push_token`
- 在线状态主要由 websocket 连接生命周期驱动

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
  "icon_url": "/uploads/icon_user_xxx.png",
  "bio": "我擅长活动执行、临时搬运和打扫整理，周末全天可接单。",
  "is_online": true,
  "device_type": "browser",
  "last_seen_at": "2026-03-24T10:20:30+08:00"
}
```

字段说明：

- `is_online`：当前用户聚合在线状态
- `device_type`：当前用户最近活跃设备类型
- `last_seen_at`：最近一次活跃时间，可能为空

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
      "device_type": "browser",
      "logged_in_at": "2026-03-22T08:00:00Z"
    }
  ]
}
```

`login_method` 当前可能值：

- `register`
- `password`
- `passkey`

`device_type` 当前可能值：

- `browser`
- `ios`
- `android`

## Passkey 登录

说明：

- Passkey 登录完成接口与普通登录一样，也支持通过请求头登记设备类型与 `push token`

### 发起 Passkey 登录

**POST** `/api/passkey/login/begin`

请求体：
```json
{
  "email": "john@example.com"
}
```

成功响应：
```json
{
  "session_id": "passkey_login_xxx",
  "publicKey": {
    "challenge": "xxx",
    "rpId": "localhost",
    "allowCredentials": []
  }
}
```

### 完成 Passkey 登录

**POST** `/api/passkey/login/finish`

请求头：

- `X-Passkey-Session`：必填，来自 begin 接口返回的 `session_id`
- `X-Device-Type`：可选
- `X-Push-Token`：可选

请求体：

- WebAuthn / Passkey 标准登录响应对象，字段较长，此处省略

成功响应：
```json
{
  "message": "登录成功",
  "user_id": "xxxx",
  "username": "johndoe"
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
- 当前前端入口包括：
  - dashboard 顶部的“我的 Profile”
  - 帖子作者头像/用户名
  - 任务申请者头像/用户名
  - 任务成果提交者头像/用户名

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
- 前端当前使用“覆盖式提交”，不是多条追加评论

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

## 用户自定义 LLM 与 Bot

说明：

- 以下接口对普通已登录用户开放，每个用户只可管理自己的配置和 Bot
- `api_key` 当前由服务端保存，但接口不会回传明文
- 建议先调用“测试配置”确认连通，再保存配置
- 每个 Bot 都会对应一个可私聊的 `user_id`
- Bot 与官方 `system` 助理共用同一套私聊入口，但运行时配置来源不同
- 当前推荐的职责划分：
  - `LLM Config`：保存 `Base URL / Model / API Key` 等连接信息
  - `Bot User`：保存 Bot 的默认 `LLM Config` 和运行时 `system_prompt`
  - `llm_thread`：只负责切换当前话题使用哪个 `LLM Config`

### 获取 LLM Config 列表

**GET** `/api/llm-configs`

权限要求：已登录用户

成功响应：
```json
{
  "configs": [
    {
      "id": 3,
      "owner_user_id": "u_018",
      "name": "OpenAI 生产配置",
      "base_url": "https://api.openai.com/v1/chat/completions",
      "model": "gpt-4.1-mini",
      "has_api_key": true,
      "created_at": "2026-03-24T12:00:00+08:00",
      "updated_at": "2026-03-24T12:00:00+08:00"
    }
  ]
}
```

### 测试 LLM Config

**POST** `/api/llm-configs/test`

权限要求：已登录用户

说明：

- 直接使用本次请求体进行模型连通性测试
- 不会写入数据库
- 适合在前端“保存配置”前先检测 `Base URL / Model / API Key` 是否可用
- `system_prompt` 在这里仅用于测试时构造一轮请求，不会保存为 Bot 的运行时 Prompt

请求体：
```json
{
  "base_url": "https://api.openai.com/v1/chat/completions",
  "model": "gpt-4.1-mini",
  "api_key": "sk-xxx",
  "system_prompt": "你是一个活动策划助手。"
}
```

成功响应：
```json
{
  "message": "连接成功，模型配置可用"
}
```

失败响应示例：
```json
{
  "error": "Not found the model gpt-4.1-mini or Permission denied"
}
```

### 创建 LLM Config

**POST** `/api/llm-configs`

权限要求：已登录用户

请求体：
```json
{
  "name": "OpenAI 生产配置",
  "base_url": "https://api.openai.com/v1/chat/completions",
  "model": "gpt-4.1-mini",
  "api_key": "sk-xxx"
}
```

成功响应：
```json
{
  "message": "配置已创建",
  "config": {
    "id": 3,
    "owner_user_id": "u_018",
    "name": "OpenAI 生产配置",
    "base_url": "https://api.openai.com/v1/chat/completions",
    "model": "gpt-4.1-mini",
    "has_api_key": true,
    "created_at": "2026-03-24T12:00:00+08:00",
    "updated_at": "2026-03-24T12:00:00+08:00"
  }
}
```

### 更新 LLM Config

**PUT** `/api/llm-configs/:id`

权限要求：已登录用户且为配置拥有者

请求体：
```json
{
  "name": "OpenAI 生产配置",
  "base_url": "https://api.openai.com/v1/chat/completions",
  "model": "gpt-4.1-mini",
  "api_key": "sk-new",
  "update_api_key": true
}
```

说明：

- `update_api_key = false` 或不传时，服务端保持原有 key 不变
- 编辑已有配置时可只改 `name/base_url/model`

### 删除 LLM Config

**DELETE** `/api/llm-configs/:id`

权限要求：已登录用户且为配置拥有者

成功响应：
```json
{
  "message": "配置已删除"
}
```

注意：

- 若该配置仍被 Bot 引用，删除会失败，应先改绑或删除对应 Bot

### 获取 Bot 列表

**GET** `/api/bots`

权限要求：已登录用户

成功响应：
```json
{
  "bots": [
    {
      "id": 5,
      "owner_user_id": "u_018",
      "bot_user_id": "bot_abcd1234efgh5678",
      "name": "翻译助手",
      "description": "负责中英文翻译和润色",
      "system_prompt": "你是一个专业翻译助手，请优先保持原意，再兼顾自然表达。",
      "llm_config_id": 3,
      "config_name": "OpenAI 生产配置",
      "created_at": "2026-03-24T12:10:00+08:00",
      "updated_at": "2026-03-24T12:10:00+08:00"
    }
  ]
}
```

### 创建 Bot

**POST** `/api/bots`

权限要求：已登录用户

请求体：
```json
{
  "name": "翻译助手",
  "description": "负责中英文翻译和润色",
  "system_prompt": "你是一个专业翻译助手，请优先保持原意，再兼顾自然表达。",
  "llm_config_id": 3
}
```

成功响应：
```json
{
  "message": "Bot 已创建",
  "bot": {
    "id": 5,
    "owner_user_id": "u_018",
    "bot_user_id": "bot_abcd1234efgh5678",
    "name": "翻译助手",
    "description": "负责中英文翻译和润色",
    "llm_config_id": 3,
    "config_name": "OpenAI 生产配置",
    "created_at": "2026-03-24T12:10:00+08:00",
    "updated_at": "2026-03-24T12:10:00+08:00"
  }
}
```

### 更新 Bot

**PUT** `/api/bots/:id`

权限要求：已登录用户且为 Bot 拥有者

请求体：
```json
{
  "name": "翻译助手",
  "description": "负责中英文翻译和润色",
  "llm_config_id": 3
}
```

成功响应：
```json
{
  "message": "Bot 已更新"
}
```

### 删除 Bot

**DELETE** `/api/bots/:id`

权限要求：已登录用户且为 Bot 拥有者

成功响应：
```json
{
  "message": "Bot 已删除"
}
```

说明：

- 删除 Bot 时，会同时删除该 Bot 对应的 bot user
- 之后不能再通过原来的 `bot_user_id` 发起私聊

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

说明：
- 帖子图片上传后，服务端会保留原图并生成两个衍生尺寸：
  - `small_url`：适合列表/缩略图
  - `medium_url`：适合详情页正文展示
  - `original_url`：原始图片
- `images` 字段继续保留，兼容旧客户端；其值会优先返回 `medium_url`

成功响应：
```json
{
  "message": "发布成功",
  "id": 12,
  "images": ["/uploads/20260319_120000_abcd1234_md.jpg"],
  "image_items": [
    {
      "small_url": "/uploads/20260319_120000_abcd1234_sm.jpg",
      "medium_url": "/uploads/20260319_120000_abcd1234_md.jpg",
      "original_url": "/uploads/20260319_120000_abcd1234.png"
    }
  ],
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
      "images": ["/uploads/20260319_120000_abcd1234_md.jpg"],
      "image_items": [
        {
          "small_url": "/uploads/20260319_120000_abcd1234_sm.jpg",
          "medium_url": "/uploads/20260319_120000_abcd1234_md.jpg",
          "original_url": "/uploads/20260319_120000_abcd1234.png"
        }
      ],
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
    "images": ["/uploads/20260319_120000_abcd1234_md.jpg"],
    "image_items": [
      {
        "small_url": "/uploads/20260319_120000_abcd1234_sm.jpg",
        "medium_url": "/uploads/20260319_120000_abcd1234_md.jpg",
        "original_url": "/uploads/20260319_120000_abcd1234.png"
      }
    ],
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
- 图片建议优先使用 `image_items`：
  - 列表卡片使用 `small_url`
  - 详情页使用 `medium_url`
  - 原图预览或下载使用 `original_url`
- `images` 字段保留为兼容字段，值优先为中图地址，缺失时回退到原图地址。
- `videos` 字段保留为纯地址数组以兼容旧客户端；新客户端应优先使用 `video_items[].poster_url` 展示统一封面。
- 删帖权限规则：管理员可删除任意帖子；普通用户只能删除自己发布的帖子。
- 帖子删除后，列表接口不会再返回该帖子；详情接口访问该帖子会返回 `404`。

## 私聊与在线状态

说明：

- 当前私聊 websocket 地址为 `/ws/chat`
- websocket 连接成功后，服务端会把对应用户设备标记为在线
- websocket 断开后，服务端会更新该设备离线状态，并重新聚合用户在线状态
- 这些在线状态是为了下一步“离线用户 Push 通知”做准备
- 系统内置了一个 `system` 用户作为 AI 助理
- 发给 `system` 的私信会转给后台 AI agent
- 用户也可以创建自己的 `bot user`，发给这些 Bot 的私信会按其绑定的 LLM Config 转给后台 AI agent
- `system` 会读取程序运行目录下的文档摘要作为上下文的一部分
- 用户自建 `bot user` 不读取运行目录文档，只读取当前 `llm thread` 下的消息上下文
- Bot 会话支持 `llm_thread`，用于在同一个私聊里拆分多个话题
- AI agent 的长回复会先写入 `markdown_entries`，再作为 `shared_markdown` 消息返回聊天线程
- AI 调用失败时，会写入一条 `failed = true` 的失败消息；客户端可调用重试接口重新投递上一条用户消息

### AI 助理状态

**GET** `/api/system-agent`

权限要求：已登录用户

成功响应：
```json
{
  "user_id": "system",
  "username": "system",
  "ready": true,
  "message": "system 助理可通过 user_id=system 发起私聊"
}
```

### 创建或获取私聊会话

**POST** `/api/chats/start`

权限要求：已登录用户

请求体：
```json
{
  "user_id": "u_018"
}
```

成功响应：
```json
{
  "chat": {
    "id": 12,
    "other_user_id": "system",
    "other_username": "system",
    "other_user_icon": "",
    "other_user_online": false,
    "other_user_device_type": "browser",
    "other_user_last_seen_at": "2026-03-24T10:20:30+08:00",
    "last_message": "[AI 文档回复]",
    "last_message_at": "2026-03-24T10:20:30+08:00",
    "created_at": "2026-03-24T09:00:00+08:00",
    "unread_count": 0
  }
}
```

### 获取会话列表

**GET** `/api/chats?limit=20&offset=0`

权限要求：已登录用户

成功响应：
```json
{
  "chats": [
    {
      "id": 12,
      "other_user_id": "u_018",
      "other_username": "Bob",
      "other_user_icon": "/uploads/icon_u_018.png",
      "other_user_online": false,
      "other_user_device_type": "android",
      "other_user_last_seen_at": "2026-03-24T09:55:00+08:00",
      "last_message": "好的，收到",
      "last_message_at": "2026-03-24T09:50:00+08:00",
      "created_at": "2026-03-24T09:00:00+08:00",
      "unread_count": 2
    }
  ],
  "has_more": false,
  "next_offset": 1
}
```

字段说明：

- `other_user_online`：对方当前是否在线
- `other_user_device_type`：对方最近活跃设备类型
- `other_user_last_seen_at`：对方最近在线时间，可能为空

### 获取 Bot 话题列表

**GET** `/api/chats/:id/llm-threads?active_thread_id=1`

权限要求：会话参与者，且该会话对端必须是 `system` 或 `bot user`

说明：

- 用于列出当前 AI 会话下的话题列表
- `active_thread_id` 可选，用于指定当前激活话题
- 若当前会话尚无话题，服务端会按需要返回默认话题
- `llm_config_id / config_name / config_model` 表示这个话题当前实际使用的模型配置

成功响应：
```json
{
  "threads": [
    {
      "id": 21,
      "chat_thread_id": 12,
      "owner_user_id": "u_018",
      "bot_user_id": "bot_translate_01",
      "llm_config_id": 3,
      "config_name": "OpenAI 生产配置",
      "config_model": "gpt-4.1-mini",
      "title": "合同翻译",
      "created_at": "2026-03-25T09:00:00+08:00",
      "updated_at": "2026-03-25T09:10:00+08:00",
      "last_message_at": "2026-03-25T09:10:00+08:00"
    }
  ],
  "active_thread": {
    "id": 21,
    "chat_thread_id": 12,
    "owner_user_id": "u_018",
    "bot_user_id": "bot_translate_01",
    "llm_config_id": 3,
    "config_name": "OpenAI 生产配置",
    "config_model": "gpt-4.1-mini",
    "title": "合同翻译",
    "created_at": "2026-03-25T09:00:00+08:00",
    "updated_at": "2026-03-25T09:10:00+08:00",
    "last_message_at": "2026-03-25T09:10:00+08:00"
  }
}
```

### 创建 Bot 话题

**POST** `/api/chats/:id/llm-threads`

权限要求：会话参与者，且该会话对端必须是 `system` 或 `bot user`

请求体：
```json
{
  "title": "新话题"
}
```

成功响应：
```json
{
  "message": "话题已创建",
  "thread": {
    "id": 22,
    "chat_thread_id": 12,
    "owner_user_id": "u_018",
    "bot_user_id": "bot_translate_01",
    "title": "新话题",
    "created_at": "2026-03-25T09:30:00+08:00",
    "updated_at": "2026-03-25T09:30:00+08:00",
    "last_message_at": null
  },
  "threads": []
}
```

### 更新 Bot 话题标题

**PUT** `/api/chats/:id/llm-threads/:threadId`

权限要求：会话参与者且为该话题拥有者

请求体：
```json
{
  "title": "报价整理"
}
```

### 切换 Bot 话题模型配置

**PUT** `/api/chats/:id/llm-threads/:threadId/config`

权限要求：会话参与者且该话题对端必须是用户自建 `bot user`

说明：

- 只切换当前 `llm_thread` 使用的 `LLM Config`
- 不会修改 Bot 自身的默认配置
- 不会影响该 Bot 的其他话题
- 只影响后续回复，不会重跑历史消息
- Bot 的运行时 `system_prompt` 不跟随这里切换，仍然来自 Bot 自身配置

请求体：
```json
{
  "llm_config_id": 4
}
```

成功响应：
```json
{
  "message": "当前话题模型已切换，后续回复将使用新配置",
  "thread": {
    "id": 22,
    "chat_thread_id": 12,
    "owner_user_id": "u_018",
    "bot_user_id": "bot_translate_01",
    "llm_config_id": 4,
    "config_name": "Qwen 备用配置",
    "config_model": "qwen-plus",
    "title": "报价整理",
    "created_at": "2026-03-25T09:30:00+08:00",
    "updated_at": "2026-03-25T09:40:00+08:00",
    "last_message_at": "2026-03-25T09:34:00+08:00"
  },
  "threads": []
}
```

成功响应：
```json
{
  "message": "话题标题已更新",
  "thread": {
    "id": 22,
    "chat_thread_id": 12,
    "owner_user_id": "u_018",
    "bot_user_id": "bot_translate_01",
    "title": "报价整理",
    "created_at": "2026-03-25T09:30:00+08:00",
    "updated_at": "2026-03-25T09:35:00+08:00",
    "last_message_at": "2026-03-25T09:34:00+08:00"
  },
  "threads": []
}
```

### 获取会话消息

**GET** `/api/chats/:id/messages?limit=200&offset=0&llm_thread_id=21`

权限要求：会话参与者

说明：

- 普通私聊可不传 `llm_thread_id`
- AI 会话建议传 `llm_thread_id`，仅拉取当前话题消息
- 该接口会同时返回当前激活话题

成功响应：
```json
{
  "messages": [
    {
      "id": 88,
      "thread_id": 12,
      "llm_thread_id": 21,
      "sender_id": "system",
      "sender_username": "system",
      "sender_icon": "",
      "message_type": "shared_markdown",
      "failed": false,
      "content": "以下是本次 AI 回复的摘要预览……",
      "markdown_entry_id": 135,
      "markdown_title": "活动执行 SOP 建议",
      "created_at": "2026-03-24T10:20:30+08:00",
      "deleted": false
    }
  ],
  "active_thread": {
    "id": 21,
    "chat_thread_id": 12,
    "owner_user_id": "u_018",
    "bot_user_id": "system",
    "title": "活动执行 SOP",
    "created_at": "2026-03-24T10:00:00+08:00",
    "updated_at": "2026-03-24T10:20:30+08:00",
    "last_message_at": "2026-03-24T10:20:30+08:00"
  },
  "active_thread_id": 21,
  "has_more": false,
  "next_offset": 1
}
```

`message_type` 当前可能值：

- `text`：普通文本消息
- `shared_markdown`：共享 Markdown 消息

通用字段补充：

- `llm_thread_id`：消息所属话题，普通私聊可能为空
- `failed`：是否为失败态 AI 消息；通常只会出现在 `system` 或 `bot user` 回复失败时

当 `message_type = "shared_markdown"` 时：

- `content`：仅用于消息卡片和会话列表中的简短预览
- `markdown_entry_id`：对应的 Markdown 记录 ID
- `markdown_title`：对应 Markdown 标题

### 读取共享 Markdown 消息正文

**GET** `/api/chats/:id/messages/:messageId/markdown`

权限要求：会话参与者

说明：

- 仅适用于 `message_type = "shared_markdown"` 的消息
- 前端可用该接口实现“放大/缩小”“复制”“公开分享”“收藏”

成功响应：
```json
{
  "entry": {
    "id": 135,
    "user_id": "system",
    "title": "活动执行 SOP 建议",
    "file_path": "data/markdown/activity-sop_20260324_system.md",
    "is_public": false,
    "uploaded_at": "2026-03-24T10:20:30+08:00"
  },
  "content": "# 活动执行 SOP 建议\n\n1. 提前确认场地\n2. 明确人员分工",
  "message": {
    "id": 88,
    "thread_id": 12,
    "sender_id": "system",
    "sender_username": "system",
    "message_type": "shared_markdown",
    "content": "以下是本次 AI 回复的摘要预览……",
    "markdown_entry_id": 135,
    "markdown_title": "活动执行 SOP 建议",
    "created_at": "2026-03-24T10:20:30+08:00"
  },
  "can_edit": false
}
```

### 发送消息

**POST** `/api/chats/:id/messages`

权限要求：会话参与者

请求体：
```json
{
  "content": "你好",
  "llm_thread_id": 21
}
```

说明：

- 普通私聊不需要传 `llm_thread_id`
- AI 会话传入 `llm_thread_id` 后，消息会进入指定话题
- 若 `llm_thread_id` 对应话题标题仍是默认值“新话题”，服务端会在首轮消息后自动生成摘要标题

成功响应：
```json
{
  "message": "发送成功",
  "id": 88,
  "active_thread": {
    "id": 21,
    "chat_thread_id": 12,
    "owner_user_id": "u_018",
    "bot_user_id": "system",
    "title": "活动执行 SOP",
    "created_at": "2026-03-24T10:00:00+08:00",
    "updated_at": "2026-03-24T10:20:30+08:00",
    "last_message_at": "2026-03-24T10:20:30+08:00"
  }
}
```

### 重试失败的 AI 消息

**POST** `/api/chats/:id/messages/:messageId/retry`

权限要求：会话参与者

说明：

- 仅适用于 AI 会话中的失败消息，即 `failed = true`
- 服务端会找到这条失败消息前的上一条用户消息，重新投递给 AI agent
- 若重试请求被接受，原失败消息会被服务端标记为 `deleted_by = "retry"`，客户端应直接从列表移除

成功响应：
```json
{
  "message": "已重新提交上一条用户消息",
  "content": "请帮我整理这份活动执行 SOP"
}
```

### 撤回消息

**DELETE** `/api/chats/:id/messages/:messageId`

权限要求：消息发送者本人

成功响应：
```json
{
  "message": "撤回成功"
}
```

### WebSocket

**GET** `/ws/chat`

认证方式：

- 依赖登录后下发的 Cookie Session
- 当前连接建立时会读取会话中的设备类型信息
- `shared_markdown` 消息也会通过 `message` 事件下发，但事件里只包含预览和引用信息

客户端收到的事件类型如下。

#### 新消息事件 `message`

```json
{
  "type": "message",
  "chat_id": 12,
  "message": {
    "id": 88,
    "thread_id": 12,
    "llm_thread_id": 21,
    "sender_id": "system",
    "sender_username": "system",
    "message_type": "shared_markdown",
    "failed": false,
    "content": "以下是本次 AI 回复的摘要预览……",
    "markdown_entry_id": 135,
    "markdown_title": "活动执行 SOP 建议",
    "created_at": "2026-03-24T10:20:30+08:00"
  }
}
```

#### 已读事件 `read`

```json
{
  "type": "read",
  "chat_id": 12,
  "user_id": "u_018",
  "read_at": "2026-03-24T10:21:00+08:00"
}
```

#### 撤回事件 `revoke`

```json
{
  "type": "revoke",
  "chat_id": 12,
  "message_id": 88,
  "deleted_at": "2026-03-24T10:22:00+08:00",
  "user_id": "retry"
}
```

补充说明：

- 当 `user_id = "retry"` 时，表示该条失败 AI 消息已被重试流程替换，前端可直接把这条消息从列表中移除，而不是显示“消息已撤回”

#### 在线状态事件 `presence`

```json
{
  "type": "presence",
  "user_id": "u_018",
  "online": true,
  "device_type": "ios",
  "last_seen_at": "2026-03-24T10:23:00+08:00"
}
```

字段说明：

- `user_id`：发生在线状态变化的用户
- `online`：聚合后的用户在线状态
- `device_type`：最近活跃设备类型
- `last_seen_at`：最近活跃时间

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

补充说明：

- 聊天里的 `shared_markdown` 消息执行“公开分享”时，会复用该接口并传 `is_public = true`
- 执行“收藏”时，同样复用该接口，但会传 `is_public = false`
- 收藏成功后，前端通常会跳转到 `/editor.html?id=<id>` 继续编辑

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
