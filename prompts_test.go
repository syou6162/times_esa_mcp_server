package main

// TODO: プロンプト関連のテストを実装
/*
import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

func TestQuickPostPromptHandler(t *testing.T) {
	tests := []struct {
		name        string
		arguments   map[string]string
		expectedMsg string
	}{
		{
			name: "正常な引数",
			arguments: map[string]string{
				"text": "今日の作業完了",
			},
			expectedMsg: "以下の内容をtimes_esaに投稿してください：\n\n今日の作業完了",
		},
		{
			name:        "引数なし",
			arguments:   map[string]string{},
			expectedMsg: "以下の内容をtimes_esaに投稿してください：\n\n内容を入力してください",
		},
		{
			name: "空の引数",
			arguments: map[string]string{
				"text": "",
			},
			expectedMsg: "以下の内容をtimes_esaに投稿してください：\n\n内容を入力してください",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			request := mcp.GetPromptRequest{
				Params: struct {
					Name      string            `json:"name"`
					Arguments map[string]string `json:"arguments,omitempty"`
				}{
					Name:      PromptNameQuickPost,
					Arguments: tt.arguments,
				},
			}

			result, err := quickPostPromptHandler(ctx, request)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, PromptDescriptionQuickPost, result.Description)
			assert.Len(t, result.Messages, 1)
			assert.Equal(t, mcp.RoleUser, result.Messages[0].Role)

			// TextContentの内容を確認
			textContent, ok := result.Messages[0].Content.(mcp.TextContent)
			assert.True(t, ok, "Content should be TextContent")
			assert.Equal(t, "text", textContent.Type)
			assert.Equal(t, tt.expectedMsg, textContent.Text)
		})
	}
}
*/