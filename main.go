package main

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "times-esa-mcp-server",
			Version: "1.0.0",
		},
		nil,
	)

	// times-esaツールの定義（日報投稿用 - textパラメータとconfirmed_by_userパラメータ）
	timesEsaTool := mcp.NewServerTool[TimesEsaPostRequest, TimesEsaPostResponse](
		"times-esa",
		"times-esaに日報を投稿します",
		submitDailyReportHandler,
		mcp.Input(
			mcp.Property("text",
				mcp.Description("投稿するテキスト内容"),
				mcp.Required(true),
			),
			mcp.Property("confirmed_by_user",
				mcp.Description("ユーザーが投稿内容を確認したかどうか（true: 確認済みで投稿実行）"),
				mcp.Required(true),
			),
		),
	)

	// ツールの登録
	s.AddTool(timesEsaTool)

	if err := s.Run(context.Background(), mcp.NewStdioTransport()); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
