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

// EsaClientInterface はesa.ioとの通信を担当するインターフェース
type EsaClientInterface interface {
	SearchPostByCategory(category string) (*EsaPost, error)
	CreatePost(text string) (*EsaPost, error)
	UpdatePost(existingPost *EsaPost, text string) (*EsaPost, error)
}

// HTTPClientInterface はHTTPクライアントの操作をモック可能にするインターフェース
type HTTPClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

// standardHTTPClient は標準のhttp.Clientをラップする構造体
type standardHTTPClient struct {
	client *http.Client
}

func (c *standardHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// NewHTTPClient は新しいHTTPClientInterfaceを返す
func NewHTTPClient(timeout time.Duration) HTTPClientInterface {
	return &standardHTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// EsaClient はesa.ioのAPIクライアント
type EsaClient struct {
	httpClient HTTPClientInterface
	config     EsaConfig
}

// NewEsaClient は新しいEsaClientを作成する
func NewEsaClient(httpClient HTTPClientInterface, config EsaConfig) *EsaClient {
	return &EsaClient{
		httpClient: httpClient,
		config:     config,
	}
}

// ConfigFromEnv は環境変数からEsaConfigを生成する
func ConfigFromEnv() EsaConfig {
	teamName := os.Getenv("ESA_TEAM_NAME")
	accessToken := os.Getenv("ESA_ACCESS_TOKEN")
	return EsaConfig{
		TeamName:    teamName,
		AccessToken: accessToken,
	}
}

// SearchPostByCategory はカテゴリから投稿を検索する
func (c *EsaClient) SearchPostByCategory(category string) (*EsaPost, error) {
	// 検索クエリの構築
	url := fmt.Sprintf("https://api.esa.io/v1/teams/%s/posts?q=category:%s", c.config.TeamName, category)

	// リクエストの作成
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+c.config.AccessToken)

	// リクエストの実行
	resp, err := c.httpClient.Do(req)
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

// CreatePost は新しい投稿を作成する
func (c *EsaClient) CreatePost(text string) (*EsaPost, error) {
	// デフォルト値の設定
	now := time.Now()
	category := fmt.Sprintf("日報/%04d/%02d/%02d", now.Year(), now.Month(), now.Day())
	title := "日報"
	var tags []string

	url := fmt.Sprintf("https://api.esa.io/v1/teams/%s/posts", c.config.TeamName)

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
	req.Header.Add("Authorization", "Bearer "+c.config.AccessToken)

	// リクエストの実行
	resp, err := c.httpClient.Do(req)
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

// UpdatePost は既存の投稿を更新する
func (c *EsaClient) UpdatePost(existingPost *EsaPost, text string) (*EsaPost, error) {
	url := fmt.Sprintf("https://api.esa.io/v1/teams/%s/posts/%d", c.config.TeamName, existingPost.Number)

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
	req.Header.Add("Authorization", "Bearer "+c.config.AccessToken)

	// リクエストの実行
	resp, err := c.httpClient.Do(req)
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

// 以下は後方互換性のための関数です
// getEsaConfig は環境変数からesa.ioの設定を取得する
func getEsaConfig() EsaConfig {
	return ConfigFromEnv()
}

// createHTTPClient はHTTPクライアントを作成する
func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

// searchPostByCategory はカテゴリから投稿を検索する
func searchPostByCategory(client *http.Client, config EsaConfig, category string) (*EsaPost, error) {
	httpClient := &standardHTTPClient{client: client}
	esaClient := NewEsaClient(httpClient, config)
	return esaClient.SearchPostByCategory(category)
}

// createPost は新しい投稿を作成する
func createPost(client *http.Client, config EsaConfig, text string) (*EsaPost, error) {
	httpClient := &standardHTTPClient{client: client}
	esaClient := NewEsaClient(httpClient, config)
	return esaClient.CreatePost(text)
}

// updatePost は既存の投稿を更新する
func updatePost(client *http.Client, config EsaConfig, existingPost *EsaPost, text string) (*EsaPost, error) {
	httpClient := &standardHTTPClient{client: client}
	esaClient := NewEsaClient(httpClient, config)
	return esaClient.UpdatePost(existingPost, text)
}
