package main

import (
	"context"
	"errors"
	"fmt"

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

func submitDailyReport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := request.Params.Arguments["name"].(string)
	if !ok {
		return nil, errors.New("name must be a string")
	}

	return mcp.NewToolResultText(fmt.Sprintf("Hello, %s!", name)), nil
}

func main() {
	s := server.NewMCPServer(
		"times-esa-mcp-server",
		"1.0.0",
	)

	tool := mcp.NewTool("times-esa",
		mcp.WithDescription("times-esaに日報を投稿します"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the person to greet"),
		),
	)

	s.AddTool(tool, submitDailyReport)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
