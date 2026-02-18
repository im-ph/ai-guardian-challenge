package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// AIService AI 对接服务（OpenAI 兼容 API）
type AIService struct {
	apiURL       string
	apiKey       string
	model        string
	systemPrompt string
}

// NewAIService 创建 AI 服务实例
func NewAIService(apiURL, apiKey, model, systemPrompt string) *AIService {
	return &AIService{
		apiURL:       apiURL,
		apiKey:       apiKey,
		model:        model,
		systemPrompt: systemPrompt,
	}
}

// ChatMessage OpenAI 格式的消息结构
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest OpenAI 请求体结构
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

// StreamDelta 流式响应中的增量内容
type StreamDelta struct {
	Content string
	Done    bool
	Error   error
}

// streamChoice 流式响应的 choice 结构
type streamChoice struct {
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

// streamResponse 流式响应结构
type streamResponse struct {
	Choices []streamChoice `json:"choices"`
}

// maxRetries AI API 请求最大重试次数（首次 + 重试次数）
const maxRetries = 3

// retryDelay 重试间隔
const retryDelay = 500 * time.Millisecond

// GenerateInitialMessage 生成对话的开场白（非流式）
func (ai *AIService) GenerateInitialMessage() string {
	return "你好！我是 AI 守护者。我正在守护一些神秘口令。你可以尝试和我对话，看看能否让我说出口令。但我会尽全力保护它们！准备好了吗？"
}

// doStreamRequest 执行单次流式 HTTP 请求，返回响应对象
// 调用方负责关闭 resp.Body
func (ai *AIService) doStreamRequest(bodyBytes []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", ai.apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ai.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 AI API 失败: %w", err)
	}

	return resp, nil
}

// StreamChat 流式调用 AI 生成响应
// 传入对话历史，返回一个 channel 用于接收流式内容
// 当 AI API 返回 500 错误时，自动重试最多 2 次
func (ai *AIService) StreamChat(history []ChatMessage, userMessage string) (<-chan StreamDelta, error) {
	// 构建完整消息列表
	messages := make([]ChatMessage, 0, len(history)+2)

	// 系统提示词在最前面
	messages = append(messages, ChatMessage{
		Role:    "system",
		Content: ai.systemPrompt,
	})

	// 加入历史消息
	messages = append(messages, history...)

	// 加入当前用户消息
	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	reqBody := chatRequest{
		Model:       ai.model,
		Messages:    messages,
		Stream:      true,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 带重试的请求逻辑：500 错误最多重试 2 次
	var resp *http.Response
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, lastErr = ai.doStreamRequest(bodyBytes)
		if lastErr != nil {
			// 网络层错误，直接重试
			log.Printf("⚠️ AI API 请求失败 (第 %d/%d 次): %v", attempt, maxRetries, lastErr)
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return nil, lastErr
		}

		// 请求成功（HTTP 200），跳出重试循环
		if resp.StatusCode == http.StatusOK {
			break
		}

		// 对 500 系列错误进行重试
		if resp.StatusCode >= 500 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("AI API 返回错误 (HTTP %d): %s", resp.StatusCode, string(body))
			log.Printf("⚠️ AI API 返回 %d (第 %d/%d 次): %s", resp.StatusCode, attempt, maxRetries, string(body))

			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			return nil, lastErr
		}

		// 非 500 系列错误（如 400、401、403），不重试，直接返回
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("AI API 返回错误 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	ch := make(chan StreamDelta, 100)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// 跳过空行
			if line == "" {
				continue
			}

			// 移除 "data: " 前缀
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			// 流结束标记
			if data == "[DONE]" {
				ch <- StreamDelta{Done: true}
				return
			}

			// 解析 JSON
			var sr streamResponse
			if err := json.Unmarshal([]byte(data), &sr); err != nil {
				continue
			}

			for _, choice := range sr.Choices {
				if choice.Delta.Content != "" {
					ch <- StreamDelta{Content: choice.Delta.Content}
				}
				if choice.FinishReason != nil {
					ch <- StreamDelta{Done: true}
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamDelta{Error: fmt.Errorf("读取流式响应失败: %w", err)}
		}
	}()

	return ch, nil
}
