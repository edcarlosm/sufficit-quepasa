package models

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
	whatsapp "github.com/sufficit/sufficit-quepasa/whatsapp"
)

type QPWhatsappServer struct {
	QpServerWebhookCollection
	Bot            *QPBot                       `json:"bot"`
	connection     whatsapp.IWhatsappConnection `json:"-"`
	syncConnection *sync.Mutex                  `json:"-"` // Objeto de sinaleiro para evitar chamadas simultâneas a este objeto
	syncMessages   *sync.Mutex                  `json:"-"` // Objeto de sinaleiro para evitar chamadas simultâneas a este objeto
	Battery        WhatsAppBateryStatus         `json:"battery,omitempty"`
	Timestamp      time.Time                    `json:"starttime,omitempty"`
	Handler        *QPWhatsappHandlers          `json:"-"`

	stopRequested bool        `json:"-"`
	logger        *log.Logger `json:"-"`
	Log           *log.Entry  `json:"-"`
}

//region CONSTRUCTORS

// Instanciando um novo servidor para controle de whatsapp
func NewQPWhatsappServer(bot *QPBot, dbWHooks *QpDataWebhookInterface) (server *QPWhatsappServer, err error) {
	wid := bot.ID
	var serverLogLevel log.Level
	if bot.Devel {
		serverLogLevel = log.DebugLevel
	} else {
		serverLogLevel = log.InfoLevel
	}

	serverLogger := log.New()
	serverLogger.SetLevel(serverLogLevel)
	serverLogEntry := serverLogger.WithField("wid", wid)

	handler := NewQPWhatsappHandlers(bot.HandleGroups, bot.HandleBroadcast, serverLogEntry)
	server = &QPWhatsappServer{
		Bot:            bot,
		syncConnection: &sync.Mutex{},
		syncMessages:   &sync.Mutex{},
		Battery:        WhatsAppBateryStatus{},
		Timestamp:      time.Now(),
		Handler:        handler,

		stopRequested: false,
		logger:        serverLogger,
		Log:           serverLogEntry,
	}

	server.WebhookFill(wid, *dbWHooks)
	return
}

//endregion
//region IMPLEMENT OF INTERFACE STATE RECOVERY

func (server *QPWhatsappServer) GetStatus() whatsapp.WhatsappConnectionState {
	if server.connection != nil {
		return server.connection.GetStatus()
	} else if server.stopRequested {
		return whatsapp.Stopped
	} else {
		return whatsapp.Created
	}
}

//endregion
//region IMPLEMENT OF INTERFACE QUEPASA SERVER

// Returns whatsapp controller id on E164
// Ex: 5521967609095
func (server *QPWhatsappServer) GetWid() string {
	return server.Bot.ID
}

func (server *QPWhatsappServer) DownloadData(id string) ([]byte, error) {
	msg, err := server.Handler.GetMessage(id)
	if err != nil {
		return nil, err
	}

	server.Log.Infof("downloading msg data %s", id)
	return server.connection.DownloadData(&msg)
}

func (server *QPWhatsappServer) Download(id string) (att *whatsapp.WhatsappAttachment, err error) {
	msg, err := server.Handler.GetMessage(id)
	if err != nil {
		return
	}

	server.Log.Infof("downloading msg %s", id)
	att, err = server.connection.Download(&msg)
	if err != nil {
		return
	}

	return
}

//endregion

func (server *QPWhatsappServer) GetMessages(timestamp time.Time) (messages []whatsapp.WhatsappMessage) {
	messages = append(messages, server.Handler.GetMessages(timestamp)...)
	return
}

// Roda de forma assíncrona, não interessa o resultado ao chamador
// Inicia o processo de tentativas de conexão de um servidor individual
func (server *QPWhatsappServer) Initialize() {
	if server == nil {
		panic("nil server, code error")
	}

	server.Log.Info("initializing whatsapp server ...")
	err := server.Start()
	if err != nil {
		server.Log.Errorf("initializing server error: %s", err.Error())
	}
}

// Update underlying connection and ensure trivials
func (server *QPWhatsappServer) UpdateConnection(connection whatsapp.IWhatsappConnection) {

	if server.connection != nil && !server.connection.IsInterfaceNil() {
		server.connection.Dispose()
	}

	server.connection = connection
	server.connection.UpdateLog(server.Log)
	if server.Handler == nil {
		server.logger.Info("creating handlers ?!")
	}

	server.connection.UpdateHandler(server.Handler)

	// Registrando webhook
	webhookDispatcher := &QPWebhookHandler{Server: server}
	if !server.Handler.IsAttached() {
		server.Handler.Register(webhookDispatcher)
	}

	server.connection.EnsureHandlers()
}

