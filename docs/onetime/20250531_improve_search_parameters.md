# esa.io検索機能の一般化

## 背景

現在の`SearchPostByCategory`メソッドは日報検索に特化しており、以下の制限があった：
- カテゴリーの部分一致検索のみ対応
- 日付による絞り込み未対応
- タグやユーザーによる検索未対応
- ページネーション未対応

esa.io APIは豊富な検索オプションを提供しているため、これらを活用できるよう検索機能を一般化する必要がある。

## esa.io API仕様の確認

### 検索エンドポイント
- URL: `https://api.esa.io/v1/teams/:team_name/posts`
- メソッド: GET
- 認証: Bearer トークン

### 利用可能な検索オプション
1. **基本検索**
   - キーワード検索: `help`
   - 完全一致: `"exact phrase"`
   - AND検索: `keyword1 keyword2`
   - OR検索: `keyword1 OR keyword2`
   - 除外: `-keyword`

2. **カテゴリー検索**
   - 部分一致: `category:日報`
   - 前方一致: `in:日報/2024`
   - 完全一致: `on:日報/2024/12/20`

3. **その他の検索条件**
   - タグ: `tag:golang` or `#golang`
   - ユーザー: `user:screen_name` or `@screen_name`
   - 日付範囲: `created:>2024-01-01`, `updated:<2024-12-31`
   - ステータス: `wip:true`, `starred:true`

4. **ソートとページネーション**
   - ソート: `sort=updated`, `order=desc`
   - ページング: `page=1`, `per_page=50` (最大100)

## 設計案の検討

### 案1: 検索クエリビルダーパターン
```go
type SearchQuery struct {
    Keywords    []string
    Category    string
    CategoryOp  string // "partial", "prefix", "exact"
    Tags        []string
    // ...
}
```
- 利点: 構造が明確
- 欠点: 拡張時に構造体の変更が必要

### 案2: 機能的オプションパターン（採用）
```go
type SearchOption func(*searchConfig)

func WithCategory(category string) SearchOption
func WithTags(tags ...string) SearchOption
func WithDateRange(from, to time.Time) SearchOption
```
- 利点: 
  - Go言語らしいイディオム
  - 後方互換性を保ちやすい
  - 必要なオプションのみ指定可能
  - 将来の拡張が容易
- 欠点: 初見では理解しづらい可能性

### 案3: 生のクエリ文字列
```go
func SearchWithQuery(query string, page, perPage int) (*EsaSearchResult, error)
```
- 利点: 最も柔軟
- 欠点: 型安全性が低い、検証が困難

## 実装計画

### 1. 基本構造の定義
```go
type searchConfig struct {
    query      string
    page       int
    perPage    int
    sort       string
    order      string
}

type SearchOption func(*searchConfig)
```

### 2. オプション関数の実装
```go
func WithCategory(category string) SearchOption {
    return func(c *searchConfig) {
        c.query += fmt.Sprintf(" category:%s", category)
    }
}

func WithCategoryExact(category string) SearchOption {
    return func(c *searchConfig) {
        c.query += fmt.Sprintf(" on:%s", category)
    }
}

func WithTags(tags ...string) SearchOption {
    return func(c *searchConfig) {
        for _, tag := range tags {
            c.query += fmt.Sprintf(" tag:%s", tag)
        }
    }
}

func WithDateRange(field string, from, to time.Time) SearchOption {
    return func(c *searchConfig) {
        if !from.IsZero() {
            c.query += fmt.Sprintf(" %s:>%s", field, from.Format("2006-01-02"))
        }
        if !to.IsZero() {
            c.query += fmt.Sprintf(" %s:<%s", field, to.Format("2006-01-02"))
        }
    }
}

func WithPagination(page, perPage int) SearchOption {
    return func(c *searchConfig) {
        c.page = page
        c.perPage = perPage
    }
}
```

### 3. 汎用検索メソッド
```go
func (c *EsaClient) Search(options ...SearchOption) (*EsaSearchResult, error) {
    config := &searchConfig{
        page:    1,
        perPage: 20, // APIのデフォルト
        order:   "desc",
    }
    
    for _, opt := range options {
        opt(config)
    }
    
    // URLの構築とAPIリクエスト
    // ...
}
```

### 4. 既存メソッドのリファクタリング
```go
func (c *EsaClient) SearchPostByCategory(category string) (*EsaPost, error) {
    result, err := c.Search(
        WithCategory(category),
        WithPagination(1, 1),
    )
    // 既存のロジックを維持
}
```

## 使用例

```go
// 日報検索（既存の使い方）
post, err := client.SearchPostByCategory("日報/2024/12/20")

// 汎用検索の例
results, err := client.Search(
    WithCategory("日報"),
    WithDateRange("created", time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC), time.Now()),
    WithTags("golang", "mcp"),
    WithPagination(1, 50),
)

// 完全一致でのカテゴリー検索
results, err := client.Search(
    WithCategoryExact("日報/2024/12/20"),
)
```

## テスト計画

1. **APIリクエストの検証**
   - 正しいURLが構築されるか
   - クエリパラメータが適切にエンコードされるか
   - ヘッダーが正しく設定されるか

2. **オプションの組み合わせテスト**
   - 複数のオプションが正しく適用されるか
   - クエリ文字列が正しく結合されるか

3. **エラーハンドリング**
   - APIエラーレスポンスの処理
   - ネットワークエラーの処理

4. **後方互換性の確認**
   - 既存のSearchPostByCategoryが同じ動作を維持するか

## 今後の拡張可能性

- ソート条件の追加（stars, comments等）
- 高度な検索演算子のサポート（括弧、複雑なOR条件）
- 検索結果のキャッシング
- 全ページ取得のヘルパー関数