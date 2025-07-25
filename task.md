# Claude Code履歴ファイルを使った作業サマリー生成手順

## 概要
Claude Codeの対話履歴（JSONLファイル）を使って、日々の作業内容を自動的に抽出・要約する方法。

## 履歴ファイルの場所と構造

### ファイルの保存場所
```
~/.claude/projects/{プロジェクトパスをエンコードした名前}/*.jsonl
```

例:
- プロジェクトパス: `/Users/yasuhisa.yoshida/work/times-esa-mcp-server`
- 保存ディレクトリ: `~/.claude/projects/-Users-yasuhisa-yoshida-work-times-esa-mcp-server/`

### JSONLファイルの構造
各行が独立したJSONオブジェクト:
```json
{
  "type": "user",           // メッセージの種類（user, assistant, summary等）
  "timestamp": "2025-05-31T14:57:13.742Z",  // UTC形式のタイムスタンプ
  "message": {
    "role": "user",
    "content": "メッセージ内容"  // 文字列または配列
  },
  "sessionId": "d1461099-8faf-4ec3-a295-7c91d358e970",
  "userType": "external"
}
```

## 手動での分析方法（jqコマンド）

### 基本的な抽出
```bash
# type=userのメッセージを抽出
jq 'select(.type == "user")' ファイル名.jsonl

# contentが配列と文字列の両方に対応
jq -r 'select(.type == "user") | 
  .timestamp + " | " + 
  (if .message.content | type == "array" then 
    (.message.content[] | select(.type == "text") | .text // empty) 
  elif .message.content | type == "string" then 
    .message.content 
  else 
    empty 
  end)' ファイル名.jsonl
```

## DuckDBを使った分析

### タイムゾーンの設定
```sql
-- セッションのタイムゾーンをJSTに設定
SET TimeZone = 'Asia/Tokyo';
```

### 1. 基本的な読み込み
```sql
-- 特定プロジェクトの履歴を読み込む
SELECT * FROM read_json_auto(
    '/Users/yasuhisa.yoshida/.claude/projects/-Users-yasuhisa-yoshida-work-times-esa-mcp-server/*.jsonl'
);

-- 全プロジェクトの履歴を読み込む（glob pattern使用）
SELECT * FROM read_json_auto(
    '/Users/yasuhisa.yoshida/.claude/projects/**/*.jsonl',
    filename=true  -- ファイル名も取得
);
```

### 2. 作業したプロジェクトの一覧取得
```sql
-- UTCをJSTに変換して日付を判定
SELECT DISTINCT
    regexp_extract(filename, '.*/projects/([^/]+)/', 1) as project
FROM read_json_auto(
    '/Users/yasuhisa.yoshida/.claude/projects/**/*.jsonl',
    filename=true
)
WHERE type = 'user'
  -- UTC→JST変換（+9時間）して日付を判定
  AND (timestamp::TIMESTAMP AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Tokyo' >= '2025-05-31'::DATE
  AND (timestamp::TIMESTAMP AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Tokyo' < '2025-06-01'::DATE
ORDER BY project;
```

### 3. プロジェクト別の作業時間集計
```sql
WITH project_sessions AS (
    SELECT 
        regexp_extract(filename, '.*/projects/([^/]+)/', 1) as project,
        DATE_TRUNC('day', timestamp::TIMESTAMP) as work_date,
        timestamp::TIMESTAMP as ts,
        type,
        CASE 
            WHEN json_valid(message) THEN json_extract_string(message, '$.content')
            ELSE NULL
        END as content
    FROM read_json_auto('/Users/yasuhisa.yoshida/.claude/projects/**/*.jsonl', 
                        filename=true)
    WHERE type = 'user'
)
SELECT 
    work_date,
    project,
    COUNT(*) as messages,
    strftime(MIN(ts), '%H:%M') as start_time,
    strftime(MAX(ts), '%H:%M') as end_time,
    ROUND((EPOCH(MAX(ts)) - EPOCH(MIN(ts))) / 3600.0, 1) as hours
FROM project_sessions
WHERE work_date >= '2025-05-30'
  AND content IS NOT NULL
GROUP BY work_date, project
ORDER BY work_date DESC, messages DESC;
```

