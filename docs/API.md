# API 接口文档（API.md）

基础地址：`http://localhost:8080`

所有接口返回 `application/json` 格式。认证通过 Cookie（`session`）实现。

---

## 公开接口（无需登录）

### `GET /api/info` — 获取站点信息

返回活动配置和管理员联系方式，供前端渲染。

**响应示例：**

```json
{
  "deadline": "2026-02-20T00:00:00+08:00",
  "isExpired": false,
  "captchaType": "simple",
  "adminQQ": "375484682",
  "adminEmail": "unlock@wa.cx",
  "adminWechat": "x53059680"
}
```

---

### `POST /api/login` — 用户登录

**请求体：**

```json
{
  "contact": "QQ号或微信号",
  "nickname": "昵称",
  "captchaToken": "simple-verified"
}
```

**响应：**

```json
{ "success": true, "isAdmin": false }
```

登录成功后设置 `session` Cookie（HttpOnly，7天有效）。

---

### `GET /api/check-auth` — 检查认证状态

**响应：**

```json
{ "isLoggedIn": true, "isAdmin": false, "nickname": "PH" }
```

---

### `POST /api/logout` — 退出登录

清除 `session` Cookie。

---

### `GET /api/winners` — 获取获奖者列表

**参数：** `?page=1&pageSize=5`

**响应：**

```json
{
  "data": [
    {
      "nickname": "PH",
      "conversationId": "1771256669460-xxx",
      "category": "grand-first",
      "prizeType": "grand",
      "prizeAmount": "UCloud服务器",
      "password": "祝小喵科技群U在新的一年身体安康、万事如意",
      "timestamp": "2026-02-18T10:00:00+08:00"
    }
  ],
  "page": 1,
  "pageSize": 5,
  "total": 1,
  "totalPages": 1
}
```

---

### `GET /api/public/conversations` — 获取公开对话列表

**参数：** `?page=1&pageSize=15`

**响应：**

```json
{
  "data": [
    {
      "id": "1771256669460-xxx",
      "nickname": "PH",
      "isSuccess": false,
      "turnCount": 5,
      "preview": "用户的第一条消息...",
      "createdAt": "2026-02-18T10:00:00+08:00"
    }
  ],
  "page": 1,
  "pageSize": 15,
  "total": 10,
  "totalPages": 1
}
```

---

## 需登录接口

### `GET /api/conversations` — 获取当前用户对话列表

**参数：** `?page=1&pageSize=15`

返回格式同公开对话，但包含 `foundPassword` 和 `isActive` 字段。

---

### `POST /api/conversation/new` — 创建新对话

**请求体：**

```json
{ "turnstileToken": "simple-verified" }
```

**响应：**

```json
{
  "success": true,
  "conversationId": "1771256669460-xxx",
  "initialMessage": "你好！我是 AI 守护者..."
}
```

---

### `GET /api/conversation/{id}` — 获取对话详情

**响应：** 完整的 `Conversation` 对象，包含消息列表。

---

### `POST /api/conversation/message` — 发送消息（SSE 流式）

**请求体：**

```json
{
  "conversationId": "xxx",
  "message": "用户消息内容",
  "imageUrl": "/Pic/xxx.jpg"
}
```

**响应：** `text/event-stream`（SSE 格式）

SSE 事件类型：

| type | 说明 | 关键字段 |
|------|------|----------|
| `content` | AI 回复的文本片段 | `content` |
| `password_found` | 检测到口令泄露 | `password`, `prizeType`, `prizeAmount`, `isFirstWinner` |
| `bonus_offer` | 福利二选一弹窗 | `totalTurns`, `consolationPassword`, `consolationPrizeAmount` |
| `error` | 错误 | `content` |

流结束标记：`data: [DONE]`

---

### `POST /api/conversation/bonus-choice` — 福利口令选择

**请求体：**

```json
{
  "conversationId": "xxx",
  "choice": "claim"
}
```

`choice` 取值：`claim`（领取安慰奖口令）或 `continue`（放弃并继续挑战主口令）。

---

### `POST /api/upload-image` — 上传图片

**请求：** `multipart/form-data`，字段名 `image`，可选 `conversationId`

**响应：**

```json
{ "url": "/Pic/1771256669460-xxx.jpg" }
```

限制：仅支持图片格式，最大 10MB。
