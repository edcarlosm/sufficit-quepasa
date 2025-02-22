package whatsapp

import (
	"strings"
	"time"
)

// Mensagem no formato QuePasa
// Utilizada na API do QuePasa para troca com outros sistemas
type WhatsappMessage struct {

	// original message from source service
	Content interface{} `json:"-"`

	Id      string `json:"id"`
	TrackId string `json:"trackid,omitempty"` // Optional id of the system that send that message

	Timestamp time.Time           `json:"timestamp"`
	Type      WhatsappMessageType `json:"type"`

	// Em qual chat (grupo ou direct) essa msg foi postada, para onde devemos responder
	Chat WhatsappChat `json:"chat"`

	// Se a msg foi postado em algum grupo ? quem postou !
	Participant *WhatsappEndpoint `json:"participant,omitempty"`

	// Texto da msg
	Text string `json:"text"`

	Attachment *WhatsappAttachment `json:"attachment,omitempty"`

	// Do i send that ?
	// From any connected device and api
	FromMe bool `json:"fromme"`

	// Sended via api
	FromInternal bool `json:"frominternal"`

	// Quantas vezes essa msg foi encaminhada
	ForwardingScore uint32 `json:"forwardingscore,omitempty"`

	// Msg in reply of another ? Message ID
	InReply string `json:"inreply,omitempty"`
}

//region ORDER BY TIMESTAMP

type ByTimestamp []WhatsappMessage

func (m ByTimestamp) Len() int           { return len(m) }
func (m ByTimestamp) Less(i, j int) bool { return m[i].Timestamp.After(m[j].Timestamp) }
func (m ByTimestamp) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }

//endregion

//region IMPLEMENT WHATSAPP SEND RESPONSE INTERFACE

func (source WhatsappMessage) GetID() string { return source.Id }

// Get the time of server processed message
func (source WhatsappMessage) GetTime() time.Time { return source.Timestamp }

// Get the time on unix timestamp format
func (source WhatsappMessage) GetTimestamp() uint64 { return uint64(source.Timestamp.Unix()) }

//endregion

func (source *WhatsappMessage) GetChatId() string {
	return source.Chat.ID
}

func (source *WhatsappMessage) GetText() string {
	return source.Text
}

func (source *WhatsappMessage) HasAttachment() bool {
	// this attachment is a pointer to correct show info on deserialized
	attach := source.Attachment
	return attach != nil && len(attach.Mimetype) > 0
}

func (source *WhatsappMessage) GetSource() interface{} {
	return source.Content
}

func (source *WhatsappMessage) FromGroup() bool {
	return strings.HasSuffix(source.Chat.ID, "@g.us")
}

func (source *WhatsappMessage) FromBroadcast() bool {
	return source.Chat.ID == "status" || source.Chat.ID == "status@broadcast"
}

func (source *WhatsappMessage) GetAttachment() *WhatsappAttachment {
	return source.Attachment
}
