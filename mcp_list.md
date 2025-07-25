# MCPサーバーでのスラッシュコマンド実装方法

## 概要

Claude CodeでVSCodeのCopilot Agentのような `#times_esa` 形式の直接的なMCPツール呼び出し機能は現在実装されていない。しかし、MCPサーバー側で**プロンプトテンプレート**を実装することで、`/times_esa:quick-post (MCP)` のような形でスラッシュコマンドとして利用可能になる。

## 現在の状況

- **times_esa_mcp_server**: ツール(`times-esa`)のみ実装、プロンプトテンプレートは未実装
- **motherduck MCP**: ツール(`query`)とプロンプトテンプレート(`duckdb-motherduck-initial-prompt`)両方実装

→ **プロンプトテンプレートがある場合のみ、Claude Codeでスラッシュコマンドとして認識される**

## 実装例：motherduck MCPサーバー

### 現在の動作確認
- `/duckdb:duckdb-motherduck-initial-prompt (MCP)` がClaude Codeで認識されている
- リポジトリ: https://github.com/motherduckdb/mcp-server-motherduck

### 実装方法

motherduck MCPサーバーでは以下の2つのハンドラーでプロンプトテンプレートを実装：

```python
@server.list_prompts()
async def handle_list_prompts() -> list[types.Prompt]:
    """利用可能なプロンプトをリスト"""
    return [
        types.Prompt(
            name="duckdb-motherduck-initial-prompt",
            description="A prompt to initialize a connection to duckdb or motherduck and start working with it",
        )
    ]

@server.get_prompt()
async def handle_get_prompt(
    name: str, arguments: dict[str, str] | None
) -> types.GetPromptResult:
    """プロンプトの実際の内容を返す"""
    if name != "duckdb-motherduck-initial-prompt":
        raise ValueError(f"Unknown prompt: {name}")

    return types.GetPromptResult(
        description="Initial prompt for interacting with DuckDB/MotherDuck",
        messages=[
            types.PromptMessage(
                role="user",
                content=types.TextContent(type="text", text=PROMPT_TEMPLATE),
            )
        ],
    )
```

## times_esa_mcp_serverでの実装案

### 目標
`/times_esa:quick-post (MCP)` として利用可能にする

### 実装コード

```python
@server.list_prompts()
async def handle_list_prompts() -> list[types.Prompt]:
    """利用可能なプロンプトをリスト"""
    return [
        types.Prompt(
            name="quick-post",
            description="Quickly post to times_esa journal",
        ),
        types.Prompt(
            name="post-with-link",
            description="Post to times_esa with a GitHub link or URL",
        )
    ]

@server.get_prompt()
async def handle_get_prompt(
    name: str, arguments: dict[str, str] | None
) -> types.GetPromptResult:
    """プロンプトの実際の内容を返す"""
    
    if name == "quick-post":
        return types.GetPromptResult(
            description="Quick post to times_esa",
            messages=[
                types.PromptMessage(
                    role="user",
                    content=types.TextContent(
                        type="text", 
                        text="以下の内容をtimes_esaに投稿してください：\n\n$ARGUMENTS"
                    ),
                )
            ],
        )
    
    elif name == "post-with-link":
        return types.GetPromptResult(
            description="Post to times_esa with link",
            messages=[
                types.PromptMessage(
                    role="user",
                    content=types.TextContent(
                        type="text", 
                        text="以下のリンクと内容をtimes_esaに投稿してください：\n\n$ARGUMENTS"
                    ),
                )
            ],
        )
    
    else:
        raise ValueError(f"Unknown prompt: {name}")
```

### 期待される動作

実装後、Claude Codeで以下が利用可能になる：

1. `/times_esa:quick-post (MCP)` - 素早い投稿
2. `/times_esa:post-with-link (MCP)` - リンク付き投稿

### 引数の渡し方

Claude Codeでは `$ARGUMENTS` にユーザーが入力した内容が自動的に置換される。

例：
```
/times_esa:quick-post claude codeの調査完了！
```

→ プロンプトテンプレートの `$ARGUMENTS` が「claude codeの調査完了！」に置換される

## 参考情報

### 関連GitHub Issue
- Issue #703: Slash Command Registration Collision with MCP Server Configuration
- Issue #723: claude doesn't wait until mcp servers are connected before initial prompt
- Issue #1175: --permission-prompt-tool needs minimal, working example and documentation

### MCP仕様
- MCPサーバーは `list_prompts` と `get_prompt` メソッドでプロンプトテンプレート機能を提供
- Claude Codeが自動的に `/サーバー名:プロンプト名 (MCP)` 形式で認識
- `$ARGUMENTS` でユーザー入力を受け取り可能

### 次のステップ

1. times_esa_mcp_serverのソースコードを確認
2. 上記の実装コードを追加
3. テスト・動作確認
4. 必要に応じてプロンプトテンプレートの内容を調整