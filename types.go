package main

import "time"

type PostDailyReportArgs struct {
	Text string `json:"text" jsonschema:"required,description=投稿するテキスト内容"`
}

type PostDailyReportResult struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Post    EsaPost `json:"post"`
}

type QuickPostArgs struct {
	Text string `json:"text" jsonschema:"description=投稿するテキスト内容"`
}

type QuickPostResult struct {
	Description string          `json:"description"`
	Messages    []PromptMessage `json:"messages"`
}

type PromptMessage struct {
	Role    string      `json:"role"`
	Content TextContent `json:"content"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
