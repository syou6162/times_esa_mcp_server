package main

import (
	"context"
	"errors"
	"testing"
	"time"

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
			BodyMd: "<a id=\"1300\" href=\"#1300\">13:00</a> テスト投稿内容\n\n---",
		}

		// モックの振る舞いを設定
		mockEsaClient.EXPECT().SearchPostByCategory("日報/2025/05/03").Return(nil, nil)
		mockEsaClient.EXPECT().CreatePost(testText).Return(mockPost, nil)

		// リクエスト作成
		params := &mcp.CallToolParamsFor[TimesEsaPostRequest]{
			Arguments: TimesEsaPostRequest{
				Text:            testText,
				ConfirmedByUser: true,
			},
		}

		// テスト対象の関数を実行
		result, err := submitDailyReportWithClock(context.TODO(), nil, params, mockEsaClient, fixedTime)

		// 検証
		require.NoError(t, err, "submitDailyReport should not return an error")
		require.NotNil(t, result, "submitDailyReport should return a result")
		assert.True(t, result.Success)
		assert.Contains(t, result.Message, "日報を投稿しました")
		assert.Equal(t, mockPost.Number, result.Post.Number)
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
			BodyMd: "<a id=\"1000\" href=\"#1000\">10:00</a> 既存の内容\n\n---",
		}
		updatedPost := &EsaPost{
			Number: 123,
			Name:   "テスト日報",
			BodyMd: "<a id=\"1300\" href=\"#1300\">13:00</a> テスト追記内容\n\n---\n\n<a id=\"1000\" href=\"#1000\">10:00</a> 既存の内容\n\n---",
		}

		// モックの振る舞いを設定
		mockEsaClient.EXPECT().SearchPostByCategory("日報/2025/05/03").Return(existingPost, nil)
		mockEsaClient.EXPECT().UpdatePost(existingPost, testText).Return(updatedPost, nil)

		// リクエスト作成
		params := &mcp.CallToolParamsFor[TimesEsaPostRequest]{
			Arguments: TimesEsaPostRequest{
				Text:            testText,
				ConfirmedByUser: true,
			},
		}

		// テスト対象の関数を実行
		result, err := submitDailyReportWithClock(context.TODO(), nil, params, mockEsaClient, fixedTime)

		// 検証
		require.NoError(t, err, "submitDailyReport should not return an error")
		require.NotNil(t, result, "submitDailyReport should return a result")
		assert.True(t, result.Success)
		assert.Contains(t, result.Message, "日報を投稿しました")
	})

	t.Run("検索エラーテスト", func(t *testing.T) {
		// 各テストケース前にdebounceをリセット
		resetDebounce()

		// モックの作成
		mockEsaClient := NewMockEsaClientInterface(t)

		// モックの振る舞いを設定
		mockEsaClient.EXPECT().SearchPostByCategory("日報/2025/05/03").Return(nil, errors.New("API接続エラー"))

		// リクエスト作成
		params := &mcp.CallToolParamsFor[TimesEsaPostRequest]{
			Arguments: TimesEsaPostRequest{
				Text:            "テスト内容",
				ConfirmedByUser: true,
			},
		}

		// テスト対象の関数を実行
		_, err := submitDailyReportWithClock(context.TODO(), nil, params, mockEsaClient, fixedTime)

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
			BodyMd: "<a id=\"1300\" href=\"#1300\">13:00</a> テスト投稿内容\n\n---",
		}

		// モックの振る舞いを設定
		mockEsaClient.EXPECT().SearchPostByCategory("日報/2025/05/03").Return(nil, nil)
		mockEsaClient.EXPECT().CreatePost(expectedText).Return(mockPost, nil)

		// リクエスト作成
		params := &mcp.CallToolParamsFor[TimesEsaPostRequest]{
			Arguments: TimesEsaPostRequest{
				Text:            inputText,
				ConfirmedByUser: true,
			},
		}

		// テスト対象の関数を実行
		result, err := submitDailyReportWithClock(context.TODO(), nil, params, mockEsaClient, fixedTime)

		// 検証
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success)
	})

	t.Run("confirmed_by_user=falseの場合のエラーテスト", func(t *testing.T) {
		// 各テストケース前にdebounceをリセット
		resetDebounce()

		// モックの作成
		mockEsaClient := NewMockEsaClientInterface(t)

		// confirmed_by_user=falseでリクエスト作成
		params := &mcp.CallToolParamsFor[TimesEsaPostRequest]{
			Arguments: TimesEsaPostRequest{
				Text:            "テスト内容",
				ConfirmedByUser: false,
			},
		}

		// テスト対象の関数を実行
		_, err := submitDailyReportWithClock(context.TODO(), nil, params, mockEsaClient, fixedTime)

		// エラーが返ることを検証
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "投稿前にユーザーによる内容の確認が必要です")
		assert.Contains(t, err.Error(), "confirmed_by_user=trueを設定してください")
	})

	t.Run("空文字テスト", func(t *testing.T) {
		// 各テストケース前にdebounceをリセット
		resetDebounce()

		// モックの作成
		mockEsaClient := NewMockEsaClientInterface(t)

		// リクエスト作成（空文字を送信）
		params := &mcp.CallToolParamsFor[TimesEsaPostRequest]{
			Arguments: TimesEsaPostRequest{
				Text:            "",
				ConfirmedByUser: true,
			},
		}

		// テスト対象の関数を実行
		_, err := submitDailyReportWithClock(context.TODO(), nil, params, mockEsaClient, fixedTime)

		// エラーが返ることを検証
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
}
