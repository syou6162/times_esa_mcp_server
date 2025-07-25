package main

type TimesEsaPostRequest struct {
	Text            string `json:"text"`
	ConfirmedByUser bool   `json:"confirmed_by_user"`
}

type TimesEsaPostResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Post    EsaPost `json:"post"`
}
