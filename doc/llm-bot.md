# LLM Bot 重构说明

本文记录当前 LLM Bot 能力的重构结果，方便后续继续演进知识库、Push、Bot 市场和更多多轮对话能力。

## 目标

这轮重构解决了三个核心问题：

- 不再让所有 AI 能力都绑定到全局 `system`
- 允许普通用户维护自己的 `LLM Config`，并基于配置创建 `bot user`
- 在同一个 Bot 私聊下引入 `llm_thread`，把 IM 会话和 LLM 上下文分开
- 把 Bot 人设和模型连接信息拆开，避免 `system_prompt` 与模型切换互相污染

## 当前结构

现在系统里有三层会话语义：

1. `chat_thread`
- 表示“谁和谁在私聊”
- 仍然是整个聊天系统的主会话对象

2. `chat_message`
- 表示“这个私聊里发生过什么消息事件”
- 普通文本、`shared_markdown`、失败态 AI 消息都还是消息

3. `llm_thread`
- 表示“这个 Bot 私聊下的一个独立话题”
- 同一个 `chat_thread` 可以有多个 `llm_thread`
- LLM 构建上下文时按 `llm_thread` 取最近消息，而不是整条私聊历史

这个拆分的关键意义是：

- IM 会话可以长期保留
- LLM 上下文可以按话题重开
- 后续群聊 Bot、文档内对话、任务内对话也更容易复用

## 运行角色

当前 AI 相关角色分两类：

### `system`

- 系统内置用户
- 作为官方默认 AI 助理存在
- 发给 `system` 的私信会转给后台 AI agent
- `system` 在构建上下文时，会读取程序运行目录中的文档摘要

### `bot user`

- 每个 Bot 都对应一个真实的 `user_id`
- 普通用户可以创建、编辑、删除自己的 Bot
- Bot 自己持有默认 `LLM Config`
- Bot 自己持有运行时 `system_prompt`
- 发给 Bot 的私信会按“当前话题配置 + Bot Prompt”调用模型
- Bot 不读取程序运行目录文档，只读取当前 `llm_thread` 的聊天上下文

## 数据模型

### `llm_configs`

用于保存用户自己的模型接入配置。

核心字段：

- `owner_user_id`
- `name`
- `base_url`
- `model`
- `api_key`

这一层现在只负责“怎么连模型”，不再承载 Bot 的人设 Prompt。

当前说明：

- `api_key` 由服务端保存
- 接口不会把明文 key 回传前端
- 目前尚未加密存储，后续可切换为 env key 或独立密钥加密

### `bot_users`

用于保存用户创建的 Bot。

核心字段：

- `owner_user_id`
- `bot_user_id`
- `llm_config_id`
- `name`
- `description`
- `system_prompt`

设计选择：

- Bot 不是额外的虚拟对象，而是特殊 `user`
- 这样可以直接复用现有私聊、会话列表、在线状态和消息广播链路
- Bot 的默认模型配置和 Bot Prompt 都放在这一层

### `llm_threads`

用于保存 Bot 私聊下的话题。

核心字段：

- `chat_thread_id`
- `owner_user_id`
- `bot_user_id`
- `llm_config_id`
- `title`
- `last_message_at`

当前行为：

- 新建话题默认标题为“新话题”
- 首轮真正发送消息后，服务端会自动生成摘要标题
- 用户也可以手动重命名话题
- 用户也可以给当前话题切换 `LLM Config`
- 切换只影响后续回复，不会改变 Bot Prompt

### `chat_messages`

本轮新增了两个与 AI 体验直接相关的字段：

- `llm_thread_id`
- `failed`

用途：

- `llm_thread_id` 用来把消息挂到具体话题下
- `failed = true` 用来标记失败态 AI 消息，支持前端显示重试入口

## 消息流

### 用户给 Bot 发消息

1. 客户端向 `POST /api/chats/:id/messages` 发送消息
2. 若是 AI 会话，可同时带 `llm_thread_id`
3. 服务端先写入用户消息
4. 识别对端是否为 `system` 或 `bot user`
5. 若是 AI 会话，则把任务投递给后台 AI agent
6. `bot user` 会按“thread 当前 `LLM Config` + Bot 自身 `system_prompt`”生成回复

