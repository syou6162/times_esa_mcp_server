package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// buildTestURL はテスト用のAPIエンドポイントURLを構築する
func buildTestURL(teamName string, queryParams string) string {
	baseURL := fmt.Sprintf("%s%s", esaAPIBaseURL, fmt.Sprintf(esaPostsEndpoint, teamName))
	if queryParams != "" {
		return baseURL + "?" + queryParams
	}
	return baseURL
}

// TestSearch_URLConstruction はSearchメソッドが正しいURLを構築することを検証する
func TestSearch_URLConstruction(t *testing.T) {
	tests := []struct {
		name            string
		options         []SearchOption
		expectedURL     string
		expectedHeaders map[string]string
	}{
		{
			name:    "基本的な検索（オプションなし）",
			options: []SearchOption{},
			expectedURL: buildTestURL("test-team",
				"order=desc&page=1&per_page=20&sort=updated"),
			expectedHeaders: map[string]string{
				"Authorization": "Bearer test-token",
			},
		},
		{
			name: "カテゴリー検索",
			options: []SearchOption{
				WithCategory("日報/2024/12"),
			},
			expectedURL: buildTestURL("test-team",
				"order=desc&page=1&per_page=20&q=category%3A%E6%97%A5%E5%A0%B1%2F2024%2F12&sort=updated"),
			expectedHeaders: map[string]string{
				"Authorization": "Bearer test-token",
			},
		},
		{
			name: "複数のオプション",
			options: []SearchOption{
				WithCategory("日報"),
				WithTags("golang", "mcp"),
				WithPagination(2, 50),
			},
			expectedURL: buildTestURL("test-team",
				"order=desc&page=2&per_page=50&q=category%3A%E6%97%A5%E5%A0%B1+tag%3Agolang+tag%3Amcp&sort=updated"),
			expectedHeaders: map[string]string{
				"Authorization": "Bearer test-token",
			},
		},
		{
			name: "日付範囲検索",
			options: []SearchOption{
				WithDateRange("created", 
					time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
				),
			},
			expectedURL: buildTestURL("test-team",
				"order=desc&page=1&per_page=20&q=created%3A%3E2024-01-01+created%3A%3C2024-12-31&sort=updated"),
			expectedHeaders: map[string]string{
				"Authorization": "Bearer test-token",
			},
		},
		{
			name: "ソート指定",
			options: []SearchOption{
				WithSort("created", "asc"),
			},
			expectedURL: buildTestURL("test-team",
				"order=asc&page=1&per_page=20&sort=created"),
			expectedHeaders: map[string]string{
				"Authorization": "Bearer test-token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックHTTPクライアントの作成
			mockHTTPClient := NewMockHTTPClientInterface(t)
			
			// リクエストの検証
			mockHTTPClient.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool {
				// URLの検証
				actualURL := req.URL.String()
				
				// クエリパラメータをパースして比較（順序に依存しない比較）
				expectedParsed, _ := url.Parse(tt.expectedURL)
				actualParsed, _ := url.Parse(actualURL)
				
				assert.Equal(t, expectedParsed.Scheme, actualParsed.Scheme)
				assert.Equal(t, expectedParsed.Host, actualParsed.Host)
				assert.Equal(t, expectedParsed.Path, actualParsed.Path)
				
				// クエリパラメータの比較
				expectedQuery := expectedParsed.Query()
				actualQuery := actualParsed.Query()
				assert.Equal(t, expectedQuery, actualQuery)
				
				// ヘッダーの検証
				for key, value := range tt.expectedHeaders {
					assert.Equal(t, value, req.Header.Get(key))
				}
				
				return true
			})).Return(&http.Response{
				StatusCode: 200,
				Body: io.NopCloser(strings.NewReader(`{
					"posts": [],
					"total_count": 0,
					"page": 1,
					"per_page": 20,
					"max_per_page": 100
				}`)),
			}, nil)

			// EsaClientの作成とテスト実行
			config := EsaConfig{
				TeamName:    "test-team",
				AccessToken: "test-token",
			}
			client := NewEsaClient(mockHTTPClient, config)
			
			_, err := client.Search(tt.options...)
			assert.NoError(t, err)
		})
	}
}

