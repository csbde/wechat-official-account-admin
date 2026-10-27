package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/database"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"github.com/anchel/wechat-official-account-admin/internal/model"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type AIService struct {
	http *resty.Client
}

var DefaultAIService = &AIService{
	http: resty.New(),
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// GenerateAIResponse 根据公众号的 AI 配置生成 ChatGPT 智能回复
func (s *AIService) GenerateAIResponse(ctx context.Context, appid, userMessage string) (string, error) {
	var setting model.WxAISetting
	if err := database.DB.Where("app_id = ? AND enabled = ?", appid, true).First(&setting).Error; err != nil {
		return "", fmt.Errorf("ai not enabled for appid: %s", appid)
	}

	baseURL := strings.TrimRight(setting.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	apiURL := baseURL + "/chat/completions"

	timeout := time.Duration(setting.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 4 * time.Second
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	systemPrompt := setting.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "你是该微信公众号的智能助手，请用简洁、亲切、准确的中文回答用户的微信咨询。"
	}

	chatReq := ChatCompletionRequest{
		Model: setting.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		Temperature: setting.Temperature,
		MaxTokens:   setting.MaxTokens,
	}

	if chatReq.Model == "" {
		chatReq.Model = "gpt-4o-mini"
	}

	client := s.http.R().
		SetContext(reqCtx).
		SetHeader("Content-Type", "application/json")

	if setting.APIKey != "" {
		client.SetHeader("Authorization", "Bearer "+setting.APIKey)
	}

	resp, err := client.SetBody(chatReq).Post(apiURL)
	if err != nil {
		logger.Warn("AI request failed", zap.String("appid", appid), zap.Error(err))
		return "", err
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(resp.Body(), &chatResp); err != nil {
		return "", fmt.Errorf("parse ai response error: %w", err)
	}

	if chatResp.Error != nil && chatResp.Error.Message != "" {
		return "", fmt.Errorf("ai api error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("ai empty response")
	}

	result := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	return result, nil
}

// TestAIConnection 连通性测试接口
func (s *AIService) TestAIConnection(ctx context.Context, baseURL, apiKey, modelName, prompt string) (string, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	apiURL := baseURL + "/chat/completions"

	client := s.http.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json")

	if apiKey != "" {
		client.SetHeader("Authorization", "Bearer "+apiKey)
	}

	if prompt == "" {
		prompt = "你好，这是一条连接测试消息，请回复'连接成功'并介绍一下你自己。"
	}

	if modelName == "" {
		modelName = "gpt-4o-mini"
	}

	chatReq := ChatCompletionRequest{
		Model: modelName,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
		MaxTokens:   200,
	}

	resp, err := client.SetBody(chatReq).Post(apiURL)
	if err != nil {
		return "", err
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(resp.Body(), &chatResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if chatResp.Error != nil && chatResp.Error.Message != "" {
		return "", fmt.Errorf("AI 接口返回错误: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("AI 未返回内容")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}
