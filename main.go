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
	timesEsaTool := mcp.NewServerTool[PostDailyReportArgs, PostDailyReportResult](
		"times-esa",
		"times-esaに日報を投稿します",
		submitDailyReportHandler,
		mcp.Input(
			mcp.Property("text",
				mcp.Description("投稿するテキスト内容"),
				mcp.Required(true),
			),
		),
	)

	// ツールの登録
	s.AddTools(timesEsaTool)

	// プロンプトテンプレートの定義と登録
	// TODO: プロンプト関連を新しいAPIに移行
	// quickPostPrompt := mcp.NewPrompt(PromptNameQuickPost,
	// 	mcp.WithPromptDescription(PromptDescriptionQuickPost),
	// 	mcp.WithArgument("text",
	// 		mcp.ArgumentDescription(ArgumentDescriptionText),
	// 		mcp.RequiredArgument(),
	// 	),
	// )
	// s.AddPrompt(quickPostPrompt, quickPostPromptHandler)

	if err := s.Run(context.Background(), mcp.NewStdioTransport()); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
