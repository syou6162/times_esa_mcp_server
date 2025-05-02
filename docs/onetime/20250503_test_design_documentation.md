# times-esa MCP サーバー テスト設計ドキュメント

作成日: 2025年5月3日

## 概要

このドキュメントでは、times-esa MCP サーバーのテスト設計とライブラリ選択に関する検討内容をまとめています。コードベースが複数のGoファイルに分割されたため、それぞれに適切なテストを実装する方針を決定しました。

## テスト可能性に関する課題

現在のコードベースには、テスト容易性に関して以下の課題があります：

1. **環境変数への直接依存**:
   ```go
   func getEsaConfig() EsaConfig {
       teamName := os.Getenv("ESA_TEAM_NAME")
       accessToken := os.Getenv("ESA_ACCESS_TOKEN")
       // ...
   }
   ```
   - 環境変数に直接依存しているため、テスト環境での制御が難しい

2. **HTTPクライアントの直接生成**:
   ```go
   func createHTTPClient() *http.Client {
       return &http.Client{
           Timeout: 10 * time.Second,
       }
   }
   ```
   - モック化が困難な実装になっている

3. **esa.io APIとの直接通信**:
   - テスト実行時に実際のAPIと通信する必要がない設計が必要

## テスト戦略

### 1. インターフェース駆動設計の採用

```go
// EsaClientInterface はesa.ioとの通信を担当するインターフェース
type EsaClientInterface interface {
    SearchPostByCategory(category string) (*EsaPost, error)
    CreatePost(config EsaConfig, text string) (*EsaPost, error)
    UpdatePost(config EsaConfig, existingPost *EsaPost, text string) (*EsaPost, error)
    // その他必要なメソッド
}
```

### 2. 依存性注入パターンの活用

```go
// 実装クラス
type EsaClient struct {
    client *http.Client
    config EsaConfig
}

// コンストラクタ
func NewEsaClient(client *http.Client, config EsaConfig) *EsaClient {
    return &EsaClient{
        client: client,
        config: config,
    }
}
```

### 3. 既存関数のリファクタリング

例えば、`submitDailyReport`関数を以下のように改造します:

```go
func submitDailyReport(ctx context.Context, request mcp.CallToolRequest, esaClient EsaClientInterface) (*mcp.CallToolResult, error) {
    // パラメーターの取得
    text, ok := request.Params.Arguments["text"].(string)
    if !ok {
        return nil, errors.New("text must be a string")
    }

    // ...既存の処理...

    // インターフェースを通して操作
    existingPost, err := esaClient.SearchPostByCategory(category)
    // ...
}
```

## モックライブラリ検討

以下の3つの主要なモックライブラリを比較検討しました:

### 1. testify/mock

- **リポジトリ**: [github.com/stretchr/testify](https://github.com/stretchr/testify)
- **スター数**: 約20.7k
- **メンテナンス状況**: 非常に活発
- **メリット**: 簡単に始められる、柔軟性が高い
- **デメリット**: インターフェース変更時の手動更新が必要

testify/mockでは以下のようなモック実装が必要:

```go
// モッククラスの手動実装
type MockEsaClient struct {
    mock.Mock
}

// インターフェースの各メソッドを手動実装
func (m *MockEsaClient) SearchPostByCategory(category string) (*EsaPost, error) {
    args := m.Called(category)
    if post := args.Get(0); post != nil {
        return post.(*EsaPost), args.Error(1)
    }
    return nil, args.Error(1)
}

// 他のメソッドも同様に実装
```

### 2. golang/mock (gomock)

- **リポジトリ**: [github.com/golang/mock](https://github.com/golang/mock)
- **スター数**: 約8.7k
- **重要通知**: `2023年6月にメンテナンス終了、Uberのフォークに移行`

### 3. mockery

- **リポジトリ**: [github.com/vektra/mockery](https://github.com/vektra/mockery)
- **スター数**: 約4.9k
- **メンテナンス状況**: 活発
- **メリット**: インターフェースから自動的にモック実装を生成
- **デメリット**: 外部ツールへの依存

mockeryでは以下のコマンドでモック生成が可能:

```bash
go install github.com/vektra/mockery/v2@latest
mockery --name=EsaClientInterface
```

## 選定結果

単独開発環境での中長期的なメンテナンス性を考慮し、**mockery**を採用することに決定しました。主な理由は:

1. **自動生成の恩恵**: インターフェースが複雑な場合でも、変更に容易に追従できる
2. **コードの一貫性**: 自動生成によりモック実装の品質が安定
3. **活発なメンテナンス**: 定期的な更新が継続されている
4. **testifyとの互換性**: 広く使われているtestifyのmockをベースにしている

## 次のステップ

1. インターフェースの適切な設計と実装
2. mockeryのセットアップ
3. テストケースの実装
   - ユニットテスト
   - 統合テスト（必要に応じて）
4. CIへの組み込み

---

このドキュメントは、テスト設計の進捗や新しい検討内容に応じて更新されます。
