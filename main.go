package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// EsaConfig はesa.ioへの接続設定を保持する構造体
type EsaConfig struct {
	TeamName    string
	AccessToken string
}

// EsaPost はesa.ioの投稿データを表す構造体
type EsaPost struct {
	BodyMd   string   `json:"body_md"`
	BodyHtml string   `json:"body_html"`
	Number   int      `json:"number"`
	Name     string   `json:"name"`
	Tags     []string `json:"tags"`
}

// TimesEsaPostRequest は日報投稿リクエストの構造体
type TimesEsaPostRequest struct {
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
	Title    string   `json:"title"`
	Text     string   `json:"text"`
}

// EsaSearchResult は検索結果を表す構造体
type EsaSearchResult struct {
	Posts      []EsaPost `json:"posts"`
	TotalCount int       `json:"total_count"`
}

// Tag はタグ情報を表す構造体
type Tag struct {
	Name       string `json:"name"`
	PostsCount int    `json:"posts_count"`
}

// EsaTags はタグ一覧を表す構造体
type EsaTags struct {
	Tags []Tag `json:"tags"`
}

// EsaErrorResponse はエラーレスポンスを表す構造体
type EsaErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// 日報の投稿用のリクエスト
type DailyReportRequest struct {
	Text     string   `json:"text"`
	Tags     []string `json:"tags,omitempty"`
	Category string   `json:"category"`
	Title    string   `json:"title"`
}

// 日報の取得用のリクエスト
type GetDailyReportRequest struct {
	Category string `json:"category"`
}

// MCPのレスポンス用の構造体
type DailyReportResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Post    EsaPost `json:"post,omitempty"`
}

// getEsaConfig は環境変数からesa.ioの設定を取得する
func getEsaConfig() EsaConfig {
	teamName := os.Getenv("ESA_TEAM_NAME")
	accessToken := os.Getenv("ESA_ACCESS_TOKEN")
	return EsaConfig{
		TeamName:    teamName,
		AccessToken: accessToken,
	}
}

// createHTTPClient は認証済みのHTTPクライアントを作成する
func createHTTPClient(accessToken string) *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

// transformTitle はタイトルを変換する
func transformTitle(oldTitle string, newTitle string) string {
	// カンマまたは「、」でタイトルを分割し、重複を除去して結合
	var combinedTitles []string

	// 古いタイトルを分割
	oldParts := strings.FieldsFunc(oldTitle, func(r rune) bool {
		return r == ',' || r == '、'
	})

	// 新しいタイトルを分割
	newParts := strings.FieldsFunc(newTitle, func(r rune) bool {
		return r == ',' || r == '、'
	})

	// 全てのパートを結合し、マップを使って重複を除去
	titleMap := make(map[string]bool)
	for _, part := range oldParts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			titleMap[trimmed] = true
		}
	}

	for _, part := range newParts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			titleMap[trimmed] = true
		}
	}

	// マップからスライスに変換
	for title := range titleMap {
		combinedTitles = append(combinedTitles, title)
	}

	// 「日報」だけの場合は「日報」を返す
	if len(combinedTitles) == 1 && combinedTitles[0] == "日報" {
		return "日報"
	}

	// 「日報」以外のタイトル部分を「、」で結合
	var filteredTitles []string
	for _, title := range combinedTitles {
		if title != "日報" {
			filteredTitles = append(filteredTitles, title)
		}
	}

	return strings.Join(filteredTitles, "、")
}

