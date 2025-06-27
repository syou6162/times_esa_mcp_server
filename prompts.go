package main

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// プロンプトテンプレート関連の定数
const (
	// プロンプト名
	PromptNameQuickPost = "quick-post"
	
	// プロンプト説明
	PromptDescriptionQuickPost = "times_esaへの素早い投稿"
	
	// 引数説明
	ArgumentDescriptionText = "投稿するテキスト内容"
)

// quickPostPromptHandler はquick-postプロンプトを処理します
func quickPostPromptHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.GetPromptParamsFor[QuickPostArgs]) (*mcp.GetPromptResultFor[QuickPostResult], error) {
	// 引数からテキストを取得
	text := params.Arguments.Text
	if text == "" {
		text = "内容を入力してください"
	}

	// プロンプトメッセージを作成
	promptText := fmt.Sprintf("以下の内容をtimes_esaに投稿してください：\n\n%s", text)
	
	messages := []PromptMessage{
		{
			Role: "user",
			Content: TextContent{
				Type: "text",
				Text: promptText,
			},
		},
	}

	result := &QuickPostResult{
		Description: PromptDescriptionQuickPost,
		Messages:    messages,
	}

	return &mcp.GetPromptResultFor[QuickPostResult]{
		Description: result.Description,
		Messages:    convertToMCPMessages(result.Messages),
	}, nil
}

func convertToMCPMessages(messages []PromptMessage) []mcp.PromptMessage {
	mcpMessages := make([]mcp.PromptMessage, len(messages))
	for i, msg := range messages {
		mcpMessages[i] = mcp.PromptMessage{
			Role: mcp.Role(msg.Role),
			Content: mcp.TextContent{
				Type: msg.Content.Type,
				Text: msg.Content.Text,
			},
		}
	}
	return mcpMessages
}
