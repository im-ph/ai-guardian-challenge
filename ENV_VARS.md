# 配置项参考（ENV_VARS.md）

本项目全部配置通过 `config.yaml` 文件管理，不使用环境变量。

## 完整配置项

### server — HTTP 服务器

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `server.port` | int | `8080` | HTTP 监听端口 |

### ai — AI 模型接口

| 配置项 | 类型 | 默认值 | 必填 | 说明 |
|--------|------|--------|------|------|
| `ai.api_url` | string | - | ✅ | OpenAI 兼容 API 端点（`/v1/chat/completions`） |
| `ai.api_key` | string | - | ✅ | API 密钥（Bearer Token） |
| `ai.model` | string | - | ✅ | 模型名称（如 `gpt-4`、`gemini-3-pro`） |
| `ai.system_prompt` | string | - | ✅ | AI 角色设定提示词（多行文本） |

> ⚠️ `system_prompt` 中的口令文本必须与 `game.passwords` 保持一致。

### game — 游戏活动

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `game.deadline` | string | - | 活动截止时间（ISO 8601，含时区） |
| `game.max_turns` | int | `20` | 单次对话最大轮次 |
| `game.max_message_length` | int | `1500` | 单条消息最大字符数 |
| `game.bonus_consolation_threshold` | int | `55` | 安慰奖福利触发轮次（0=禁用） |
| `game.bonus_grand_threshold` | int | `80` | 主口令福利触发轮次（0=禁用） |

### game.passwords — 口令

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `game.passwords.grand` | string | 主口令（特等奖），用于实时匹配检测 |
| `game.passwords.consolation` | string | 彩蛋口令（安慰奖） |

### game.prizes — 奖品

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `game.prizes.grand_amount` | string | 特等奖奖品描述（显示在前端） |
| `game.prizes.consolation_amount` | string | 安慰奖奖品描述 |
| `game.prizes.grand_count` | int | 特等奖名额上限 |
| `game.prizes.consolation_count` | int | 安慰奖名额上限 |

### admin — 管理员

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `admin.contact` | string | 管理员 QQ 号（同时用于身份判断） |
| `admin.email` | string | 管理员邮箱（显示在首页页脚） |
| `admin.wechat` | string | 管理员微信号（显示在获奖弹窗中） |
| `admin.password` | string | 管理员登录密码 |

## 安全提醒

- `ai.api_key` 和 `admin.password` 属于敏感信息，**严禁**提交到版本控制
- 建议将 `config.yaml` 加入 `.gitignore`，仅保留 `config.yaml.example` 作为模板
