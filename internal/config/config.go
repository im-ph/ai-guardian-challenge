package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 全局配置结构体
type Config struct {
	Server ServerConfig `yaml:"server"`
	AI     AIConfig     `yaml:"ai"`
	Game   GameConfig   `yaml:"game"`
	Admin  AdminConfig  `yaml:"admin"`
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	Port int `yaml:"port"`
}

// AIConfig AI 提供商配置
type AIConfig struct {
	APIURL       string `yaml:"api_url"`
	APIKey       string `yaml:"api_key"`
	Model        string `yaml:"model"`
	SystemPrompt string `yaml:"system_prompt"`
}

// GameConfig 游戏规则配置
type GameConfig struct {
	Deadline         string          `yaml:"deadline"`
	MaxTurns         int             `yaml:"max_turns"`
	MaxMessageLength int             `yaml:"max_message_length"`
	Passwords        PasswordsConfig `yaml:"passwords"`
	Prizes           PrizesConfig    `yaml:"prizes"`
	// 福利机制：当用户总对话轮次达到阈值时，自动在 AI 回复中附带口令
	BonusConsolationThreshold int `yaml:"bonus_consolation_threshold"`
	BonusGrandThreshold       int `yaml:"bonus_grand_threshold"`
}

// PasswordsConfig 口令配置
type PasswordsConfig struct {
	Grand       string `yaml:"grand"`
	Consolation string `yaml:"consolation"`
}

// PrizesConfig 奖品配置
type PrizesConfig struct {
	GrandAmount       string `yaml:"grand_amount"`
	ConsolationAmount string `yaml:"consolation_amount"`
	ConsolationCount  int    `yaml:"consolation_count"`
	GrandCount        int    `yaml:"grand_count"`
}

// AdminConfig 管理员配置
type AdminConfig struct {
	Contact  string `yaml:"contact"`
	Email    string `yaml:"email"`
	Wechat   string `yaml:"wechat"`
	Password string `yaml:"password"`
}

// DeadlineTime 解析截止时间为 time.Time
func (c *Config) DeadlineTime() time.Time {
	t, err := time.Parse(time.RFC3339, c.Game.Deadline)
	if err != nil {
		// 如果解析失败，默认7天后
		return time.Now().Add(7 * 24 * time.Hour)
	}
	return t
}

// IsExpired 判断活动是否已过期
func (c *Config) IsExpired() bool {
	return time.Now().After(c.DeadlineTime())
}

// Load 从 YAML 文件加载配置
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// 设置默认值
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Game.MaxTurns == 0 {
		cfg.Game.MaxTurns = 20
	}
	if cfg.Game.MaxMessageLength == 0 {
		cfg.Game.MaxMessageLength = 1500
	}

	return cfg, nil
}