func (server *QPWhatsappServer) EnsureUnderlying() (err error) {
	server.syncConnection.Lock()

	// conectar dispositivo
	if server.connection == nil {
		server.Log.Infof("trying to create new whatsapp connection ...")
		wid := server.GetWid()
		log := server.logger

		var connection whatsapp.IWhatsappConnection
		connection, err = NewWhatsmeowConnection(wid, log)
		if err != nil {
			waError, ok := err.(whatsapp.WhatsappError)
			if ok {
				if waError.Unauthorized() {
					server.MarkVerified(false)
				}
			}

		} else {
			server.connection = connection
		}
	}

	server.syncConnection.Unlock()
	return
}

func (server *QPWhatsappServer) Start() (err error) {
	err = server.EnsureUnderlying()
	if err != nil {
		return
	}

	state := server.GetStatus()
	server.Log.Infof("starting whatsapp server ... on %s state", state)

	if !IsValidToStart(state) {
		err = fmt.Errorf("trying to start a server on an invalid state :: %s", state)
		server.Log.Warnf(err.Error())
		return
	}

	// reset stop requested token
	server.stopRequested = false

	// Registrando webhook
	webhookDispatcher := &QPWebhookHandler{Server: server}
	server.Handler.Register(webhookDispatcher)

	// Atualizando manipuladores de eventos
	server.connection.UpdateHandler(server.Handler)

	server.Log.Infof("requesting connection ...")
	err = server.connection.Connect()
	if err != nil {
		if _, ok := err.(*whatsapp.UnAuthorizedError); ok {
			server.Log.Warningf("unauthorized, setting unverified")
			err = server.MarkVerified(false)
		}
		return
	}

	// If at this moment the connect is already logged, ensure a valid mark
	if server.connection.IsValid() {
		server.MarkVerified(true)
	}

	return
}

func (server *QPWhatsappServer) Stop(cause string) (err error) {
	if !server.stopRequested {
		// setting token
		server.stopRequested = true

		if server.connection != nil {
			server.connection.Dispose()
			server.connection = nil
		}
		server.Handler.Clear()
	}

	return
}

func (server *QPWhatsappServer) Restart() {
	server.Log.Info("restart requested ....")
	// Somente executa caso não esteja em estado de processo de conexão
	// Evita chamadas simultâneas desnecessárias
	/*
		if server.Status != Starting {
			server.Status = Restarting
			server.log.Infof("Restarting WhatsApp Server ...")

			server.Connection.Disconnect()
			server.Status = Disconnected

			// Inicia novamente o servidor e os Handlers(alças)
			err := server.Initialize()
			if err != nil {
				server.Status = Failed
				server.log.Infof("(ERR) Critical error on WhatsApp Server: %s", err.Error())
			}
		}
	*/
}

// Somente usar em caso de não ser permitida a reconxão automática
func (server *QPWhatsappServer) Disconnect(cause string) {
	server.Log.Infof("Disconnecting WhatsApp Server: %s", cause)

	if server.connection != nil {
		server.connection.Disconnect()
	}
}

// Retorna o titulo em cache (se houver) do id passado em parametro
func (server *QPWhatsappServer) GetTitle(Wid string) string {
	return server.connection.GetTitle(Wid)
}

// Usado para exibir os servidores/bots de cada usuario em suas respectivas telas
func (server *QPWhatsappServer) GetOwnerID() string {
	return server.Bot.UserID
}

//region QP BOT EXTENSIONS

// Check if the current connection state is valid for a start call
func IsValidToStart(status whatsapp.WhatsappConnectionState) bool {
	if status == whatsapp.Stopped {
		return true
	}
	if status == whatsapp.Disconnected {
		return true
	}
	if status == whatsapp.Failed {
		return true
	}
	return false
}

func (server *QPWhatsappServer) GetWorking() bool {
	status := server.GetStatus()
	if status <= whatsapp.Stopped {
		return false
	} else if status == whatsapp.Disconnected {
		return false
	}
	return true
}

func (server *QPWhatsappServer) GetStatusString() string {
	return server.GetStatus().String()
}

func (server *QPWhatsappServer) ID() string {
	return server.Bot.ID
}

// Traduz o Wid para um número de telefone em formato E164
func (server *QPWhatsappServer) GetNumber() string {
	return server.Bot.GetNumber()
}

func (server *QPWhatsappServer) GetTimestamp() (timestamp uint64) {
	return server.Bot.GetTimestamp()
}

func (server *QPWhatsappServer) GetStartedTime() (timestamp time.Time) {
	return server.Bot.GetStartedTime()
}

