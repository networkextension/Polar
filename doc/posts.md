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
post_type | 帖子类型，`standard` 或 `task` | string
content | 帖子正文 | string
created_at | 发布时间（UTC） | string (ISO8601)
like_count | 点赞数 | number
reply_count | 回复数 | number
liked_by_me | 当前登录用户是否已点赞 | boolean
images | 图片 URL 列表 | string[]
videos | 视频 URL 列表 | string[]
video_items | 视频对象列表，包含 `url`、`poster_url` | object[]

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

1. **发帖媒体可选**  
   `POST /api/posts` 中 `images`、`videos` 都是可选文件数组（字段名分别为 `images`、`videos`）。

2. **点赞为幂等操作**  
   重复点赞不会报错，`like_count` 保持不变。取消点赞同理。

3. **回复列表按时间升序**  
   便于“对话”式阅读。

4. **帖子列表按时间倒序**  
   新帖子在最上面。

5. **标签可选**  
   `tag_id` 可为空，用于后续板块功能扩展。

6. **帖子支持按板块筛选**  
   广场页支持：
   - 查看全部最新帖子
   - 查看普通帖子
   - 查看零工任务帖
   - 按指定 Tag / 板块筛选帖子

7. **任务帖仍属于帖子体系**  
   任务帖通过 `post_type = "task"` 区分，本质仍然是帖子，因此同样可以绑定 `tag_id`。

8. **删帖权限**  
   管理员可删除任意帖子；普通用户只能删除自己发布的帖子。

9. **删除后不可见**  
   帖子删除后不会出现在列表接口中，详情接口访问该帖子会返回 `404`。

## 3. 前端交互建议

### 3.1 帖子广场（posts.html）

- 页面显示最新帖子列表
- 顶部支持筛选：
  - 最新帖子
  - 普通帖子
  - 零工
  - Tag / 板块
- 每条帖子展示：
  - 用户名、发布时间、所属 Tag、正文、图片缩略图、视频播放器
  - 点赞按钮 + 数量
  - “查看详情”跳转
  - 当前用户为管理员或帖子作者本人时，显示“删除帖子”
- 点赞按钮点击后立即更新 UI

### 3.2 帖子详情（post.html）

- 顶部展示帖子内容、图片与视频
- 点赞按钮（同广场）
- 当前用户为管理员或帖子作者本人时，显示“删除帖子”
- 回复区：默认展开，支持实时追加
- 提交回复成功后刷新回复列表

### 3.3 发帖（post.html / 未来可拆出 new-post.html）

- 可选选择一个 Tag / 板块
- 普通帖子与任务帖共用发帖入口
- 内容为必填
- 图片可选上传（支持多选）
- 视频可选上传（支持多选）
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
- 视频上传需使用 `multipart/form-data`，字段名 `videos`
- 如需指定板块，提交 `tag_id`
- 如需筛选板块，可在列表接口传 `tag_id`
- 如需筛选帖子类型，可在列表接口传 `post_type=all|standard|task`
- 帖子详情建议缓存图片 URL
- 视频封面由服务端在上传后使用 `ffmpeg` 生成，并通过 `video_items[].poster_url` 返回
- 点赞状态由 `liked_by_me` 驱动，避免本地猜测
- 删帖按钮展示规则应与服务端权限一致：管理员可删任意帖子，普通用户只能删自己的帖子
