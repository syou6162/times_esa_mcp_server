package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func submitDailyReport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// パラメーターの取得
	text, ok := request.Params.Arguments["text"].(string)
	if !ok {
		return nil, errors.New("text must be a string")
	}

	// #times-esa除去（prefix自体と直後の空白のみ除去、他は一切変更しない）
	text = stripPrefix(text, "#times-esa")

	// debounceチェック - 同じテキストが短時間内に複数回送信されたら拒否
	if isDebounced(text) {
		return nil, errors.New("10秒以内に同じ内容の投稿が行われました。しばらく待ってから再試行してください")
	}

	// esa.ioの設定を取得
	esaConfig := getEsaConfig()
	if esaConfig.TeamName == "" || esaConfig.AccessToken == "" {
		return nil, errors.New("ESA_TEAM_NAME または ESA_ACCESS_TOKEN が設定されていません")
	}

	client := createHTTPClient()

	// 日付ベースのカテゴリを生成
	now := time.Now()
	category := fmt.Sprintf("日報/%04d/%02d/%02d", now.Year(), now.Month(), now.Day())

	// 既存の投稿を検索
	existingPost, err := searchPostByCategory(client, esaConfig, category)
	if err != nil {
		return nil, fmt.Errorf("投稿の検索に失敗しました: %w", err)
	}

	var post *EsaPost
	if existingPost == nil {
		// 新しい投稿を作成
		post, err = createPost(client, esaConfig, text)
		if err != nil {
			return nil, fmt.Errorf("新規投稿の作成に失敗しました: %w", err)
		}
	} else {
		// 既存の投稿を更新（テキストのみ）
		post, err = updatePost(client, esaConfig, existingPost, text)
		if err != nil {
			return nil, fmt.Errorf("投稿の更新に失敗しました: %w", err)
		}
	}

	// レスポンスの作成
	response := DailyReportResponse{
		Success: true,
		Message: "日報を投稿しました",
		Post:    *post,
	}

	// JSONに変換してレスポンスを返す
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("レスポンスのJSON変換に失敗: %w", err)
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
