# 部署指南（DEPLOY.md）

## 部署方式

本项目编译后为单个可执行文件，部署时只需：
1. 可执行文件（`ai-guardian`）
2. 配置文件（`config.yaml`）
3. 前端资源目录（`web/`）

三者放在同一目录下即可。

## Linux 服务器部署

### 1. 交叉编译

```bash
# 在开发机上编译 Linux 版本
GOOS=linux GOARCH=amd64 go build -o ai-guardian .
```

### 2. 上传文件

```bash
scp ai-guardian config.yaml user@server:/opt/ai-guardian/
scp -r web/ user@server:/opt/ai-guardian/web/
```

### 3. 创建 Systemd 服务

```ini
# /etc/systemd/system/ai-guardian.service
[Unit]
Description=AI Guardian Challenge Game
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/ai-guardian
ExecStart=/opt/ai-guardian/ai-guardian
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable ai-guardian
sudo systemctl start ai-guardian
```

### 4. 反向代理（Caddy 示例）

```caddyfile
game.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

或 Nginx：

```nginx
server {
    listen 80;
    server_name game.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        # SSE 流式传输必须关闭缓冲
        proxy_buffering off;
        proxy_set_header Connection '';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

> ⚠️ **关键**：SSE（流式对话）要求反向代理**关闭响应缓冲**（`proxy_buffering off`），否则 AI 回复会等全部生成完才一次性返回。

## Windows 部署

```powershell
# 编译
go build -o ai-guardian.exe .

# 运行（前台）
.\ai-guardian.exe

# 后台运行（推荐用 NSSM 注册为 Windows 服务）
nssm install AIGuardian C:\path\to\ai-guardian.exe
nssm start AIGuardian
```

## 数据备份

```bash
# SQLite 数据库备份（支持热备份）
cp /opt/ai-guardian/data.db /backup/data.db.$(date +%Y%m%d)

# 用户上传图片备份
cp -r /opt/ai-guardian/web/Pic/ /backup/Pic.$(date +%Y%m%d)/
```

## 健康检查

```bash
# 检查服务是否存活（返回站点信息即正常）
curl -s http://localhost:8080/api/info | jq .
```
