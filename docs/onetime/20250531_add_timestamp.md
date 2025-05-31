# タイムスタンプへのアンカーリンク追加

## 概要

[times_esa PR #679](https://github.com/syou6162/times_esa/pull/679) と同様に、日報投稿時のタイムスタンプにアンカーリンクを追加する変更を実装する。これにより、esa上で特定の時刻の投稿に直接リンクできるようになる。

## 変更内容

### 変更前の形式
```
12:34 投稿内容
```

### 変更後の形式
```html
<a id="1234" href="#1234">12:34</a> 投稿内容
```

## 実装チェックリスト

### 1. esa_client.go の修正

#### CreatePost メソッド（140-141行目）
- [ ] 現在のコード:
  ```go
  timePrefix := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
  reqBody.Post.BodyMd = fmt.Sprintf("%s %s\n\n---", timePrefix, text)
  ```
- [ ] 変更後:
  ```go
  timeStr := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
  anchorId := fmt.Sprintf("%02d%02d", now.Hour(), now.Minute())
  timePrefix := fmt.Sprintf("<a id=\"%s\" href=\"#%s\">%s</a>", anchorId, anchorId, timeStr)
  reqBody.Post.BodyMd = fmt.Sprintf("%s %s\n\n---", timePrefix, text)
  ```

#### UpdatePost メソッド（201-204行目）
- [ ] 現在のコード:
  ```go
  timePrefix := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
  reqBody.Post.BodyMd = fmt.Sprintf("%s %s\n\n---\n\n%s", timePrefix, text, existingPost.BodyMd)
  ```
- [ ] 変更後:
  ```go
  timeStr := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
  anchorId := fmt.Sprintf("%02d%02d", now.Hour(), now.Minute())
  timePrefix := fmt.Sprintf("<a id=\"%s\" href=\"#%s\">%s</a>", anchorId, anchorId, timeStr)
  reqBody.Post.BodyMd = fmt.Sprintf("%s %s\n\n---\n\n%s", timePrefix, text, existingPost.BodyMd)
  ```

### 2. タイムスタンプ生成関数の作成

- [ ] utils.go に共通関数を追加
  ```go
  // GenerateTimestampWithAnchor は時刻をアンカーリンク付きで生成する
  func GenerateTimestampWithAnchor(t time.Time) string {
      timeStr := fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
      anchorId := fmt.Sprintf("%02d%02d", t.Hour(), t.Minute())
      return fmt.Sprintf("<a id=\"%s\" href=\"#%s\">%s</a>", anchorId, anchorId, timeStr)
  }
  ```

### 3. 既存投稿の読み込み時の互換性確保

- [ ] 既存のプレーンテキスト形式（`HH:MM`）も読み込めることを確認
- [ ] アンカー付き形式も正しく解析できることを確認

### 4. テストケースの更新

- [ ] handlers_test.go の更新
  - 新規投稿時のアンカーリンク確認
  - 既存投稿への追記時のアンカーリンク確認
- [ ] タイムスタンプ形式の確認テストを追加

### 5. エンドツーエンドテスト

- [ ] 実際のesa.ioでの表示確認
- [ ] アンカーリンクが正しく機能することの確認
- [ ] 既存の日報との互換性確認

## 技術的な考慮事項

1. **アンカーIDの重複**
   - 同じ分に複数回投稿した場合、アンカーIDが重複する
   - 秒数は含めない（エディタ上で編集時に邪魔になるため）
   - **対処方針**: 同じ分に複数回投稿があった場合は諦める（最初の投稿のみリンクが有効）

2. **既存投稿との互換性**
   - 既にある日報への追記時に、既存のタイムスタンプ形式を壊さない

3. **HTMLエスケープ**
   - **不要**: マークダウン内にHTMLタグ（details、summaryなど）を書くことは通常の用途
   - 自分専用のアプリケーションのため、セキュリティ上の懸念は不要

## 参考資料

- [times_esa PR #679](https://github.com/syou6162/times_esa/pull/679)
- esa.io APIドキュメント