package main

import (
	"strings"
	"sync"
	"time"
	"unicode"
)

// debounce設定を管理する構造体
type DebounceConfig struct {
	Duration            time.Duration // debounceする時間
	SimilarityThreshold float64       // 類似度のしきい値（0.0〜1.0）
}

// debounce用の構造体
type debounceEntry struct {
	text      string
	timestamp time.Time
}

// debounceを管理するマップとミューテックス
var (
	debounceMap    = make(map[string]debounceEntry)
	debounceMutex  sync.Mutex
	debounceConfig = DebounceConfig{
		Duration:            10 * time.Second,
		SimilarityThreshold: 0.9, // デフォルトは90%以上の類似度でデバウンス
	}
)

// レーベンシュタイン距離を計算する関数
// s1, s2の2つの文字列間の編集距離を返す
// マルチバイト文字（日本語など）を正しく処理するためにrune単位で計算
func levenshteinDistance(s1, s2 string) int {
	// 文字列をrune（Unicode文字）のスライスに変換
	runes1 := []rune(s1)
	runes2 := []rune(s2)

	s1Len := len(runes1)
	s2Len := len(runes2)

	// 最適化: どちらかが空文字列なら、もう一方の長さが距離
	if s1Len == 0 {
		return s2Len
	}
	if s2Len == 0 {
		return s1Len
	}

	// 文字列が同一なら距離0
	if s1 == s2 {
		return 0
	}

	// 2次元配列の初期化
	matrix := make([][]int, s1Len+1)
	for i := range matrix {
		matrix[i] = make([]int, s2Len+1)
	}

	// 初期値設定
	for i := 0; i <= s1Len; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= s2Len; j++ {
		matrix[0][j] = j
	}

	// 距離計算
	for i := 1; i <= s1Len; i++ {
		for j := 1; j <= s2Len; j++ {
			cost := 1
			if runes1[i-1] == runes2[j-1] {
				cost = 0
			}
			// Go 1.21の標準min関数を使用
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // 削除
				matrix[i][j-1]+1,      // 挿入
				matrix[i-1][j-1]+cost, // 置換または一致
			)
		}
	}

	return matrix[s1Len][s2Len]
}

// テキスト類似度を計算する関数
// 0.0〜1.0の値を返す（1.0は完全一致、0.0は完全に異なる）
func textSimilarity(s1, s2 string) float64 {
	// 同一テキストなら類似度1.0
	if s1 == s2 {
		return 1.0
	}

	// どちらかが空文字列の場合
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// 文字列をruneのスライスに変換
	runes1 := []rune(s1)
	runes2 := []rune(s2)

	// 編集距離を計算
	distance := levenshteinDistance(s1, s2)

	// より厳格な類似度計算（完全に異なる場合は0に近くなるように調整）
	// 文字数（rune数）で計算
	maxAllowedDistance := max(len(runes1), len(runes2))

	// 編集距離が最大許容値を超える場合は0とする
	if distance >= maxAllowedDistance {
		return 0.0
	}

	// 距離から類似度へ変換
	return 1.0 - float64(distance)/float64(maxAllowedDistance)
}

// テスト用にdebounceをリセットする関数
func resetDebounce() {
	debounceMutex.Lock()
	defer debounceMutex.Unlock()
	debounceMap = make(map[string]debounceEntry)
}

// SetDebounceConfig はデバウンス設定を変更する関数
func SetDebounceConfig(duration time.Duration, similarityThreshold float64) {
	debounceMutex.Lock()
	defer debounceMutex.Unlock()

	debounceConfig.Duration = duration
	debounceConfig.SimilarityThreshold = similarityThreshold
}

// isDebounced は指定されたテキストが短時間内に処理済みかチェックする
// テキストの完全一致だけでなく、高い類似度を持つテキストもデバウンスする
func isDebounced(text string) bool {
	debounceMutex.Lock()
	defer debounceMutex.Unlock()

	// 空テキストは常にデバウンスする（処理させない）
	if text == "" {
		return true
	}

	// 完全一致チェック
	if entry, exists := debounceMap[text]; exists {
		if time.Since(entry.timestamp) < debounceConfig.Duration {
			// 設定時間以内の同一テキスト入力
			return true
		}
	}

	// 類似度チェック
	for storedText, entry := range debounceMap {
		// 有効期限内のエントリのみチェック
		if time.Since(entry.timestamp) < debounceConfig.Duration {
			// 両方のテキストが意味のある長さを持つ場合のみ類似度を計算
			if len(text) > 1 && len(storedText) > 1 {
				similarity := textSimilarity(text, storedText)

				if similarity >= debounceConfig.SimilarityThreshold {
					return true
				}
			}
		}
	}

	// エントリを追加
	debounceMap[text] = debounceEntry{
		text:      text,
		timestamp: time.Now(),
	}

	// マップのクリーンアップ（古いエントリを削除）
	for key, entry := range debounceMap {
		if time.Since(entry.timestamp) > debounceConfig.Duration*2 {
			delete(debounceMap, key)
		}
	}

	return false
}

// 指定したprefixで始まる場合に、prefix自体と、その直後の連続する空白類（Unicodeホワイトスペース）だけを除去し、他は一切変更しない
func stripPrefix(s string, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		return strings.TrimLeftFunc(s[len(prefix):], unicode.IsSpace)
	}
	return s
}
