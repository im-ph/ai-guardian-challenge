# AI 守护者挑战（AI Guardian Challenge）

一个互动式 AI 对话游戏，玩家通过与 AI 对话，尝试诱导 AI 说出它所守护的神秘口令来赢取奖品。

## 🎮 游戏规则

1. AI 守护着两个口令：**主口令**（特等奖）和**彩蛋口令**（安慰奖）
2. 每次对话最多 20 轮，玩家需要在有限轮次内获取口令
3. AI 会尽全力保护口令，但玩家可以通过各种策略尝试诱导
4. 系统实时检测 AI 回复中是否泄露了口令（支持容错匹配）
5. 对话累计达到一定轮次后，触发福利机制自动发放口令

## ✨ 核心特性

- **流式 AI 对话**：使用 SSE（Server-Sent Events）实时推送 AI 回复
- **实时口令检测**：三层容错匹配（精确 → 去标点 → 关键词片段）
- **福利机制**：累计 55 轮弹出二选一；放弃后累计 80 轮自动发放主口令
- **获奖系统**：首页实时展示成功获取口令的获奖者
- **图片上传**：对话中支持发送图片（多模态）
- **管理后台**：管理员可查看/隐藏对话记录

## 🏗️ 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25 + 标准库 `net/http` |
| 数据库 | SQLite（`modernc.org/sqlite`，纯 Go 无 CGO） |
| 前端 | 原生 HTML/CSS/JavaScript（无框架） |
| AI 接口 | OpenAI 兼容 Chat Completions API |
| 配置 | YAML（`gopkg.in/yaml.v3`） |

## 📁 项目结构

```
ai-guardian-challenge/
├── main.go                     # 程序入口：路由注册、服务初始化
├── config.yaml                 # 全局配置文件（详细注释）
├── data.db                     # SQLite 数据库（运行时自动创建）
├── go.mod / go.sum             # Go 模块依赖
├── internal/                   # 后端核心代码（私有包）
│   ├── config/config.go        #   配置文件解析与结构体定义
│   ├── handler/                #   HTTP 处理器层
│   │   ├── auth.go             #     登录/登出/认证检查
│   │   ├── chat.go             #     对话管理/消息发送/福利机制
│   │   ├── info.go             #     站点信息/获奖者/公开对话
│   │   └── upload.go           #     图片上传
│   ├── middleware/              #   中间件
│   ├── model/model.go          #   数据模型定义
│   ├── service/                #   业务逻辑层
│   │   ├── ai.go               #     AI 接口调用（流式）
│   │   └── password.go         #     口令检测（三层容错匹配）
│   └── store/store.go          #   数据持久化层（SQLite CRUD）
└── web/                        # 前端静态资源
    ├── index.html              #   首页（活动介绍/倒计时/获奖榜）
    ├── chat.html / chat.js     #   对话页面（流式消息/获奖弹窗）
    ├── user.html / user.js     #   用户中心（我的对话列表）
    ├── conversation.html       #   对话详情（公开查看）
    ├── app.js                  #   首页逻辑
    └── style.css               #   全局样式
```

## 🚀 快速开始

```bash
# 1. 安装依赖
go mod download

# 2. 配置 config.yaml（修改 AI API 密钥等）
cp config.yaml.example config.yaml  # 如有模板

# 3. 编译运行
go build -o ai-guardian .
./ai-guardian

# 4. 访问 http://localhost:8080
```

详见 [INSTALL.md](docs/INSTALL.md) 和 [USAGE.md](docs/USAGE.md)。

## 📚 文档索引

| 文档 | 说明 |
|------|------|
| [INSTALL.md](docs/INSTALL.md) | 环境准备与安装步骤 |
| [USAGE.md](docs/USAGE.md) | 使用说明与游戏流程 |
| [DEPLOY.md](docs/DEPLOY.md) | 生产环境部署指南 |
| [API.md](docs/API.md) | 后端 API 接口文档 |
| [ENV_VARS.md](docs/ENV_VARS.md) | 配置项参考（config.yaml） |
| [SPECIAL_ENV.md](docs/SPECIAL_ENV.md) | 特殊环境与边界情况说明 |