### AI 回复成功

1. AI agent 读取运行配置
2. 构建上下文
3. 调用模型接口
4. 若回复较长，则先写入 `markdown_entries`
5. 再向聊天里写一条 `shared_markdown` 消息
6. 通过 websocket 广播给当前会话参与者

### AI 回复失败

1. AI agent 调用模型失败
2. 服务端生成一条失败态文本消息
3. 该消息会写入 `chat_messages`
4. 这条消息带 `failed = true`
5. 前端展示失败标识和 `Retry` 按钮

### Retry

1. 客户端调用 `POST /api/chats/:id/messages/:messageId/retry`
2. 服务端校验目标消息必须是 `failed = true`
3. 找到这条失败消息之前的上一条用户消息
4. 将该用户消息重新投递给 AI agent
5. 原失败消息被标记为 `deleted_by = "retry"`
6. 前端在收到 `revoke` 事件后，直接把失败消息从列表移除

这样做的好处是：

- 用户不用手动重新输入同样的问题
- 重试成功后不会残留一条无效失败消息

## 为什么要有 `shared_markdown`

AI 长回复如果直接落在 `chat_messages.content` 里，会带来几个问题：

- 会话列表最后一条预览会过长
- 聊天消息表会积累大量长文本
- 复制、收藏、公开分享不方便复用

所以现在长回复改成两步：

1. 先写入 `markdown_entries`
2. 聊天里只发一条 `shared_markdown` 引用消息

前端基于这条引用消息提供：

- 放大 / 缩小
- 复制 Markdown 原文
- 公开分享
- 收藏到自己的 Markdown

## 上下文策略

当前上下文构建规则：

- `system`：运行目录文档摘要 + 当前 `llm_thread` 最近消息
- `bot user`：当前 `llm_thread` 最近消息

这样做的原因：

- 官方助理需要具备站内文档感知能力
- 用户自建 Bot 不应默认读取服务器运行目录文件
- 用户自建 Bot 更适合围绕自己的 Bot Prompt 和会话上下文工作

## 配置职责划分

当前推荐的职责边界如下：

- `LLM Config`
  - 负责 `Base URL / Model / API Key`
  - 负责“怎么连这个模型”

- `Bot User`
  - 负责 Bot 名称、简介、默认 `LLM Config`
  - 负责 Bot 的运行时 `system_prompt`
  - 负责“这个 Bot 是谁”

- `llm_thread`
  - 负责当前话题使用哪个 `LLM Config`
  - 不负责 Prompt
  - 负责“这次对话用哪个模型配置”

## UI 对应关系

### Dashboard

- 普通用户可以在设置中心管理 `LLM Config`
- 普通用户可以在单独的 `Bot 管理` 分栏创建和维护 `Bot User`
- 可在 Bot 列表里直接跳转到私聊

### Chat

- AI 会话顶部支持选择当前话题
- 支持新建话题
- 支持重命名话题
- 用户自建 Bot 的话题支持切换当前 `LLM Config`
- `shared_markdown` 消息支持展开、复制、公开分享、收藏
- 失败态 AI 消息支持重试

## 这轮重构后的边界

当前已经支持：

- 用户级 `LLM Config`
- 用户级 `Bot User`
- AI 私聊多话题
- 长回复 `shared_markdown`
- 失败态消息与 Retry

当前还没有做：

- `api_key` 加密存储
- Bot 专属知识库
- 群聊里的 Bot
- 工具调用
- 多 provider 的更深兼容层
- 话题归档、删除、摘要卡片列表

## 后续建议

下一阶段最值得做的方向：

1. `api_key` 加密存储
- 避免明文落库

2. Bot 知识库
- 让 Bot 绑定自己的 Markdown / 文件集合

3. Thread 生命周期
- 增加归档、删除、摘要、固定等操作

4. Push 集成
- AI 回复用户离线时，结合设备表和 Push Token 发送提醒
