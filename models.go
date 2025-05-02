package main

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
