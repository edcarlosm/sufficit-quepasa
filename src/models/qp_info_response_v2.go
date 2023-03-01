package models

type QpInfoResponseV2 struct {
	QpResponse
	Id     string `json:"id"`
	Number string `json:"number"`
}
