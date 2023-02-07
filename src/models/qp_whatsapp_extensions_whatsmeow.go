package models

import (
	"strings"

	log "github.com/sirupsen/logrus"

	whatsapp "github.com/sufficit/sufficit-quepasa/whatsapp"
	whatsmeow "github.com/sufficit/sufficit-quepasa/whatsmeow"
)

func NewWhatsmeowEmptyConnection() (whatsapp.IWhatsappConnection, error) {
	return whatsmeow.WhatsmeowService.CreateEmptyConnection()
}

func NewWhatsmeowConnection(wid string, logger *log.Logger) (whatsapp.IWhatsappConnection, error) {
	return whatsmeow.WhatsmeowService.CreateConnection(wid, logger)
}

func ToQPMessageV2(source whatsapp.WhatsappMessage, wid string) (message QPMessageV2) {
	message.ID = source.Id
	message.Timestamp = uint64(source.Timestamp.Unix())
	message.Text = source.Text
	message.FromMe = source.FromMe

	message.Controller = QPEndpointV2{}
	if !strings.Contains(wid, "@") {
		message.Controller.ID = wid + "@c.us"
	} else {
		message.Controller.ID = wid
	}

	message.ReplyTo = ChatToQPEndPointV2(source.Chat)
	message.Chat = ChatToQPChatV2(source.Chat)

	if source.Participant != nil {
		message.Participant = ChatToQPEndPointV2(*source.Participant)
	}

	if source.HasAttachment() {
		message.Attachment = ToQPAttachmentV1(source.Attachment, message.ID, wid)
	}

	if len(source.InReply) > 0 {
		message.Text = "*(IN REPLY) " + message.Text
	}

	if source.ForwardingScore > 0 {
		message.Text = "*(FORWARDED) " + message.Text
	}

	return
}

func ToQPMessageV1(source whatsapp.WhatsappMessage, wid string) (message QPMessageV1) {
	message.ID = source.Id
	message.Timestamp = uint64(source.Timestamp.Unix())
	message.Text = source.Text
	message.FromMe = source.FromMe

	message.Controller = QPEndpointV1{}
	if !strings.Contains(wid, "@") {
		message.Controller.ID = wid + "@c.us"
	} else {
		message.Controller.ID = wid
	}

	message.ReplyTo = ChatToQPEndPointV1(source.Chat)

	if source.Participant != nil {
		message.Participant = ChatToQPEndPointV1(*source.Participant)
		//message.Participant = ToQPEndPointV1(source.Participant)
	}

	if source.HasAttachment() {
		message.Attachment = ToQPAttachmentV1(source.Attachment, message.ID, wid)
	}

	return
}