func (server *QPWhatsappServer) GetBatteryInfo() WhatsAppBateryStatus {
	return server.Bot.GetBatteryInfo()
}

func (server *QPWhatsappServer) GetConnection() whatsapp.IWhatsappConnection {
	return server.connection
}

func (server *QPWhatsappServer) Toggle() (err error) {
	if !server.GetWorking() {
		err = server.Start()
	} else {
		err = server.Stop("toggling")
	}
	return
}

func (server *QPWhatsappServer) IsDevelopmentGlobal() bool {
	return ENV.IsDevelopment()
}

func (server *QPWhatsappServer) CycleToken() (err error) {
	value := uuid.New().String()
	err = server.Bot.UpdateToken(value)
	if err != nil {
		return
	}

	server.Log.Infof("cycling token: %v", value)
	return
}

func (server *QPWhatsappServer) Token() string {
	return server.Bot.Token
}

func (server *QPWhatsappServer) MarkVerified(value bool) (err error) {
	err = server.Bot.UpdateVerified(value)
	if err != nil {
		return
	}

	server.Log.Infof("updating verified status: %v", value)
	return
}

func (server *QPWhatsappServer) Verified() bool {
	return server.Bot.Verified
}

func (server *QPWhatsappServer) ToggleGroups() (err error) {
	err = server.Bot.UpdateGroups(!server.Bot.HandleGroups)
	if err != nil {
		return
	}

	server.Handler.HandleGroups = server.Bot.HandleGroups
	server.Log.Infof("toggling handler of group messages: %v", server.Handler.HandleGroups)
	return
}

func (server *QPWhatsappServer) HandleGroups() bool {
	return server.Bot.HandleGroups
}

func (server *QPWhatsappServer) ToggleBroadcast() (err error) {
	err = server.Bot.UpdateBroadcast(!server.Bot.HandleBroadcast)
	if err != nil {
		return
	}

	server.Handler.HandleBroadcast = server.Bot.HandleBroadcast

	server.Log.Infof("toggling handler of broadcast messages: %v", server.Handler.HandleBroadcast)
	return
}

func (server *QPWhatsappServer) HandleBroadcast() bool {
	return server.Bot.HandleBroadcast
}

func (server *QPWhatsappServer) ToggleDevel() (err error) {
	err = server.Bot.UpdateDevel(!server.Bot.Devel)
	if err != nil {
		return
	}

	if server.Bot.Devel {
		server.logger.SetLevel(log.DebugLevel)
	} else {
		server.logger.SetLevel(log.InfoLevel)
	}

	server.Log.Infof("toggling logger level: %v", server.logger.Level)
	return
}

func (server *QPWhatsappServer) Devel() bool {
	return server.Bot.Devel
}

func (server *QPWhatsappServer) SetVersion(value string) error {
	return server.Bot.UpdateVersion(value)
}

func (server *QPWhatsappServer) Version() string {
	return server.Bot.Version
}

//endregion

func (server *QPWhatsappServer) Delete() (err error) {
	if server.connection != nil {
		err = server.connection.Delete()
		if err != nil {
			return
		}
	}

	return server.Bot.Delete()
}

//endregion

//#region SEND

// Default send message method
func (server *QPWhatsappServer) SendMessage(msg *whatsapp.WhatsappMessage) (response whatsapp.IWhatsappSendResponse, err error) {
	server.Log.Debugf("sending msg to: %v", msg.Chat.ID)

	if msg.HasAttachment() {
		if len(msg.Text) > 0 {

			// Overriding filename with caption text if IMAGE or VIDEO
			if msg.Type == whatsapp.ImageMessageType || msg.Type == whatsapp.VideoMessageType {
				msg.Attachment.FileName = msg.Text
			} else {

				// Copying and send text before file
				textMsg := *msg
				textMsg.Type = whatsapp.TextMessageType
				textMsg.Attachment = nil
				_, err = server.connection.Send(&textMsg)
				if err == nil {
					server.Handler.Message(&textMsg)
				}
			}
		}
	}

	// sending default msg
	response, err = server.connection.Send(msg)
	if err == nil {
		server.Handler.Message(msg)
	}
	return
}

//#endregion
//#region PROFILE PICTURE

func (server *QPWhatsappServer) GetProfilePicture(wid string, knowingId string) (picture *whatsapp.WhatsappProfilePicture, err error) {
	server.Log.Debugf("getting info about profile picture for: %s, with id: %s", wid, knowingId)

	return server.connection.GetProfilePicture(wid, knowingId)
}

//#endregion
//#region GROUP INVITE LINK

func (server *QPWhatsappServer) GetInvite(groupId string) (link string, err error) {
	return server.connection.GetInvite(groupId)
}

//#endregion