### 4. 作業内容をCSVファイルに出力

#### 方法1: 改行を含むデータに対応したCSV出力
```sql
-- 全プロジェクトの作業内容を一括で抽出してCSVに出力（適切なクォート処理付き）
COPY (
    WITH project_messages AS (
        SELECT 
            regexp_extract(filename, '.*/projects/([^/]+)/', 1) as project,
            -- UTC→JSTに変換してから時刻フォーマット
            strftime((timestamp::TIMESTAMP AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Tokyo', '%Y-%m-%d %H:%M:%S') as time,
            CASE 
                WHEN json_valid(message) THEN 
                    CASE
                        WHEN json_extract_string(message, '$.content[0].text') IS NOT NULL 
                        THEN json_extract_string(message, '$.content[0].text')
                        ELSE json_extract_string(message, '$.content')
                    END
                ELSE NULL
            END as content
        FROM read_json_auto(
            '/Users/yasuhisa.yoshida/.claude/projects/**/*.jsonl',
            filename=true
        )
        WHERE type = 'user'
          -- UTC→JST変換して日付判定
          AND (timestamp::TIMESTAMP AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Tokyo' >= '2025-05-31'::DATE
          AND (timestamp::TIMESTAMP AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Tokyo' < '2025-06-01'::DATE
          AND json_valid(message)
    )
    SELECT * FROM project_messages
    WHERE content IS NOT NULL
      AND content NOT LIKE '%<command-%'      -- コマンド関連を除外
      AND content NOT LIKE 'Caveat:%'         -- 警告メッセージを除外
      AND content NOT LIKE '[{"tool_use_id"%' -- ツール使用結果を除外
      AND LENGTH(content) > 20                -- 短すぎるメッセージを除外
    ORDER BY project, time
) TO '/Users/yasuhisa.yoshida/work/claude_work_summary/2025-05-31/all_projects.csv' (
    HEADER true,
    DELIMITER ',',
    QUOTE '"',           -- ダブルクォートで囲む
    ESCAPE '"',          -- エスケープ文字
    FORCE_QUOTE *        -- 全フィールドを強制的にクォート
);
```

#### 方法2: 改行を除去してからCSV出力
```sql
-- 改行を空白に置換してからCSV出力
COPY (
    WITH project_messages AS (
        SELECT 
            regexp_extract(filename, '.*/projects/([^/]+)/', 1) as project,
            strftime(timestamp::TIMESTAMP, '%Y-%m-%d %H:%M:%S') as time,
            -- 改行文字を空白に置換
            REPLACE(REPLACE(
                CASE 
                    WHEN json_valid(message) THEN 
                        CASE
                            WHEN json_extract_string(message, '$.content[0].text') IS NOT NULL 
                            THEN json_extract_string(message, '$.content[0].text')
                            ELSE json_extract_string(message, '$.content')
                        END
                    ELSE NULL
                END, 
                CHR(10), ' '),  -- LFを空白に
                CHR(13), ' '    -- CRを空白に
            ) as content
        FROM read_json_auto(
            '/Users/yasuhisa.yoshida/.claude/projects/**/*.jsonl',
            filename=true
        )
        WHERE type = 'user'
          AND timestamp::TIMESTAMP >= '2025-05-31'
          AND timestamp::TIMESTAMP < '2025-06-01'
          AND json_valid(message)
    )
    SELECT * FROM project_messages
    WHERE content IS NOT NULL
      AND content NOT LIKE '%<command-%'
      AND content NOT LIKE 'Caveat:%'
      AND content NOT LIKE '[{"tool_use_id"%'
      AND LENGTH(content) > 20
    ORDER BY project, time
) TO '/Users/yasuhisa.yoshida/work/claude_work_summary/2025-05-31/all_projects_clean.csv' (HEADER, DELIMITER ',');
```

