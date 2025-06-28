package main

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	s := mcp.NewServer(
		"times-esa-mcp-server",
		"1.0.0",
		nil,
	)

	// times-esaツールの定義（日報投稿用 - textパラメータのみに簡略化）
	timesEsaTool := mcp.NewTool("times-esa",
		mcp.WithDescription("times-esaに日報を投稿します"),
		mcp.WithString("text",
			mcp.Required(),
			mcp.Description("投稿するテキスト内容"),
		),
	)

	// ツールの登録（後方互換性のあるラッパー関数を使用）
	s.AddTool(timesEsaTool, submitDailyReportLegacy)

	// プロンプトテンプレートの定義と登録
	quickPostPrompt := mcp.NewPrompt(PromptNameQuickPost,
		mcp.WithPromptDescription(PromptDescriptionQuickPost),
		mcp.WithArgument("text",
			mcp.ArgumentDescription(ArgumentDescriptionText),
			mcp.RequiredArgument(),
		),
	)
	s.AddPrompt(quickPostPrompt, quickPostPromptHandler)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
