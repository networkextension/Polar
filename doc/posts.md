# 帖子业务说明

本文件用于补充帖子业务的详细说明，便于前端与 iOS 客户端实现。

## 1. 核心对象与字段解释

### 1.1 Post（帖子）

字段 | 说明 | 类型
--- | --- | ---
id | 帖子 ID | number
user_id | 发帖用户 ID | string
username | 发帖用户名 | string
tag_id | 标签 ID（可为空） | number \| null
content | 帖子正文 | string
created_at | 发布时间（UTC） | string (ISO8601)
like_count | 点赞数 | number
reply_count | 回复数 | number
liked_by_me | 当前登录用户是否已点赞 | boolean
images | 图片 URL 列表 | string[]

### 1.2 PostReply（回复）

字段 | 说明 | 类型
--- | --- | ---
id | 回复 ID | number
post_id | 所属帖子 ID | number
user_id | 回复用户 ID | string
username | 回复用户名 | string
content | 回复内容 | string
created_at | 回复时间（UTC） | string (ISO8601)

### 1.3 Tag（标签）

字段 | 说明 | 类型
--- | --- | ---
id | 标签 ID | number
name | 标签名称 | string
slug | 标签标识（唯一） | string
description | 描述 | string
sort_order | 排序权重（越大越靠前） | number
created_at | 创建时间 | string (ISO8601)
updated_at | 更新时间 | string (ISO8601)

## 2. 业务规则

1. **发帖至少 1 张图片**  
   `POST /api/posts` 必须带 `images` 文件数组，空数组会返回 400。

2. **点赞为幂等操作**  
   重复点赞不会报错，`like_count` 保持不变。取消点赞同理。

3. **回复列表按时间升序**  
   便于“对话”式阅读。

4. **帖子列表按时间倒序**  
   新帖子在最上面。

5. **标签可选**  
   `tag_id` 可为空，用于后续板块功能扩展。

## 3. 前端交互建议

### 3.1 帖子广场（posts.html）

- 页面显示最新帖子列表
- 每条帖子展示：
  - 用户名、发布时间、正文、图片缩略图
  - 点赞按钮 + 数量
  - “查看详情”跳转
- 点赞按钮点击后立即更新 UI

### 3.2 帖子详情（post.html）

- 顶部展示帖子内容与图片
- 点赞按钮（同广场）
- 回复区：默认展开，支持实时追加
- 提交回复成功后刷新回复列表

### 3.3 发帖（post.html / 未来可拆出 new-post.html）

- 内容为必填
- 图片至少 1 张
- 成功后跳转至详情页

## 4. 错误码约定（建议）

当前后端统一返回：
```json
{ "error": "..." }
```

建议前端/iOS 处理方式：

HTTP 状态码 | 说明 | UI 处理
--- | --- | ---
400 | 参数错误/缺少字段 | 显示表单错误提示
401 | 未登录/会话失效 | 跳转登录
403 | 权限不足 | 显示“无权限”提示
404 | 资源不存在 | 显示空态页
409 | 数据冲突（如 slug 冲突） | 显示冲突提示
500 | 服务器异常 | 显示系统错误提示

## 5. iOS 端实现提示

- 图片上传需使用 `multipart/form-data`，字段名 `images`
- 帖子详情建议缓存图片 URL
- 点赞状态由 `liked_by_me` 驱动，避免本地猜测

