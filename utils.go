package main

import (
	"strings"
	"sync"
	"time"
	"unicode"
)

// debounce用の構造体
type debounceEntry struct {
	text      string
	timestamp time.Time
}

// debounceを管理するマップとミューテックス
var (
	debounceMap   = make(map[string]debounceEntry)
	debounceMutex sync.Mutex
	debounceTime  = 10 * time.Second
)

// テスト用にdebounceをリセットする関数
func resetDebounce() {
	debounceMutex.Lock()
	defer debounceMutex.Unlock()
	debounceMap = make(map[string]debounceEntry)
}

// isDebounced は指定されたテキストが短時間内に処理済みかチェックする
func isDebounced(text string) bool {
	debounceMutex.Lock()
	defer debounceMutex.Unlock()

	if entry, exists := debounceMap[text]; exists {
		if time.Since(entry.timestamp) < debounceTime {
			// 10秒以内の同一テキスト入力
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
		if time.Since(entry.timestamp) > debounceTime*2 {
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
