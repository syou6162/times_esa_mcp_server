package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// quickPostPromptHandler はquick-postプロンプトを処理します
func quickPostPromptHandler(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	// 引数からテキストを取得
	text, exists := request.Params.Arguments["text"]
	if !exists || text == "" {
		text = "内容を入力してください"
	}

	// プロンプトメッセージを作成
	promptText := fmt.Sprintf("以下の内容をtimes_esaに投稿してください：\n\n%s", text)
	
	messages := []mcp.PromptMessage{
		{
			Role: mcp.RoleUser,
			Content: mcp.TextContent{
				Type: "text",
				Text: promptText,
			},
		},
	}

	return mcp.NewGetPromptResult(
		PromptDescriptionQuickPost,
		messages,
	), nil
}