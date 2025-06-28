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
	timesEsaTool := mcp.NewServerTool[TimesEsaPostRequest, TimesEsaPostResponse](
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

	if err := s.Run(context.Background(), mcp.NewStdioTransport()); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
