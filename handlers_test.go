package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubmitDailyReport(t *testing.T) {
	// テスト用の現在時刻を固定
	fixedTime := time.Date(2025, 5, 3, 13, 0, 0, 0, time.Local)

	t.Run("新規投稿テスト", func(t *testing.T) {
		// 各テストケース前にdebounceをリセット
		resetDebounce()

		// モックの作成
		mockEsaClient := NewMockEsaClientInterface(t)

		// テスト用データ
		testText := "テスト投稿内容"
		mockPost := &EsaPost{
			Number: 123,
			Name:   "テスト日報",
			BodyMd: "13:00 テスト投稿内容\n\n---",
		}

		// モックの振る舞いを設定
		mockEsaClient.EXPECT().SearchPostByCategory("日報/2025/05/03").Return(nil, nil)
		mockEsaClient.EXPECT().CreatePost(testText).Return(mockPost, nil)

		// リクエスト作成
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"text": testText,
		}
		// テスト対象の関数を実行
		result, err := submitDailyReportWithTime(context.TODO(), req, mockEsaClient, fixedTime)

		// 検証
		require.NoError(t, err, "submitDailyReport should not return an error")
		require.NotNil(t, result, "submitDailyReport should return a result")
		require.Len(t, result.Content, 1, "Result should contain one content item")
		require.IsType(t, mcp.TextContent{}, result.Content[0], "Content item should be TextContent")

		// レスポンスのJSONをパースして内容を検証
		textContent := result.Content[0].(mcp.TextContent).Text
		var response DailyReportResponse
		err = json.Unmarshal([]byte(textContent), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Contains(t, response.Message, "日報を投稿しました")
	})

	t.Run("既存投稿更新テスト", func(t *testing.T) {
		// 各テストケース前にdebounceをリセット
		resetDebounce()

		// モックの作成
		mockEsaClient := NewMockEsaClientInterface(t)

		// テスト用データ
		testText := "テスト追記内容"
		existingPost := &EsaPost{
			Number: 123,
			Name:   "テスト日報",
			BodyMd: "10:00 既存の内容\n\n---",
		}
		updatedPost := &EsaPost{
			Number: 123,
			Name:   "テスト日報",
			BodyMd: "13:00 テスト追記内容\n\n---\n\n10:00 既存の内容\n\n---",
		}

		// モックの振る舞いを設定
		mockEsaClient.EXPECT().SearchPostByCategory("日報/2025/05/03").Return(existingPost, nil)
		mockEsaClient.EXPECT().UpdatePost(existingPost, testText).Return(updatedPost, nil)

		// リクエスト作成
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"text": testText,
		}

		// テスト対象の関数を実行
		result, err := submitDailyReportWithTime(context.TODO(), req, mockEsaClient, fixedTime)

		// 検証
		require.NoError(t, err, "submitDailyReport should not return an error")
		require.NotNil(t, result, "submitDailyReport should return a result")
		require.Len(t, result.Content, 1, "Result should contain one content item")
		require.IsType(t, mcp.TextContent{}, result.Content[0], "Content item should be TextContent")

		textContent := result.Content[0].(mcp.TextContent).Text
		var response DailyReportResponse
		err = json.Unmarshal([]byte(textContent), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
		assert.Contains(t, response.Message, "日報を投稿しました")
	})

	t.Run("検索エラーテスト", func(t *testing.T) {
		// 各テストケース前にdebounceをリセット
		resetDebounce()

		// モックの作成
		mockEsaClient := NewMockEsaClientInterface(t)

		// モックの振る舞いを設定
		mockEsaClient.EXPECT().SearchPostByCategory("日報/2025/05/03").Return(nil, errors.New("API接続エラー"))

		// リクエスト作成
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"text": "テスト内容",
		}
		// テスト対象の関数を実行
		_, err := submitDailyReportWithTime(context.TODO(), req, mockEsaClient, fixedTime)

		// エラーが返ることを検証
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "投稿の検索に失敗")
	})

	t.Run("プレフィックス除去テスト", func(t *testing.T) {
		// 各テストケース前にdebounceをリセット
		resetDebounce()

		// モックの作成
		mockEsaClient := NewMockEsaClientInterface(t)

		// テスト用データ
		inputText := "#times-esa テスト投稿内容" // #times-esaプレフィックス付き
		expectedText := "テスト投稿内容"         // プレフィックスが除去された状態
		mockPost := &EsaPost{
			Number: 123,
			Name:   "テスト日報",
			BodyMd: "13:00 テスト投稿内容\n\n---",
		}

		// モックの振る舞いを設定
		mockEsaClient.EXPECT().SearchPostByCategory("日報/2025/05/03").Return(nil, nil)
		mockEsaClient.EXPECT().CreatePost(expectedText).Return(mockPost, nil)

		// リクエスト作成
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"text": inputText,
		}

		// テスト対象の関数を実行
		result, err := submitDailyReportWithTime(context.TODO(), req, mockEsaClient, fixedTime)

		// 検証
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Content, 1)
	})

	t.Run("引数不正テスト", func(t *testing.T) {
		// 各テストケース前にdebounceをリセット
		resetDebounce()

		// モックの作成
		mockEsaClient := NewMockEsaClientInterface(t)

		// text引数が文字列でない場合
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"text": 123, // 文字列ではなく数値
		}

		// テスト対象の関数を実行
		_, err := submitDailyReportWithTime(context.TODO(), req, mockEsaClient, fixedTime)

		// エラーが返ることを検証
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "text must be a string")
	})
}
