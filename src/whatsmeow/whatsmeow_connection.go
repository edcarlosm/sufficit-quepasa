package whatsmeow

import (
	"context"
	"fmt"
	"sync"
	"unicode"

	log "github.com/sirupsen/logrus"

	whatsapp "github.com/sufficit/sufficit-quepasa/whatsapp"
	whatsmeow "go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	types "go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// Must Implement IWhatsappConnection
type WhatsmeowConnection struct {
	Client      *whatsmeow.Client
	Handlers    *WhatsmeowHandlers
	waLogger    waLog.Logger
	logger      *log.Logger
	log         *log.Entry
	failedToken bool
}

//region IMPLEMENT INTERFACE WHATSAPP CONNECTION

func (conn *WhatsmeowConnection) GetVersion() string { return "multi" }

func (conn *WhatsmeowConnection) GetWid() (wid string, err error) {
	if conn.Client == nil {
		err = fmt.Errorf("client not defined on trying to get wid")
	} else {
		if conn.Client.Store == nil {
			err = fmt.Errorf("device store not defined on trying to get wid")
		} else {
			if conn.Client.Store.ID == nil {
				err = fmt.Errorf("device id not defined on trying to get wid")
			} else {
				wid = conn.Client.Store.ID.User
			}
		}
	}

	return
}

func (conn *WhatsmeowConnection) GetStatus() (state whatsapp.WhatsappConnectionState) {
	if conn != nil {
		state = whatsapp.Created
		if conn.Client != nil {
			if conn.Client.IsConnected() {
				state = whatsapp.Connected
				if conn.Client.IsLoggedIn() {
					state = whatsapp.Ready
				}
			} else {
				state = whatsapp.Disconnected
				if conn.failedToken {
					state = whatsapp.Failed
				}
			}
		}
	}
	return
}

// Retorna algum titulo válido apartir de um jid
func (conn *WhatsmeowConnection) GetTitle(Wid string) string {
	jid := types.NewJID(Wid, "")
	var result string
	contact, err := conn.Client.Store.Contacts.GetContact(jid)
	if err == nil {
		result = contact.PushName
	}

	return result
}

// Connect to websocket only, dot not authenticate yet, errors come after
func (conn *WhatsmeowConnection) Connect() (err error) {
	conn.log.Info("starting whatsmeow connection")

	err = conn.Client.Connect()
	if err != nil {
		conn.failedToken = true
		return
	}

	if !conn.Client.IsLoggedIn() {
		conn.failedToken = true
		conn.log.Warn("starting whatsmeow connection, connected but not logged")
		return
	}

	conn.failedToken = false
	return
}

// func (cli *Client) Download(msg DownloadableMessage) (data []byte, err error)
func (conn *WhatsmeowConnection) DownloadData(imsg whatsapp.IWhatsappMessage) (data []byte, err error) {
	msg := imsg.GetSource()
	downloadable, ok := msg.(whatsmeow.DownloadableMessage)
	if !ok {
		conn.log.Debug("not downloadable, trying default message")
		waMsg, ok := msg.(*waProto.Message)
		if !ok {
			attach := imsg.GetAttachment()
			if attach != nil {
				data := attach.GetContent()
				if data != nil {
					return *data, err
				}
			}

			err = fmt.Errorf("parameter msg cannot be converted to an original message")
			return
		}
		return conn.Client.DownloadAny(waMsg)
	}
	return conn.Client.Download(downloadable)
}

func (conn *WhatsmeowConnection) Download(imsg whatsapp.IWhatsappMessage) (att *whatsapp.WhatsappAttachment, err error) {
	data, err := conn.DownloadData(imsg)
	if err != nil {
		return
	}

	att = &whatsapp.WhatsappAttachment{}
	att.SetContent(&data)

	sourceAttach := imsg.GetAttachment()
	att.FileName = sourceAttach.FileName

	return
}

func (conn *WhatsmeowConnection) GetProfilePicture(wid string, knowingId string) (picture *whatsapp.WhatsappProfilePicture, err error) {
	jid, err := types.ParseJID(wid)
	if err != nil {
		return
	}

	pictureInfo, err := conn.Client.GetProfilePictureInfo(jid, false, knowingId)
	if err != nil {
		return
	}

	if pictureInfo != nil {
		picture = &whatsapp.WhatsappProfilePicture{
			Id:   pictureInfo.ID,
			Type: pictureInfo.Type,
			Url:  pictureInfo.URL,
		}
	}
	return
}

func isASCII(s string) bool {
	for _, c := range s {
		if c > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// Default SEND method using WhatsappMessage Interface
func (conn *WhatsmeowConnection) Send(msg *whatsapp.WhatsappMessage) (whatsapp.IWhatsappSendResponse, error) {

	var err error
	messageText := msg.GetText()

	var newMessage *waProto.Message
	if !msg.HasAttachment() {
		internal := &waProto.ExtendedTextMessage{Text: &messageText}
		newMessage = &waProto.Message{ExtendedTextMessage: internal}
	} else {
		newMessage, err = conn.UploadAttachment(*msg)
		if err != nil {
			return msg, err
		}
	}

	// Formatting destination accordly
	formatedDestination, _ := whatsapp.FormatEndpoint(msg.GetChatId())

	// Avoid common issue with incorrect non ascii chat id
	if !isASCII(formatedDestination) {
		err = fmt.Errorf("not an ASCII formated chat id")
		return msg, err
	}

	jid, err := types.ParseJID(formatedDestination)
	if err != nil {
		conn.log.Infof("send error on get jid: %s", err)
		return msg, err
	}

	// Generating a new unique MessageID
	if len(msg.Id) == 0 {
		msg.Id = whatsmeow.GenerateMessageID()
	}

	resp, err := conn.Client.SendMessage(context.Background(), jid, msg.Id, newMessage)
	if err != nil {
		conn.log.Infof("send error: %s", err)
		return msg, err
	}
	msg.Timestamp = resp.Timestamp

	conn.log.Infof("send: %s, on: %s", msg.Id, msg.Timestamp)
	return msg, err
}

// func (cli *Client) Upload(ctx context.Context, plaintext []byte, appInfo MediaType) (resp UploadResponse, err error)
func (conn *WhatsmeowConnection) UploadAttachment(msg whatsapp.WhatsappMessage) (result *waProto.Message, err error) {

	content := *msg.Attachment.GetContent()
	if len(content) == 0 {
		err = fmt.Errorf("null or empty content")
		return
	}

	mediaType := GetMediaTypeFromString(msg.Attachment.Mimetype)
	response, err := conn.Client.Upload(context.Background(), content, mediaType)
	if err != nil {
		return
	}

	result = NewWhatsmeowMessageAttachment(response, msg.Attachment, mediaType)
	return
}

func (conn *WhatsmeowConnection) Disconnect() (err error) {
	if conn.Client != nil {
		if conn.Client.IsConnected() {
			conn.Client.Disconnect()
		}
	}
	return
}

func (conn *WhatsmeowConnection) Delete() (err error) {
	if conn != nil {
		if conn.Client != nil {
			if conn.Client.IsLoggedIn() {
				err = conn.Client.Logout()
				if err != nil {
					return
				}
			}

			if conn.Client.IsConnected() {
				conn.Client.Disconnect()
			}

			if conn.Client.Store != nil {
				err = conn.Client.Store.Delete()
				return
			}
		}
	}
	return
}

func (conn *WhatsmeowConnection) GetInvite(groupId string) (link string, err error) {
	jid, err := types.ParseJID(groupId)
	if err != nil {
		conn.log.Infof("getting invite error on parse jid: %s", err)
	}

	link, err = conn.Client.GetGroupInviteLink(jid, false)
	return
}

func (conn *WhatsmeowConnection) GetWhatsAppQRChannel(result chan<- string) (err error) {

	// No ID stored, new login
	qrChan, _ := conn.Client.GetQRChannel(context.Background())
	err = conn.Client.Connect()
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)

	for evt := range qrChan {
		if evt.Event == "code" {
			result <- evt.Code
		} else {
			wg.Done()
			break
		}
	}

	wg.Wait()
	close(result)
	return
}

func (conn *WhatsmeowConnection) UpdateLog(entry *log.Entry) {
	conn.log = entry
}

func (conn *WhatsmeowConnection) UpdateHandler(handlers whatsapp.IWhatsappHandlers) {
	conn.Handlers.WAHandlers = handlers
}

//endregion

func (conn *WhatsmeowConnection) EnsureHandlers() error {
	return nil
}

/*
<summary>
	Disconnect if connected
	Cleanup Handlers
</summary>
*/
func (conn *WhatsmeowConnection) Dispose() {
	if conn.logger != nil {
		conn.logger.Warnf("disposing connection ...")
		conn.logger = nil
	}

	if conn.Handlers != nil {
		conn.Handlers.UnRegister()
		conn.Handlers = nil
	}

	if conn.Client != nil {
		if conn.Client.IsConnected() {
			conn.Client.Disconnect()
		}
		conn.Client = nil
	}

	conn = nil
}

func (conn *WhatsmeowConnection) IsInterfaceNil() bool {
	return nil == conn
}