// createOrUpdatePost は日報を作成または更新する
func createOrUpdatePost(config EsaConfig, category string, tags []string, title string, text string) (*EsaPost, error) {
	client := createHTTPClient(config.AccessToken)

	// 既存の投稿を検索
	existingPost, err := searchPostByCategory(client, config, category)
	if err != nil {
		return nil, fmt.Errorf("投稿の検索に失敗しました: %w", err)
	}

	if existingPost == nil {
		// 新しい投稿を作成
		return createPost(client, config, category, tags, title, text)
	}

	// 既存の投稿を更新
	return updatePost(client, config, existingPost, category, tags, title, text)
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

// createPost は新しい投稿を作成する
func createPost(client *http.Client, config EsaConfig, category string, tags []string, title string, text string) (*EsaPost, error) {
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
	reqBody.Post.BodyMd = text
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

// updatePost は既存の投稿を更新する
func updatePost(client *http.Client, config EsaConfig, existingPost *EsaPost, category string, tags []string, title string, text string) (*EsaPost, error) {
	url := fmt.Sprintf("https://api.esa.io/v1/teams/%s/posts/%d", config.TeamName, existingPost.Number)

	// リクエストボディの作成
	type patchRequest struct {
		Post struct {
			Name     string   `json:"name"`
			Category string   `json:"category"`
			Tags     []string `json:"tags"`
			BodyMd   string   `json:"body_md"`
			Wip      bool     `json:"wip"`
		} `json:"post"`
	}

	reqBody := patchRequest{}
	reqBody.Post.Name = transformTitle(existingPost.Name, title)
	reqBody.Post.Category = category

	// タグをマージ（重複を除去）
	allTags := make(map[string]bool)
	for _, tag := range existingPost.Tags {
		allTags[tag] = true
	}
	for _, tag := range tags {
		allTags[tag] = true
	}
	var mergedTags []string
	for tag := range allTags {
		mergedTags = append(mergedTags, tag)
	}
	reqBody.Post.Tags = mergedTags

	// テキストを追記（新しいテキストを上に）
	if text != "" {
		reqBody.Post.BodyMd = text + "\n" + existingPost.BodyMd
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

// getDailyReport は日報を取得する
func getDailyReport(config EsaConfig, category string) (*EsaPost, error) {
	client := createHTTPClient(config.AccessToken)

	// 既存の投稿を検索
	post, err := searchPostByCategory(client, config, category)
	if err != nil {
		return nil, err
	}

	if post == nil {
		return nil, errors.New("今日の日報はまだありません")
	}

	// 投稿の詳細情報を取得
	url := fmt.Sprintf("https://api.esa.io/v1/teams/%s/posts/%d", config.TeamName, post.Number)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+config.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp EsaErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("エラーレスポンスの解析に失敗: %w", err)
		}
		return nil, fmt.Errorf("%s: %s", errorResp.Error, errorResp.Message)
	}

	var postDetail EsaPost
	if err := json.NewDecoder(resp.Body).Decode(&postDetail); err != nil {
		return nil, fmt.Errorf("投稿の解析に失敗: %w", err)
	}

	return &postDetail, nil
}

// getTagList はタグ一覧を取得する
func getTagList(config EsaConfig) (*EsaTags, error) {
	client := createHTTPClient(config.AccessToken)

	url := fmt.Sprintf("https://api.esa.io/v1/teams/%s/tags", config.TeamName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+config.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp EsaErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("エラーレスポンスの解析に失敗: %w", err)
		}
		return nil, fmt.Errorf("%s: %s", errorResp.Error, errorResp.Message)
	}

	var tags EsaTags
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("タグの解析に失敗: %w", err)
	}

	return &tags, nil
}

func submitDailyReport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// パラメーターの取得
	text, ok := request.Params.Arguments["text"].(string)
	if !ok {
		return nil, errors.New("text must be a string")
	}

	// カテゴリパラメーターの取得（デフォルトは日付ベースのカテゴリ）
	category, ok := request.Params.Arguments["category"].(string)
	if !ok || category == "" {
		// デフォルトのカテゴリ: 日報/YYYY/MM/DD
		now := time.Now()
		category = fmt.Sprintf("日報/%04d/%02d/%02d", now.Year(), now.Month(), now.Day())
	}

	// タイトルパラメーターの取得（デフォルトは「日報」）
	title, ok := request.Params.Arguments["title"].(string)
	if !ok || title == "" {
		title = "日報"
	}

	// タグパラメーターの取得（オプショナル）
	var tags []string
	tagsRaw, ok := request.Params.Arguments["tags"]
	if ok {
		tagsArray, ok := tagsRaw.([]interface{})
		if ok {
			for _, tag := range tagsArray {
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, tagStr)
				}
			}
		}
	}

	// esa.ioの設定を取得
	esaConfig := getEsaConfig()
	if esaConfig.TeamName == "" || esaConfig.AccessToken == "" {
		return nil, errors.New("ESA_TEAM_NAME または ESA_ACCESS_TOKEN が設定されていません")
	}

	// 日報を作成または更新
	post, err := createOrUpdatePost(esaConfig, category, tags, title, text)
	if err != nil {
		return nil, fmt.Errorf("日報の投稿に失敗しました: %w", err)
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

	// mcp.NewToolResultJSON の代わりに mcp.NewToolResultText を使用
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func main() {
	s := server.NewMCPServer(
		"times-esa-mcp-server",
		"1.0.0",
	)

	// times-esaツールの定義（日報投稿用）
	timesEsaTool := mcp.NewTool("times-esa",
		mcp.WithDescription("times-esaに日報を投稿します"),
		mcp.WithString("text",
			mcp.Required(),
			mcp.Description("投稿するテキスト内容"),
		),
		mcp.WithString("category",
			mcp.Description("投稿するカテゴリ（デフォルトは日報/YYYY/MM/DD）"),
		),
		mcp.WithString("title",
			mcp.Description("投稿のタイトル（デフォルトは「日報」）"),
		),
		mcp.WithArray("tags",
			mcp.Description("投稿に付けるタグ（オプション）"),
		),
	)

	// ツールの登録
	s.AddTool(timesEsaTool, submitDailyReport)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
