package whatsmeow

import (
	"fmt"
	"reflect"

	log "github.com/sirupsen/logrus"
	whatsapp "github.com/sufficit/sufficit-quepasa/whatsapp"
	whatsmeow "go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

type WhatsmeowHandlers struct {
	Client                   *whatsmeow.Client
	WAHandlers               whatsapp.IWhatsappHandlers
	eventHandlerID           uint32
	unregisterRequestedToken bool
	log                      *log.Entry
}

func (handler *WhatsmeowHandlers) UnRegister() {
	handler.unregisterRequestedToken = true

	// if is this session
	found := handler.Client.RemoveEventHandler(handler.eventHandlerID)
	if found {
		handler.log.Infof("handler unregistered, id: %v", handler.eventHandlerID)
	}
}

func (handler *WhatsmeowHandlers) Register() (err error) {
	if handler.Client.Store == nil {
		err = fmt.Errorf("this client lost the store, probably a logout from whatsapp phone")
		return
	}

	handler.unregisterRequestedToken = false
	handler.eventHandlerID = handler.Client.AddEventHandler(handler.EventsHandler)
	handler.log.Infof("handler registered, id: %v", handler.eventHandlerID)

	return
}

// Define os diferentes tipos de eventos a serem reconhecidos
// Aqui se define se vamos processar mensagens | confirmações de leitura | etc
func (handler *WhatsmeowHandlers) EventsHandler(evt interface{}) {
	if handler.unregisterRequestedToken {
		handler.log.Info("unregister event handler requested")
		handler.Client.RemoveEventHandler(handler.eventHandlerID)
		return
	}

	switch v := evt.(type) {

	case *events.Message:
		go handler.Message(*v)
		return

	case *events.Connected:
		// zerando contador de tentativas de reconexão
		// importante para zerar o tempo entre tentativas em caso de erro
		handler.Client.AutoReconnectErrors = 0
		return

	case *events.LoggedOut:
		handler.UnRegister()
		return

	case
		*events.AppState,
		*events.AppStateSyncComplete,
		*events.Contact,
		*events.DeleteChat,
		*events.DeleteForMe,
		*events.HistorySync,
		*events.MarkChatAsRead,
		*events.Mute,
		*events.OfflineSyncCompleted,
		*events.OfflineSyncPreview,
		*events.PairSuccess,
		*events.Pin,
		*events.PushName,
		*events.PushNameSetting,
		*events.QR,
		*events.Receipt:
		return // ignoring not implemented yet

	default:
		handler.log.Debugf("event not handled: %v", reflect.TypeOf(v))
		return
	}
}

//region EVENT MESSAGE

// Aqui se processar um evento de recebimento de uma mensagem genérica
func (handler *WhatsmeowHandlers) Message(evt events.Message) {
	handler.log.Trace("event Message !")
	if evt.Message == nil {
		handler.log.Error("nil message on receiving whatsmeow events | try use rawMessage !")
		return
	}

	message := &whatsapp.WhatsappMessage{Content: evt.Message}

	// basic information
	message.Id = evt.Info.ID
	message.Timestamp = evt.Info.Timestamp
	message.FromMe = evt.Info.IsFromMe

	message.Chat = whatsapp.WhatsappChat{}
	chatID := fmt.Sprint(evt.Info.Chat.User, "@", evt.Info.Chat.Server)
	message.Chat.ID = chatID

	if evt.Info.IsGroup {
		gInfo, _ := handler.Client.GetGroupInfo(evt.Info.Chat)
		if gInfo != nil {
			message.Chat.Title = gInfo.Name
		}

		message.Participant = &whatsapp.WhatsappEndpoint{}

		participantID := fmt.Sprint(evt.Info.Sender.User, "@", evt.Info.Sender.Server)
		message.Participant.ID = participantID
		message.Participant.Title = evt.Info.PushName
	} else if !message.FromMe {
		message.Chat.Title = evt.Info.PushName
	}

	// Process diferent message types
	HandleKnowingMessages(handler, message, evt.Message)
	if message.Type == whatsapp.UnknownMessageType {
		HandleUnknownMessage(handler.log, evt)
	}

	if handler.WAHandlers != nil {

		// following to internal handlers
		go handler.WAHandlers.Message(message)
	}
}

//endregion
