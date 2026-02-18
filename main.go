package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"ai-guardian-challenge/internal/config"
	"ai-guardian-challenge/internal/handler"
	"ai-guardian-challenge/internal/service"
	"ai-guardian-challenge/internal/store"
)

func main() {
	// 切换工作目录到可执行文件所在目录，确保相对路径（config.yaml、data.db）正确
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		os.Chdir(execDir)
		log.Printf("工作目录: %s", execDir)
	}

	// 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化 SQLite 存储
	dataStore := store.New("data.db")
	defer dataStore.Close()

	// 初始化 AI 服务
	aiService := service.NewAIService(
		cfg.AI.APIURL,
		cfg.AI.APIKey,
		cfg.AI.Model,
		cfg.AI.SystemPrompt,
	)

	// 初始化口令检测器
	passwordChecker := service.NewPasswordChecker(
		cfg.Game.Passwords.Grand,
		cfg.Game.Passwords.Consolation,
	)

	// 初始化 Handler
	authHandler := handler.NewAuthHandler(dataStore, cfg)
	infoHandler := handler.NewInfoHandler(dataStore, cfg)

	// 确定上传目录（web/Pic/）
	uploadDir := filepath.Join("web", "Pic")
	os.MkdirAll(uploadDir, 0755)
	uploadHandler := handler.NewUploadHandler(uploadDir)

	chatHandler := handler.NewChatHandler(dataStore, cfg, aiService, passwordChecker)

	// 创建路由
	mux := http.NewServeMux()

	// ========== API 路由 ==========
	// 公开接口：无需登录
	mux.HandleFunc("/api/info", infoHandler.GetSiteInfo)
	mux.HandleFunc("/api/check-auth", authHandler.CheckAuth)
	mux.HandleFunc("/api/login", authHandler.Login)
	mux.HandleFunc("/api/logout", authHandler.Logout)
	mux.HandleFunc("/api/verify-captcha", authHandler.VerifyCaptcha)
	mux.HandleFunc("/api/winners", infoHandler.GetWinners)
	mux.HandleFunc("/api/public/conversations", infoHandler.GetPublicConversations)

	// 需登录接口
	mux.HandleFunc("/api/conversations", infoHandler.GetUserConversations)
	mux.HandleFunc("/api/conversation/new", chatHandler.NewConversation)
	mux.HandleFunc("/api/conversation/message", chatHandler.SendMessage)
	mux.HandleFunc("/api/upload-image", uploadHandler.UploadImage)
	mux.HandleFunc("/api/conversation/bonus-choice", chatHandler.BonusChoice)

	// 对话详情路由（支持 /api/conversation/{id} 格式）
	mux.HandleFunc("/api/conversation/", chatHandler.GetConversation)

	// ========== 静态文件 ==========
	// 上传的图片目录
	mux.Handle("/Pic/", http.StripPrefix("/Pic/", http.FileServer(http.Dir(uploadDir))))

	// Web 静态文件
	webDir := "web"
	mux.Handle("/style.css", http.FileServer(http.Dir(webDir)))
	mux.Handle("/app.js", http.FileServer(http.Dir(webDir)))
	mux.Handle("/user.js", http.FileServer(http.Dir(webDir)))
	mux.Handle("/chat.js", http.FileServer(http.Dir(webDir)))
	mux.Handle("/user.html", http.FileServer(http.Dir(webDir)))
	mux.Handle("/chat.html", http.FileServer(http.Dir(webDir)))
	mux.Handle("/conversation.html", http.FileServer(http.Dir(webDir)))

	// 首页（index.html）
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
			return
		}
		// 尝试从 web 目录提供其他静态资源
		http.FileServer(http.Dir(webDir)).ServeHTTP(w, r)
	})

	// 启动服务器
	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Server.Port)
	log.Printf("🚀 AI 守护者挑战游戏服务已启动")
	log.Printf("📍 访问地址: http://0.0.0.0:%d", cfg.Server.Port)
	log.Printf("⏰ 活动截止: %s", cfg.Game.Deadline)
	log.Printf("🔑 主口令: %s", cfg.Game.Passwords.Grand)
	log.Printf("🎁 彩蛋口令: %s", cfg.Game.Passwords.Consolation)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