#### 方法3: Parquet形式で出力（推奨）
```sql
-- Parquet形式なら改行や特殊文字の問題を回避できる
COPY (
    -- 同じWITH句を使用
    WITH project_messages AS (
        SELECT 
            regexp_extract(filename, '.*/projects/([^/]+)/', 1) as project,
            strftime(timestamp::TIMESTAMP, '%Y-%m-%d %H:%M:%S') as time,
            CASE 
                WHEN json_valid(message) THEN 
                    CASE
                        WHEN json_extract_string(message, '$.content[0].text') IS NOT NULL 
                        THEN json_extract_string(message, '$.content[0].text')
                        ELSE json_extract_string(message, '$.content')
                    END
                ELSE NULL
            END as content
        FROM read_json_auto(
            '/Users/yasuhisa.yoshida/.claude/projects/**/*.jsonl',
            filename=true
        )
        WHERE type = 'user'
          AND timestamp::TIMESTAMP >= '2025-05-31'
          AND timestamp::TIMESTAMP < '2025-06-01'
          AND json_valid(message)
    )
    SELECT * FROM project_messages
    WHERE content IS NOT NULL
      AND content NOT LIKE '%<command-%'
      AND content NOT LIKE 'Caveat:%'
      AND content NOT LIKE '[{"tool_use_id"%'
      AND LENGTH(content) > 20
    ORDER BY project, time
) TO '/Users/yasuhisa.yoshida/work/claude_work_summary/2025-05-31/all_projects.parquet' (FORMAT PARQUET);

-- Parquetファイルを読み込む場合
SELECT * FROM read_parquet('/Users/yasuhisa.yoshida/work/claude_work_summary/2025-05-31/all_projects.parquet');
```

### 5. 週次サマリーの生成
```sql
WITH weekly_work AS (
    SELECT 
        DATE_TRUNC('day', timestamp::TIMESTAMP) as work_date,
        regexp_extract(filename, '.*/projects/([^/]+)/', 1) as project,
        timestamp::TIMESTAMP as ts
    FROM read_json_auto(
        '/Users/yasuhisa.yoshida/.claude/projects/**/*.jsonl',
        filename=true
    )
    WHERE type = 'user'
      AND timestamp::TIMESTAMP >= CURRENT_DATE - INTERVAL 7 DAY
)
SELECT 
    strftime(work_date, '%Y-%m-%d (%a)') as date,
    COUNT(DISTINCT project) as projects_worked_on,
    COUNT(*) as total_messages,
    STRING_AGG(DISTINCT project, ', ' ORDER BY project) as projects
FROM weekly_work
GROUP BY work_date
ORDER BY work_date DESC;
```

## 実行手順

### 1. 作業ディレクトリの作成
```bash
mkdir -p ~/work/claude_work_summary/2025-05-31
```

### 2. DuckDBでデータを抽出
```bash
# claude codeのMCPツールを使用
mcp__duckdb__query
```

上記のSQLクエリ（特に「作業内容をCSVファイルに出力」）を実行。

### 3. 作業内容の要約
出力されたCSVファイルを読み込んで、各プロジェクトの作業内容を要約。

## 出力ファイル例

### CSVファイル（all_projects.csv）
```csv
project,time,content
-Users-yasuhisa-yoshida-work-times-esa-mcp-server,2025-05-31 05:15:23,ひとまずこれを見てくれる? https://github.com/syou6162/times_esa/pull/679
...
```

### 要約ファイル（work_summary.md）
プロジェクトごとに以下の情報をまとめる:
- 作業時間
- 主な作業内容
- 技術的な成果

## Tips

1. **message.contentの形式**
   - 文字列の場合: そのまま使用
   - 配列の場合: `[{"type":"text","text":"実際のメッセージ"}]`形式

2. **ノイズの除去**
   - `<command-`で始まる: CLIコマンド実行
   - `Caveat:`で始まる: システムメッセージ
   - `[{"tool_use_id"`で始まる: ツール使用結果

3. **glob patternの活用**
   - `/**/*.jsonl`: 全サブディレクトリを再帰的に検索
   - `/*times*/*.jsonl`: 特定のパターンにマッチするプロジェクトのみ

## 自動化の可能性
- 定期的にこのスクリプトを実行して日報を自動生成
- times_esa MCPサーバーと連携して自動投稿
- プロジェクトごとの作業時間レポートの生成