package main

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer(
		"times-esa-mcp-server",
		"1.0.0",
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
	quickPostPrompt := mcp.NewPrompt("quick-post",
		mcp.WithPromptDescription("times_esaへの素早い投稿"),
		mcp.WithArgument("text", 
			mcp.ArgumentDescription("投稿するテキスト内容"),
			mcp.RequiredArgument(),
		),
	)
	s.AddPrompt(quickPostPrompt, quickPostPromptHandler)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
