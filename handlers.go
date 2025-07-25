package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultHandlerFactory は標準的なハンドラーを生成します
type DefaultHandlerFactory struct{}

// CreateEsaClient はesa.ioクライアントを生成します
func (f *DefaultHandlerFactory) CreateEsaClient() (EsaClientInterface, error) {
	config := ConfigFromEnv()
	if config.TeamName == "" || config.AccessToken == "" {
		return nil, errors.New("ESA_TEAM_NAME または ESA_ACCESS_TOKEN が設定されていません")
	}
	httpClient := NewHTTPClient(10 * time.Second)
	return NewEsaClient(httpClient, config), nil
}

// submitDailyReportWithClock は日報を投稿するハンドラー（時間指定可能、テスト用）
func submitDailyReportWithClock(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[TimesEsaPostRequest], esaClient EsaClientInterface, now time.Time) (*mcp.CallToolResultFor[TimesEsaPostResponse], error) {

	// パラメーターの取得
	text := params.Arguments.Text

	// 空文字チェック
	if text == "" {
		return nil, fmt.Errorf("text parameter cannot be empty")
	}

	// confirmed_by_userパラメータの確認
	confirmedByUser := params.Arguments.ConfirmedByUser

	// ユーザーによる確認が取れていない場合はエラーで停止
	if !confirmedByUser {
		return nil, fmt.Errorf("投稿前にユーザーによる内容の確認が必要です。内容の確認をユーザーに行ったら、confirmed_by_user=trueを設定してください")
	}

	// #times-esa除去（prefix自体と直後の空白のみ除去、他は一切変更しない）
	text = stripPrefix(text, "#times-esa")

	// debounceチェック - 同じテキストが短時間内に複数回送信されたら拒否
	if isDebounced(text) {
		// デバウンス時間を秒単位でメッセージに含める
		debounceSeconds := int(debounceConfig.Duration.Seconds())
		return nil, fmt.Errorf("%d秒以内に同じ内容の投稿が行われました。しばらく待ってから再試行してください", debounceSeconds)
	}

	// 日付ベースのカテゴリを生成
	category := fmt.Sprintf("日報/%04d/%02d/%02d", now.Year(), now.Month(), now.Day())

	// 既存の投稿を検索
	existingPost, err := esaClient.SearchPostByCategory(category)
	if err != nil {
		return nil, fmt.Errorf("投稿の検索に失敗しました: %w", err)
	}

	var post *EsaPost
	if existingPost == nil {
		// 新しい投稿を作成
		post, err = esaClient.CreatePost(text)
		if err != nil {
			return nil, fmt.Errorf("新規投稿の作成に失敗しました: %w", err)
		}
	} else {
		// 既存の投稿を更新（テキストのみ）
		post, err = esaClient.UpdatePost(existingPost, text)
		if err != nil {
			return nil, fmt.Errorf("投稿の更新に失敗しました: %w", err)
		}
	}

	// レスポンスを返す
	return &mcp.CallToolResultFor[TimesEsaPostResponse]{
		StructuredContent: TimesEsaPostResponse{
			Success: true,
			Message: "日報を投稿しました",
			Post:    *post,
		},
	}, nil
}

// submitDailyReportHandler は日報を投稿するハンドラー
func submitDailyReportHandler(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[TimesEsaPostRequest]) (*mcp.CallToolResultFor[TimesEsaPostResponse], error) {
	factory := &DefaultHandlerFactory{}
	esaClient, err := factory.CreateEsaClient()
	if err != nil {
		return nil, err
	}
	return submitDailyReportWithClock(ctx, ss, params, esaClient, time.Now())
}
