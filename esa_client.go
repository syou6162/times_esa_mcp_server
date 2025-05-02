package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

// getEsaConfig は環境変数からesa.ioの設定を取得する
func getEsaConfig() EsaConfig {
	teamName := os.Getenv("ESA_TEAM_NAME")
	accessToken := os.Getenv("ESA_ACCESS_TOKEN")
	return EsaConfig{
		TeamName:    teamName,
		AccessToken: accessToken,
	}
}

// createHTTPClient はHTTPクライアントを作成する
func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

// searchPostByCategory はカテゴリから投稿を検索する
func searchPostByCategory(client *http.Client, config EsaConfig, category string) (*EsaPost, error) {
	// 検索クエリの構築
	url := fmt.Sprintf("https://api.esa.io/v1/teams/%s/posts?q=category:%s", config.TeamName, category)

	// リクエストの作成
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+config.AccessToken)

	// リクエストの実行
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// レスポンスの解析
	if resp.StatusCode != http.StatusOK {
		var errorResp EsaErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("エラーレスポンスの解析に失敗: %w", err)
		}
		return nil, fmt.Errorf("%s: %s", errorResp.Error, errorResp.Message)
	}

	var searchResult EsaSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("検索結果の解析に失敗: %w", err)
	}

	// 検索結果の処理
	if searchResult.TotalCount == 0 {
		// 投稿が存在しない
		return nil, nil
	} else if searchResult.TotalCount > 1 {
		// 複数の投稿が存在する
		return nil, errors.New("複数の日報が存在します")
	}

	// 最新の投稿を返す
	return &searchResult.Posts[0], nil
}

// createPost は新しい投稿を作成する（デフォルト値設定を内部化）
func createPost(client *http.Client, config EsaConfig, text string) (*EsaPost, error) {
	// デフォルト値の設定
	now := time.Now()
	category := fmt.Sprintf("日報/%04d/%02d/%02d", now.Year(), now.Month(), now.Day())
	title := "日報"
	var tags []string

	url := fmt.Sprintf("https://api.esa.io/v1/teams/%s/posts", config.TeamName)

	// リクエストボディの作成
	type postRequest struct {
		Post struct {
			Name     string   `json:"name"`
			Category string   `json:"category"`
			Tags     []string `json:"tags"`
			BodyMd   string   `json:"body_md"`
			Wip      bool     `json:"wip"`
		} `json:"post"`
	}

	reqBody := postRequest{}
	reqBody.Post.Name = title
	reqBody.Post.Category = category
	reqBody.Post.Tags = tags

	// 現在時刻をhh:mm形式で取得し、テキストの前に追加、その後に区切り線を追加
	timePrefix := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
	reqBody.Post.BodyMd = fmt.Sprintf("%s %s\n\n---", timePrefix, text)

	reqBody.Post.Wip = false

	// JSONに変換
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("リクエストのJSON変換に失敗: %w", err)
	}

	// リクエストの作成
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+config.AccessToken)

	// リクエストの実行
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// レスポンスの解析
	if resp.StatusCode != http.StatusCreated {
		var errorResp EsaErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("エラーレスポンスの解析に失敗: %w", err)
		}
		return nil, fmt.Errorf("%s: %s", errorResp.Error, errorResp.Message)
	}

	var post EsaPost
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		return nil, fmt.Errorf("投稿の解析に失敗: %w", err)
	}

	return &post, nil
}

// updatePost は既存の投稿を更新する（テキストのみ追記）
func updatePost(client *http.Client, config EsaConfig, existingPost *EsaPost, text string) (*EsaPost, error) {
	url := fmt.Sprintf("https://api.esa.io/v1/teams/%s/posts/%d", config.TeamName, existingPost.Number)

	// リクエストボディの作成
	type patchRequest struct {
		Post struct {
			BodyMd string `json:"body_md"`
			Wip    bool   `json:"wip"`
		} `json:"post"`
	}

	reqBody := patchRequest{}

	// テキストを追記（新しいテキストを上に）
	if text != "" {
		// 現在時刻をhh:mm形式で取得
		now := time.Now()
		timePrefix := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())

		// 区切り線と時刻付きテキストを追記
		reqBody.Post.BodyMd = fmt.Sprintf("%s %s\n\n---\n\n%s", timePrefix, text, existingPost.BodyMd)
	} else {
		reqBody.Post.BodyMd = existingPost.BodyMd
	}
	reqBody.Post.Wip = false

	// JSONに変換
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("リクエストのJSON変換に失敗: %w", err)
	}

	// リクエストの作成
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+config.AccessToken)

	// リクエストの実行
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// レスポンスの解析
	if resp.StatusCode != http.StatusOK {
		var errorResp EsaErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("エラーレスポンスの解析に失敗: %w", err)
		}
		return nil, fmt.Errorf("%s: %s", errorResp.Error, errorResp.Message)
	}

	var post EsaPost
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		return nil, fmt.Errorf("投稿の解析に失敗: %w", err)
	}

	return &post, nil
}
