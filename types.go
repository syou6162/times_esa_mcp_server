package main

type TimesEsaPostRequest struct {
	Text string `json:"text"`
}

type TimesEsaPostResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Post    EsaPost `json:"post"`
}
