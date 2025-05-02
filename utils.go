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
func levenshteinDistance(s1, s2 string) int {
	s1Len := len(s1)
	s2Len := len(s2)

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
			if s1[i-1] == s2[j-1] {
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
func isDebounced(text string) bool {
	debounceMutex.Lock()
	defer debounceMutex.Unlock()

	if entry, exists := debounceMap[text]; exists {
		if time.Since(entry.timestamp) < debounceConfig.Duration {
			// 設定時間以内の同一テキスト入力
			return true
		}
	}

	// エントリを更新または追加
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
