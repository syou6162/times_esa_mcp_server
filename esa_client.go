package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// esaAPIBaseURL はesa.io APIのベースURL
	esaAPIBaseURL = "https://api.esa.io/v1"
	
	// エンドポイントのパス定義
	esaPostsEndpoint = "/teams/%s/posts"     // チームの投稿一覧
	esaPostEndpoint  = "/teams/%s/posts/%d"  // 特定の投稿
)

// searchConfig は検索オプションを保持する構造体
type searchConfig struct {
	query         string
	categoryQuery string // カテゴリー検索専用フィールド（排他的）
	page          int
	perPage       int
	sort          string
	order         string
}

// SearchOption は検索設定を変更する関数型
type SearchOption func(*searchConfig)

// EsaClientInterface はesa.ioとの通信を担当するインターフェース
type EsaClientInterface interface {
	Search(options ...SearchOption) (*EsaSearchResult, error)
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
	// Searchメソッドを使用
	result, err := c.Search(
		WithCategory(category),
		WithPagination(1, 1),
	)
	if err != nil {
		return nil, err
	}

	// 検索結果の処理
	if result.TotalCount == 0 {
		// 投稿が存在しない
		return nil, nil
	} else if result.TotalCount > 1 {
		// 複数の投稿が存在する
		return nil, errors.New("複数の日報が存在します")
	}

	// 最新の投稿を返す
	return &result.Posts[0], nil
}

// Search は汎用的な検索を実行する
func (c *EsaClient) Search(options ...SearchOption) (*EsaSearchResult, error) {
	// デフォルト設定
	config := &searchConfig{
		page:    1,
		perPage: 20, // APIのデフォルト値
		sort:    "updated",
		order:   "desc",
	}

	// オプションを適用
	for _, opt := range options {
		opt(config)
	}

	// URLの構築
	apiURL := fmt.Sprintf("%s%s", esaAPIBaseURL, fmt.Sprintf(esaPostsEndpoint, c.config.TeamName))
	
	// クエリパラメータの構築
	params := make(map[string]string)
	
	// クエリ文字列の構築
	queryParts := []string{}
	if config.categoryQuery != "" {
		queryParts = append(queryParts, config.categoryQuery)
	}
	if config.query != "" {
		queryParts = append(queryParts, config.query)
	}
	if len(queryParts) > 0 {
		params["q"] = strings.Join(queryParts, " ")
	}
	if config.page > 0 {
		params["page"] = fmt.Sprintf("%d", config.page)
	}
	if config.perPage > 0 {
		params["per_page"] = fmt.Sprintf("%d", config.perPage)
	}
	if config.sort != "" {
		params["sort"] = config.sort
	}
	if config.order != "" {
		params["order"] = config.order
	}

	// クエリパラメータをURLに追加
	if len(params) > 0 {
		values := url.Values{}
		for key, value := range params {
			values.Add(key, value)
		}
		apiURL += "?" + values.Encode()
	}

	// リクエストの作成
	req, err := http.NewRequest("GET", apiURL, nil)
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

	return &searchResult, nil
}

// CreatePost は新しい投稿を作成する
func (c *EsaClient) CreatePost(text string) (*EsaPost, error) {
	// デフォルト値の設定
	now := time.Now()
	category := fmt.Sprintf("日報/%04d/%02d/%02d", now.Year(), now.Month(), now.Day())
	title := "日報"
	var tags []string

	url := fmt.Sprintf("%s%s", esaAPIBaseURL, fmt.Sprintf(esaPostsEndpoint, c.config.TeamName))

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

	// 現在時刻をアンカーリンク付きで取得し、テキストの前に追加、その後に区切り線を追加
	timePrefix := GenerateTimestampWithAnchor(now)
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
	url := fmt.Sprintf("%s%s", esaAPIBaseURL, fmt.Sprintf(esaPostEndpoint, c.config.TeamName, existingPost.Number))

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
		// 現在時刻をアンカーリンク付きで取得
		now := time.Now()
		timePrefix := GenerateTimestampWithAnchor(now)

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

// WithCategory はカテゴリーの部分一致検索オプションを返す
func WithCategory(category string) SearchOption {
	return func(c *searchConfig) {
		c.categoryQuery = fmt.Sprintf("category:%s", category)
	}
}

// WithCategoryExact はカテゴリーの完全一致検索オプションを返す
func WithCategoryExact(category string) SearchOption {
	return func(c *searchConfig) {
		c.categoryQuery = fmt.Sprintf("on:%s", category)
	}
}

// WithCategoryPrefix はカテゴリーの前方一致検索オプションを返す
func WithCategoryPrefix(category string) SearchOption {
	return func(c *searchConfig) {
		c.categoryQuery = fmt.Sprintf("in:%s", category)
	}
}

// WithTags はタグ検索オプションを返す
func WithTags(tags ...string) SearchOption {
	return func(c *searchConfig) {
		for _, tag := range tags {
			if c.query != "" {
				c.query += " "
			}
			c.query += fmt.Sprintf("tag:%s", tag)
		}
	}
}

// WithKeywords はキーワード検索オプションを返す
func WithKeywords(keywords ...string) SearchOption {
	return func(c *searchConfig) {
		for _, keyword := range keywords {
			if c.query != "" {
				c.query += " "
			}
			c.query += keyword
		}
	}
}

// WithUser はユーザー検索オプションを返す
func WithUser(screenName string) SearchOption {
	return func(c *searchConfig) {
		if c.query != "" {
			c.query += " "
		}
		c.query += fmt.Sprintf("user:%s", screenName)
	}
}

// WithDateRange は日付範囲検索オプションを返す
func WithDateRange(field string, from, to time.Time) SearchOption {
	return func(c *searchConfig) {
		if !from.IsZero() {
			if c.query != "" {
				c.query += " "
			}
			c.query += fmt.Sprintf("%s:>%s", field, from.Format("2006-01-02"))
		}
		if !to.IsZero() {
			if c.query != "" {
				c.query += " "
			}
			c.query += fmt.Sprintf("%s:<%s", field, to.Format("2006-01-02"))
		}
	}
}

// WithWIP はWIP状態検索オプションを返す
func WithWIP(wip bool) SearchOption {
	return func(c *searchConfig) {
		if c.query != "" {
			c.query += " "
		}
		c.query += fmt.Sprintf("wip:%t", wip)
	}
}

// WithStarred はスター状態検索オプションを返す
func WithStarred(starred bool) SearchOption {
	return func(c *searchConfig) {
		if c.query != "" {
			c.query += " "
		}
		c.query += fmt.Sprintf("starred:%t", starred)
	}
}

// WithPagination はページネーションオプションを返す
func WithPagination(page, perPage int) SearchOption {
	return func(c *searchConfig) {
		c.page = page
		c.perPage = perPage
	}
}

// WithSort はソートオプションを返す
func WithSort(sort, order string) SearchOption {
	return func(c *searchConfig) {
		c.sort = sort
		c.order = order
	}
}
