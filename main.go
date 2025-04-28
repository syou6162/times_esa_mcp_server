package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
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

// EsaSearchResult は検索結果を表す構造体
type EsaSearchResult struct {
	Posts      []EsaPost `json:"posts"`
	TotalCount int       `json:"total_count"`
}

// EsaErrorResponse はエラーレスポンスを表す構造体
type EsaErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
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

func submitDailyReport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// パラメーターの取得
	text, ok := request.Params.Arguments["text"].(string)
	if !ok {
		return nil, errors.New("text must be a string")
	}

	// esa.ioの設定を取得
	esaConfig := getEsaConfig()
	if esaConfig.TeamName == "" || esaConfig.AccessToken == "" {
		return nil, errors.New("ESA_TEAM_NAME または ESA_ACCESS_TOKEN が設定されていません")
	}

	client := createHTTPClient(esaConfig.AccessToken)

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

func main() {
	s := server.NewMCPServer(
		"times-esa-mcp-server",
		"1.0.0",
	)

	// times-esaツールの定義（日報投稿用 - textパラメータのみに簡略化）
	timesEsaTool := mcp.NewTool("times-esa",
		mcp.WithDescription("times-esaに日報を投稿します"),
		mcp.WithString("text",
			mcp.Required(),
			mcp.Description("投稿するテキスト内容"),
		),
	)

	// ツールの登録
	s.AddTool(timesEsaTool, submitDailyReport)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
