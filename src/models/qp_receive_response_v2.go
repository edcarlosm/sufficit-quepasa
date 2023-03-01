package models

type QpReceiveResponseV2 struct {
	QpResponse
	Messages []QpMessageV2 `json:"messages"`
	Bot      QPBotV2       `json:"bot"`
}
