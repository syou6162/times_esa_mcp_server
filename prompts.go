package main

// import (
// 	"context"
// 	"fmt"
//
// 	"github.com/modelcontextprotocol/go-sdk/mcp"
// )

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
// TODO: プロンプトAPIに移行
/*
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
*/
