package main

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	s := mcp.NewServer("times-esa-mcp-server", "1.0.0", nil)

	// times-esaツールの定義（日報投稿用）
	timesEsaTool := mcp.NewServerTool[PostDailyReportArgs, PostDailyReportResult](
		"times-esa",
		"times-esaに日報を投稿します",
		submitDailyReportHandler,
	)

	s.AddTools(timesEsaTool)

	// プロンプトテンプレートの定義と登録
	quickPostPrompt := mcp.NewServerPrompt[QuickPostArgs](
		PromptNameQuickPost,
		PromptDescriptionQuickPost,
		quickPostPromptHandler,
	)
	s.AddPrompts(quickPostPrompt)

	if err := s.Serve(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
