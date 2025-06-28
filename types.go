package main

import "fmt"

type PostDailyReportArgs struct {
	Text string `json:"text"`
}

type PostDailyReportResult struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Post    EsaPost `json:"post"`
}

type QuickPostArgs struct {
	Text string `json:"text"`
}

// 一時的な互換性のための型定義（後で削除予定）
type CallToolRequest map[string]interface{}

func (r CallToolRequest) RequireString(key string) (string, error) {
	v, ok := r[key]
	if !ok {
		return "", fmt.Errorf("required parameter %s not found", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s is not a string", key)
	}
	return s, nil
}

type CallToolResult struct {
	Content []Content `json:"content"`
}

type Content interface{}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func NewToolResultText(text string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			&TextContent{
				Type: "text",
				Text: text,
			},
		},
	}
}
