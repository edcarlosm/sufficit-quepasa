package models

import (
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	whatsapp "github.com/sufficit/sufficit-quepasa/whatsapp"
)

type QpSendRequest struct {
	// (Optional) Used if passed
	Id string `json:"id,omitempty"`

	// Recipient of this message
	ChatId string `json:"chatId"`

	// (Optional) TrackId - less priority (urlparam -> query -> header -> body)
	TrackId string `json:"trackId,omitempty"`

	Text string `json:"text,omitempty"`

	// (Optional) Sugested filename on user download
	FileName string `json:"fileName,omitempty"`

	Content []byte
}

func (source *QpSendRequest) EnsureChatId(r *http.Request) (err error) {
	if len(source.ChatId) == 0 {
		source.ChatId = GetChatId(r)
	}

	if len(source.ChatId) == 0 {
		err = fmt.Errorf("chat id missing")
	}
	return
}

func (source *QpSendRequest) EnsureValidChatId(r *http.Request) (err error) {
	err = source.EnsureChatId(r)
	if err != nil {
		return
	}

	chatid, err := whatsapp.FormatEndpoint(source.ChatId)
	if err != nil {
		return
	}

	source.ChatId = chatid
	return
}

func (source *QpSendRequest) ToWhatsappMessage() (msg *whatsapp.WhatsappMessage, err error) {
	chatId, err := whatsapp.FormatEndpoint(source.ChatId)
	if err != nil {
		return
	}

	chat := whatsapp.WhatsappChat{ID: chatId}
	msg = &whatsapp.WhatsappMessage{
		Id:           source.Id,
		TrackId:      source.TrackId,
		Text:         source.Text,
		Chat:         chat,
		FromMe:       true,
		FromInternal: true,
	}

	// setting default type
	if len(msg.Text) > 0 {
		msg.Type = whatsapp.TextMessageType
	}

	return
}

func (source *QpSendRequest) ToWhatsappAttachment() (attach *whatsapp.WhatsappAttachment, err error) {
	attach = &whatsapp.WhatsappAttachment{}

	mimeType := http.DetectContentType(source.Content)
	if mimeType == "application/octet-stream" && len(source.FileName) > 0 {
		extension := filepath.Ext(source.FileName)
		newMimeType := mime.TypeByExtension(extension)
		if len(newMimeType) > 0 {
			mimeType = newMimeType
		}
	}

	log.Tracef("detected mime type: %s, filename: %s", mimeType, source.FileName)

	fileName := source.FileName
	// Defining a filename if not found before
	if len(fileName) == 0 {
		const layout = "20060201150405"
		t := time.Now().UTC()
		fileName = "file-" + t.Format(layout)

		// get file extension from mime type
		extension, _ := mime.ExtensionsByType(mimeType)
		if len(extension) > 0 {
			fileName = fileName + extension[0]
		}
	}

	if strings.Contains(mimeType, "ogg") && !strings.Contains(mimeType, "opus") {
		mimeType = "audio/ogg; codecs=opus"
	}

	attach.FileName = fileName
	attach.FileLength = uint64(len(source.Content))
	attach.Mimetype = mimeType
	attach.SetContent(&source.Content)
	return
}