// TestSearch_QueryConstruction は各SearchOptionが正しくクエリを構築することを検証する
func TestSearch_QueryConstruction(t *testing.T) {
	tests := []struct {
		name          string
		options       []SearchOption
		expectedQuery string
	}{
		{
			name: "WithCategoryExact",
			options: []SearchOption{
				WithCategoryExact("日報/2024/12/20"),
			},
			expectedQuery: "on:日報/2024/12/20",
		},
		{
			name: "WithCategoryPrefix",
			options: []SearchOption{
				WithCategoryPrefix("日報/2024"),
			},
			expectedQuery: "in:日報/2024",
		},
		{
			name: "WithKeywords",
			options: []SearchOption{
				WithKeywords("golang", "mcp", "server"),
			},
			expectedQuery: "golang mcp server",
		},
		{
			name: "WithUser",
			options: []SearchOption{
				WithUser("test_user"),
			},
			expectedQuery: "user:test_user",
		},
		{
			name: "WithWIP",
			options: []SearchOption{
				WithWIP(true),
			},
			expectedQuery: "wip:true",
		},
		{
			name: "WithStarred",
			options: []SearchOption{
				WithStarred(false),
			},
			expectedQuery: "starred:false",
		},
		{
			name: "複雑な組み合わせ",
			options: []SearchOption{
				WithCategory("日報"),
				WithTags("golang"),
				WithUser("test_user"),
				WithWIP(false),
				WithKeywords("MCP", "サーバー"),
			},
			expectedQuery: "category:日報 tag:golang user:test_user wip:false MCP サーバー",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// searchConfigを作成してオプションを適用
			config := &searchConfig{
				page:    1,
				perPage: 20,
				sort:    "updated",
				order:   "desc",
			}
			
			for _, opt := range tt.options {
				opt(config)
			}
			
			assert.Equal(t, tt.expectedQuery, config.query)
		})
	}
}

// TestSearchPostByCategory_BackwardCompatibility は既存メソッドの後方互換性を検証する
func TestSearchPostByCategory_BackwardCompatibility(t *testing.T) {
	// モックHTTPクライアントの作成
	mockHTTPClient := NewMockHTTPClientInterface(t)
	
	// 単一の結果を返すケース
	mockHTTPClient.EXPECT().Do(mock.Anything).Return(&http.Response{
		StatusCode: 200,
		Body: io.NopCloser(strings.NewReader(`{
			"posts": [{
				"number": 123,
				"name": "日報",
				"body_md": "テスト内容",
				"category": "日報/2024/12/20"
			}],
			"total_count": 1,
			"page": 1,
			"per_page": 1
		}`)),
	}, nil)

	config := EsaConfig{
		TeamName:    "test-team",
		AccessToken: "test-token",
	}
	client := NewEsaClient(mockHTTPClient, config)
	
	post, err := client.SearchPostByCategory("日報/2024/12/20")
	assert.NoError(t, err)
	assert.NotNil(t, post)
	assert.Equal(t, 123, post.Number)
	assert.Equal(t, "日報", post.Name)
}

// TestSearch_ErrorHandling はエラーハンドリングを検証する
func TestSearch_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:       "APIエラーレスポンス",
			statusCode: 401,
			responseBody: `{
				"error": "unauthorized",
				"message": "Invalid access token"
			}`,
			expectedError: "unauthorized: Invalid access token",
		},
		{
			name:       "不正なJSONレスポンス",
			statusCode: 200,
			responseBody: `{invalid json`,
			expectedError: "検索結果の解析に失敗",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTPClient := NewMockHTTPClientInterface(t)
			
			mockHTTPClient.EXPECT().Do(mock.Anything).Return(&http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
			}, nil)

			config := EsaConfig{
				TeamName:    "test-team",
				AccessToken: "test-token",
			}
			client := NewEsaClient(mockHTTPClient, config)
			
			_, err := client.Search()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// TestSearch_NetworkError はネットワークエラーを検証する
func TestSearch_NetworkError(t *testing.T) {
	mockHTTPClient := NewMockHTTPClientInterface(t)
	
	mockHTTPClient.EXPECT().Do(mock.Anything).Return(nil, fmt.Errorf("network error"))

	config := EsaConfig{
		TeamName:    "test-team",
		AccessToken: "test-token",
	}
	client := NewEsaClient(mockHTTPClient, config)
	
	_, err := client.Search()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
}