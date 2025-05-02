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

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
