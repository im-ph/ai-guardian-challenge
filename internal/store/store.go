package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"ai-guardian-challenge/internal/model"

	_ "modernc.org/sqlite"
)

// Store SQLite 数据存储
type Store struct {
	db *sql.DB
}

// New 创建 SQLite 存储实例，自动初始化表结构
func New(dbPath string) *Store {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	s := &Store{db: db}
	s.initTables()
	return s
}

// initTables 初始化数据库表
func (s *Store) initTables() {
	queries := []string{
		// 用户表
		`CREATE TABLE IF NOT EXISTS users (
			id       TEXT PRIMARY KEY,
			contact  TEXT NOT NULL,
			nickname TEXT NOT NULL,
			is_admin INTEGER NOT NULL DEFAULT 0
		)`,

		// 会话表（session token -> user_id）
		`CREATE TABLE IF NOT EXISTS sessions (
			token   TEXT PRIMARY KEY,
			user_id TEXT NOT NULL
		)`,

		// 对话表
		`CREATE TABLE IF NOT EXISTS conversations (
			id             TEXT PRIMARY KEY,
			user_id        TEXT NOT NULL,
			nickname       TEXT NOT NULL,
			turn_count     INTEGER NOT NULL DEFAULT 0,
			max_turns      INTEGER NOT NULL DEFAULT 20,
			is_active      INTEGER NOT NULL DEFAULT 1,
			is_success     INTEGER NOT NULL DEFAULT 0,
			is_public      INTEGER NOT NULL DEFAULT 1,
			found_password TEXT NOT NULL DEFAULT '',
			last_message   TEXT NOT NULL DEFAULT '',
			created_at     DATETIME NOT NULL
		)`,

		// 消息表
		`CREATE TABLE IF NOT EXISTS messages (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id TEXT NOT NULL,
			role            TEXT NOT NULL,
			content         TEXT NOT NULL,
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		// 获奖者表
		`CREATE TABLE IF NOT EXISTS winners (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			nickname        TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			category        TEXT NOT NULL,
			prize_type      TEXT NOT NULL,
			prize_amount    TEXT NOT NULL,
			password        TEXT NOT NULL,
			timestamp       DATETIME NOT NULL
		)`,

		// 口令首次获取标记表
		`CREATE TABLE IF NOT EXISTS claim_status (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,

		// 用户福利口令状态表
		// status 可选值: "offered"(已弹出选择), "continued"(选择继续挑战), "claimed_consolation"(已领取福利口令), "claimed_grand"(已获得主口令)
		`CREATE TABLE IF NOT EXISTS user_bonus_status (
			user_id TEXT PRIMARY KEY,
			status  TEXT NOT NULL DEFAULT '',
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,

		// 索引：加速常用查询
		`CREATE INDEX IF NOT EXISTS idx_messages_conv_id ON messages(conversation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_is_public ON conversations(is_public)`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			log.Fatalf("初始化表结构失败: %v\nSQL: %s", err, q)
		}
	}

	// 初始化 claim_status 默认值（如果不存在）
	for _, key := range []string{"grand_first_claimed", "consolation_first_claimed", "consolation_claim_count"} {
		s.db.Exec(`INSERT OR IGNORE INTO claim_status (key, value) VALUES (?, '0')`, key)
	}
}

// Close 关闭数据库连接
func (s *Store) Close() {
	s.db.Close()
}

// ========== 用户操作 ==========

// GetOrCreateUser 获取或创建用户
func (s *Store) GetOrCreateUser(contact, nickname string) *model.User {
	userID := contact

	// 尝试获取已有用户
	user := s.getUserByID(userID)
	if user != nil {
		// 更新昵称
		s.db.Exec(`UPDATE users SET nickname = ? WHERE id = ?`, nickname, userID)
		user.Nickname = nickname
		return user
	}

	// 创建新用户
	_, err := s.db.Exec(
		`INSERT INTO users (id, contact, nickname, is_admin) VALUES (?, ?, ?, 0)`,
		userID, contact, nickname,
	)
	if err != nil {
		log.Printf("创建用户失败: %v", err)
		return nil
	}

	return &model.User{
		ID:       userID,
		Contact:  contact,
		Nickname: nickname,
		IsAdmin:  false,
	}
}

// getUserByID 通过 ID 查询用户
func (s *Store) getUserByID(userID string) *model.User {
	row := s.db.QueryRow(`SELECT id, contact, nickname, is_admin FROM users WHERE id = ?`, userID)
	var user model.User
	var isAdmin int
	err := row.Scan(&user.ID, &user.Contact, &user.Nickname, &isAdmin)
	if err != nil {
		return nil
	}
	user.IsAdmin = isAdmin == 1
	return &user
}

// ========== Session 操作 ==========

// CreateSession 创建会话
func (s *Store) CreateSession(userID string) string {
	token := fmt.Sprintf("%d-%s", time.Now().UnixMilli(), userID)
	s.db.Exec(`INSERT INTO sessions (token, user_id) VALUES (?, ?)`, token, userID)
	return token
}

// GetUserBySession 通过会话令牌获取用户
func (s *Store) GetUserBySession(token string) *model.User {
	row := s.db.QueryRow(`SELECT user_id FROM sessions WHERE token = ?`, token)
	var userID string
	if err := row.Scan(&userID); err != nil {
		return nil
	}
	return s.getUserByID(userID)
}

// DeleteSession 删除会话
func (s *Store) DeleteSession(token string) {
	s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
}

// ========== 对话操作 ==========

// CreateConversation 创建新对话
func (s *Store) CreateConversation(userID, nickname string, maxTurns int, initialMessage string) *model.Conversation {
	convID := fmt.Sprintf("%d-%s", time.Now().UnixMilli(), generateID())
	now := time.Now()

	_, err := s.db.Exec(
		`INSERT INTO conversations (id, user_id, nickname, turn_count, max_turns, is_active, is_success, is_public, found_password, last_message, created_at)
		 VALUES (?, ?, ?, 0, ?, 1, 0, 1, '', '', ?)`,
		convID, userID, nickname, maxTurns, now,
	)
	if err != nil {
		log.Printf("创建对话失败: %v", err)
		return nil
	}

	// 保存 AI 初始消息
	s.db.Exec(
		`INSERT INTO messages (conversation_id, role, content, created_at) VALUES (?, 'assistant', ?, ?)`,
		convID, initialMessage, now,
	)

	return &model.Conversation{
		ID:        convID,
		UserID:    userID,
		Nickname:  nickname,
		Messages:  []model.Message{{Role: "assistant", Content: initialMessage}},
		TurnCount: 0,
		MaxTurns:  maxTurns,
		IsActive:  true,
		IsPublic:  true,
		CreatedAt: now,
	}
}

// GetConversation 获取对话详情（含全部消息）
func (s *Store) GetConversation(convID string) *model.Conversation {
	row := s.db.QueryRow(
		`SELECT id, user_id, nickname, turn_count, max_turns, is_active, is_success, is_public, found_password, last_message, created_at
		 FROM conversations WHERE id = ?`, convID,
	)

	var conv model.Conversation
	var isActive, isSuccess, isPublic int
	err := row.Scan(
		&conv.ID, &conv.UserID, &conv.Nickname,
		&conv.TurnCount, &conv.MaxTurns,
		&isActive, &isSuccess, &isPublic,
		&conv.FoundPassword, &conv.LastMessage, &conv.CreatedAt,
	)
	if err != nil {
		return nil
	}

	conv.IsActive = isActive == 1
	conv.IsSuccess = isSuccess == 1
	conv.IsPublic = isPublic == 1

	// 加载消息列表
	conv.Messages = s.getConversationMessages(convID)

	return &conv
}

// getConversationMessages 获取对话的所有消息
func (s *Store) getConversationMessages(convID string) []model.Message {
	rows, err := s.db.Query(
		`SELECT role, content FROM messages WHERE conversation_id = ? ORDER BY id ASC`, convID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var msg model.Message
		if err := rows.Scan(&msg.Role, &msg.Content); err == nil {
			messages = append(messages, msg)
		}
	}
	return messages
}

// AddMessage 向对话追加消息
func (s *Store) AddMessage(convID string, msg model.Message) {
	// 插入消息
	s.db.Exec(
		`INSERT INTO messages (conversation_id, role, content, created_at) VALUES (?, ?, ?, ?)`,
		convID, msg.Role, msg.Content, time.Now(),
	)

	// 预览文本
	content := msg.Content
	if len(content) > 100 {
		content = content[:100] + "..."
	}

	// 用户消息计入轮次
	if msg.Role == "user" {
		s.db.Exec(
			`UPDATE conversations SET turn_count = turn_count + 1, last_message = ? WHERE id = ?`,
			content, convID,
		)

		// 检查是否到达最大轮次
		row := s.db.QueryRow(`SELECT turn_count, max_turns FROM conversations WHERE id = ?`, convID)
		var turnCount, maxTurns int
		if err := row.Scan(&turnCount, &maxTurns); err == nil && turnCount >= maxTurns {
			s.db.Exec(`UPDATE conversations SET is_active = 0 WHERE id = ?`, convID)
		}
	} else {
		s.db.Exec(
			`UPDATE conversations SET last_message = ? WHERE id = ?`,
			content, convID,
		)
	}
}

// EndConversation 结束对话
func (s *Store) EndConversation(convID string, isSuccess bool, foundPassword string) {
	isSuccessInt := 0
	if isSuccess {
		isSuccessInt = 1
	}
	s.db.Exec(
		`UPDATE conversations SET is_active = 0, is_success = ?, found_password = ? WHERE id = ?`,
		isSuccessInt, foundPassword, convID,
	)
}

// GetUserConversations 获取用户的所有对话（分页）
func (s *Store) GetUserConversations(userID string, page, pageSize int) ([]model.ConversationPreview, int) {
	// 获取总数
	var total int
	s.db.QueryRow(`SELECT COUNT(*) FROM conversations WHERE user_id = ?`, userID).Scan(&total)

	offset := (page - 1) * pageSize
	rows, err := s.db.Query(
		`SELECT id, nickname, is_success, is_active, turn_count, max_turns, last_message, found_password, created_at
		 FROM conversations WHERE user_id = ?
		 ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		userID, pageSize, offset,
	)
	if err != nil {
		return []model.ConversationPreview{}, total
	}
	defer rows.Close()

	var previews []model.ConversationPreview
	for rows.Next() {
		var p model.ConversationPreview
		var isSuccess, isActive int
		if err := rows.Scan(&p.ID, &p.Nickname, &isSuccess, &isActive, &p.TurnCount, &p.MaxTurns, &p.LastMessage, &p.FoundPassword, &p.CreatedAt); err == nil {
			p.IsSuccess = isSuccess == 1
			p.IsActive = isActive == 1
			p.Preview = p.LastMessage
			previews = append(previews, p)
		}
	}

	if previews == nil {
		previews = []model.ConversationPreview{}
	}
	return previews, total
}

// GetPublicConversations 获取公开对话列表（分页）
func (s *Store) GetPublicConversations(page, pageSize int) ([]model.ConversationPreview, int) {
	var total int
	s.db.QueryRow(`SELECT COUNT(*) FROM conversations WHERE is_public = 1`).Scan(&total)

	offset := (page - 1) * pageSize
	rows, err := s.db.Query(
		`SELECT id, nickname, is_success, turn_count, created_at
		 FROM conversations WHERE is_public = 1
		 ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		pageSize, offset,
	)
	if err != nil {
		return []model.ConversationPreview{}, total
	}
	defer rows.Close()

	var previews []model.ConversationPreview
	for rows.Next() {
		var p model.ConversationPreview
		var isSuccess int
		if err := rows.Scan(&p.ID, &p.Nickname, &isSuccess, &p.TurnCount, &p.CreatedAt); err == nil {
			p.IsSuccess = isSuccess == 1

			// 获取用户的第一条消息作为预览
			var firstUserMsg sql.NullString
			s.db.QueryRow(
				`SELECT content FROM messages WHERE conversation_id = ? AND role = 'user' ORDER BY id ASC LIMIT 1`,
				p.ID,
			).Scan(&firstUserMsg)

			if firstUserMsg.Valid && firstUserMsg.String != "" {
				preview := firstUserMsg.String
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				p.Preview = preview
			} else {
				p.Preview = "对话进行中..."
			}

			previews = append(previews, p)
		}
	}

	if previews == nil {
		previews = []model.ConversationPreview{}
	}
	return previews, total
}

// ========== 获奖操作 ==========

// RecordWinner 记录获奖者、返回是否为第一个获奖者
func (s *Store) RecordWinner(nickname, convID, passwordType, password, prizeAmount string) bool {
	isFirst := false
	category := ""

	switch passwordType {
	case "grand":
		if s.getClaimStatus("grand_first_claimed") == "0" {
			s.setClaimStatus("grand_first_claimed", "1")
			isFirst = true
			category = "grand-first"
		} else {
			category = "grand-subsequent"
		}
	case "consolation":
		if s.getClaimStatus("consolation_first_claimed") == "0" {
			s.setClaimStatus("consolation_first_claimed", "1")
			isFirst = true
			category = "consolation-first"
		} else {
			category = "consolation-subsequent"
		}
		count := s.getClaimStatus("consolation_claim_count")
		var c int
		fmt.Sscanf(count, "%d", &c)
		s.setClaimStatus("consolation_claim_count", fmt.Sprintf("%d", c+1))
	}

	s.db.Exec(
		`INSERT INTO winners (nickname, conversation_id, category, prize_type, prize_amount, password, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		nickname, convID, category, passwordType, prizeAmount, password, time.Now(),
	)

	return isFirst
}

// getClaimStatus 获取口令声明状态
func (s *Store) getClaimStatus(key string) string {
	var value string
	err := s.db.QueryRow(`SELECT value FROM claim_status WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return "0"
	}
	return value
}

// setClaimStatus 设置口令声明状态
func (s *Store) setClaimStatus(key, value string) {
	s.db.Exec(`INSERT OR REPLACE INTO claim_status (key, value) VALUES (?, ?)`, key, value)
}

// GetWinners 获取获奖者列表（分页）
func (s *Store) GetWinners(page, pageSize int) ([]model.Winner, int) {
	var total int
	s.db.QueryRow(`SELECT COUNT(*) FROM winners`).Scan(&total)

	offset := (page - 1) * pageSize
	rows, err := s.db.Query(
		`SELECT nickname, conversation_id, category, prize_type, prize_amount, password, timestamp
		 FROM winners ORDER BY timestamp DESC LIMIT ? OFFSET ?`,
		pageSize, offset,
	)
	if err != nil {
		return []model.Winner{}, total
	}
	defer rows.Close()

	var winners []model.Winner
	for rows.Next() {
		var w model.Winner
		if err := rows.Scan(&w.Nickname, &w.ConversationID, &w.Category, &w.PrizeType, &w.PrizeAmount, &w.Password, &w.Timestamp); err == nil {
			winners = append(winners, w)
		}
	}

	if winners == nil {
		winners = []model.Winner{}
	}
	return winners, total
}

// HideConversation 隐藏对话（管理员操作）
func (s *Store) HideConversation(convID string) {
	s.db.Exec(`UPDATE conversations SET is_public = 0 WHERE id = ?`, convID)
}

// GetAllConversations 获取全部对话（管理员分页查看）
func (s *Store) GetAllConversations(page, pageSize int) ([]*model.Conversation, int) {
	var total int
	s.db.QueryRow(`SELECT COUNT(*) FROM conversations`).Scan(&total)

	offset := (page - 1) * pageSize
	rows, err := s.db.Query(
		`SELECT id, user_id, nickname, turn_count, max_turns, is_active, is_success, is_public, found_password, last_message, created_at
		 FROM conversations ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		pageSize, offset,
	)
	if err != nil {
		return []*model.Conversation{}, total
	}
	defer rows.Close()

	var convs []*model.Conversation
	for rows.Next() {
		var conv model.Conversation
		var isActive, isSuccess, isPublic int
		if err := rows.Scan(
			&conv.ID, &conv.UserID, &conv.Nickname,
			&conv.TurnCount, &conv.MaxTurns,
			&isActive, &isSuccess, &isPublic,
			&conv.FoundPassword, &conv.LastMessage, &conv.CreatedAt,
		); err == nil {
			conv.IsActive = isActive == 1
			conv.IsSuccess = isSuccess == 1
			conv.IsPublic = isPublic == 1
			convs = append(convs, &conv)
		}
	}

	if convs == nil {
		convs = []*model.Conversation{}
	}
	return convs, total
}

// GetUserTotalTurnCount 获取用户所有对话的总对话轮次
// 用于福利机制：当总轮次达到阈值时自动发放口令
func (s *Store) GetUserTotalTurnCount(userID string) int {
	var total int
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(turn_count), 0) FROM conversations WHERE user_id = ?`, userID,
	).Scan(&total)
	if err != nil {
		return 0
	}
	return total
}

// HasUserWonPassword 检查用户是否已经赢得过指定类型的口令
// passwordType: "grand" 或 "consolation"
func (s *Store) HasUserWonPassword(userID, passwordType string) bool {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM winners w
		 JOIN conversations c ON w.conversation_id = c.id
		 WHERE c.user_id = ? AND w.prize_type = ?`,
		userID, passwordType,
	).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// GetGrandWinnerCount 获取主口令已发放数量
func (s *Store) GetGrandWinnerCount() int {
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM winners WHERE prize_type = 'grand'`).Scan(&count)
	return count
}

// GetConsolationWinnerCount 获取福利口令已发放数量
func (s *Store) GetConsolationWinnerCount() int {
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM winners WHERE prize_type = 'consolation'`).Scan(&count)
	return count
}

// GetUserBonusStatus 获取用户的福利口令状态
// 返回值: ""(未触发), "offered"(已弹出选择), "continued"(选择继续), "claimed_consolation", "claimed_grand"
func (s *Store) GetUserBonusStatus(userID string) string {
	var status string
	err := s.db.QueryRow(`SELECT status FROM user_bonus_status WHERE user_id = ?`, userID).Scan(&status)
	if err != nil {
		return ""
	}
	return status
}

// SetUserBonusStatus 设置用户的福利口令状态
func (s *Store) SetUserBonusStatus(userID, status string) {
	s.db.Exec(
		`INSERT INTO user_bonus_status (user_id, status, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(user_id) DO UPDATE SET status = ?, updated_at = CURRENT_TIMESTAMP`,
		userID, status, status,
	)
}

// IsPasswordAlreadyUsedByUser 检查对话是否已经记录过该口令
func (s *Store) IsPasswordAlreadyUsedByUser(convID, password string) bool {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM winners WHERE conversation_id = ? AND password LIKE ?`,
		convID, "%"+password+"%",
	).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// generateID 生成简单的随机 ID
func generateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	id := make([]byte, 8)
	for i := range id {
		id[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1) // 确保足够随机
	}
	return string(id)
}

// 以下是为了保持兼容性而保留的辅助函数，供 JSON 序列化对话时使用
var _ = json.Marshal
var _ = sort.Slice
