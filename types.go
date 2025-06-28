package main

type PostDailyReportArgs struct {
	Text string `json:"text"`
}

type PostDailyReportResult struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Post    EsaPost `json:"post"`
}

type QuickPostArgs struct {
	Text string `json:"text"`
}

