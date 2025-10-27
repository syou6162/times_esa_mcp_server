package main

import (
	"context"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
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

	// times-esaツールのスキーマ定義
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"text": {
				Type:        "string",
				Description: "投稿するテキスト内容",
			},
			"confirmed_by_user": {
				Type:        "boolean",
				Description: "ユーザーが投稿内容を確認したかどうか（true: 確認済みで投稿実行）",
			},
		},
		Required: []string{"text", "confirmed_by_user"},
	}

	// ツールの登録
	tool := &mcp.Tool{
		Name:        "times-esa",
		Description: "times-esaに日報を投稿します",
		InputSchema: schema,
	}
	mcp.AddTool(s, tool, submitDailyReportHandler)

	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
