# 安装指南（INSTALL.md）

## 环境要求

| 依赖 | 版本要求 | 说明 |
|------|----------|------|
| Go | ≥ 1.21 | 编译后端代码 |
| Git | 任意 | 克隆代码仓库 |
| OpenAI 兼容 API | - | 需要有效的 API 密钥和可访问的端点 |

> **注意**：本项目使用纯 Go 实现的 SQLite 驱动（`modernc.org/sqlite`），无需安装 C 编译器或 CGO 环境。

## 安装步骤

### 1. 克隆代码

```bash
git clone <仓库地址>
cd ai-guardian-challenge
```

### 2. 安装 Go 依赖

```bash
go mod download
```

### 3. 配置文件

编辑 `config.yaml`，至少需要填写以下必填项：

```yaml
ai:
  api_url: "https://api.openai.com/v1/chat/completions"  # AI 接口地址
  api_key: "sk-xxxxxxxx"                                   # API 密钥（必填）
  model: "gpt-4"                                           # 模型名称

admin:
  contact: "你的QQ号"       # 管理员 QQ（用于判断管理员身份）
  password: "强密码"        # 管理员密码（请勿使用默认值）
```

详见 [ENV_VARS.md](ENV_VARS.md) 了解全部配置项。

### 4. 编译

```bash
# 编译为当前平台可执行文件
go build -o ai-guardian .

# 交叉编译 Linux（服务器部署用）
GOOS=linux GOARCH=amd64 go build -o ai-guardian .
```

### 5. 运行

```bash
./ai-guardian
```

服务启动后访问 `http://localhost:8080` 即可。

## 运行时文件

程序运行后会自动创建以下文件：

| 文件 | 说明 |
|------|------|
| `data.db` | SQLite 数据库（用户、对话、获奖记录等） |
| `web/Pic/` | 用户上传的图片目录 |

> ⚠️ `data.db` 包含所有业务数据，请定期备份。
