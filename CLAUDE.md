# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

このプロジェクトは、VS Code内からesa.ioサービスに日報を投稿するためのModel Context Protocol (MCP) サーバーです。Go 1.23.2で実装されており、mark3labs/mcp-goフレームワークを使用しています。

## 基本コマンド

### ビルド
```bash
go build -o times_esa_mcp_server .
```

### テスト実行
```bash
go test -v ./...
```

### 依存関係の管理
```bash
go mod tidy
go mod download
```

### モック生成
```bash
mockery --all
```

## アーキテクチャとコード構造

### 主要ファイルの役割

- **main.go**: MCPサーバーのエントリーポイント
- **handlers.go**: 日報投稿のハンドラー実装（`submitDailyReport`など）
- **esa_client.go**: esa.io APIとの通信を担当するクライアント実装
- **models.go**: データ構造の定義（EsaPost、EsaConfig等）
- **utils.go**: ユーティリティ関数（デバウンス処理、文字列処理など）

### インターフェース駆動設計

プロジェクトはテスト容易性を重視したインターフェース駆動設計を採用しています：

```go
type EsaClientInterface interface {
    SearchPostByCategory(category string) (*EsaPost, error)
    CreatePost(text string) (*EsaPost, error)
    UpdatePost(existingPost *EsaPost, text string) (*EsaPost, error)
}
```

## テスト方針

### テストフレームワーク
- **testify**: アサーションとモックフレームワーク
- **mockery**: インターフェースから自動的にモックを生成

### テストパターン
```go
// テーブル駆動テストを推奨
t.Run("テストケース名", func(t *testing.T) {
    // モックの作成
    mockEsaClient := NewMockEsaClientInterface(t)
    
    // 期待値の設定
    mockEsaClient.EXPECT().MethodName(args).Return(result, error)
    
    // テスト実行と検証
    result, err := functionUnderTest(mockEsaClient)
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
})
```

### テスト前のリセット
```go
// デバウンスのリセット（各テストケース前に必須）
resetDebounce()
```

## 開発時の重要な注意点

### 1. 環境変数
以下の環境変数が必要です：
- `ESA_TEAM_NAME`: esa.ioのチーム名
- `ESA_ACCESS_TOKEN`: アクセストークン（read/write権限必要）

### 2. デバウンス機能
- 同一テキストの重複投稿を防ぐため、300秒のデバウンスを実装
- テキスト類似度90%以上も重複とみなす
- テスト時は`resetDebounce()`を必ず呼ぶこと

### 3. 日報の形式
- カテゴリ: `日報/YYYY/MM/DD`
- 投稿時刻を`HH:MM`形式で自動付与
- 既存日報への追記は上部に挿入

### 4. エラーハンドリング
- API通信エラーは詳細なメッセージで返す
- デバウンスエラーは秒数を含めて返す

## CI/CD設定

GitHub Actionsで以下を実行：
- Go 1.24.3での自動ビルド
- 全テストの実行
- renovateによる依存関係の自動更新

## コード修正時のガイドライン

1. **インターフェースの変更時**
   - mockeryを実行してモックを再生成
   - 関連するテストを更新

2. **新機能追加時**
   - 対応するユニットテストを必ず追加
   - インターフェースを通じた実装を心がける

3. **リファクタリング時**
   - 既存のテストが通ることを確認
   - 後方互換性を維持（レガシー関数は残す）

## デバッグ方法

1. **ローカル実行**
   ```bash
   export ESA_TEAM_NAME=your_team
   export ESA_ACCESS_TOKEN=your_token
   ./times_esa_mcp_server
   ```

2. **VS Codeでの統合**
   settings.jsonに以下を追加：
   ```json
   {
       "mcp": {
           "servers": {
               "times-esa-mcp-server": {
                   "command": "${env:HOME}/go/bin/times_esa_mcp_server",
                   "args": [],
                   "env": {
                       "ESA_TEAM_NAME": "YOUR_TEAM_NAME",
                       "ESA_ACCESS_TOKEN": "YOUR_ACCESS_TOKEN"
                   }
               }
           }
       }
   }
   ```

## プロジェクト特有の慣習

1. **命名規則**
   - インターフェース名は`Interface`サフィックスを付ける
   - モック名は`Mock`プレフィックスを付ける

2. **エラーメッセージ**
   - 日本語でユーザーフレンドリーなメッセージを返す
   - 技術的な詳細は`fmt.Errorf`でラップ

3. **テスト構成**
   - 各ファイルに対応する`*_test.go`を作成
   - モックは`mocks_test.go`に集約（mockery自動生成）