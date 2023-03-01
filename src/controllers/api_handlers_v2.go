package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
	metrics "github.com/sufficit/sufficit-quepasa/metrics"
	models "github.com/sufficit/sufficit-quepasa/models"
	whatsapp "github.com/sufficit/sufficit-quepasa/whatsapp"
)

const APIVersion2 string = "v2"

var ControllerPrefixV2 string = fmt.Sprintf("/%s/bot/{token}", APIVersion2)

func RegisterAPIV2Controllers(r chi.Router) {

	r.Get(ControllerPrefixV2, InformationHandlerV2)
	r.Post(ControllerPrefixV2+"/send", SendAPIHandlerV2)
	r.Get(ControllerPrefixV2+"/receive", ReceiveAPIHandlerV2)

	// external for now
	r.Post(ControllerPrefixV2+"/senddocument", SendDocumentAPIHandlerV2)
	r.Post(ControllerPrefixV2+"/attachment", AttachmentAPIHandlerV2)
}

// InformationController renders route GET "/{version}/bot/{token}"
func InformationHandlerV2(w http.ResponseWriter, r *http.Request) {

	response := &models.QpInfoResponseV2{}

	server, err := GetServer(r)
	if err != nil {
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	response.Id = server.Token
	response.Number = server.GetNumber()
	RespondInterface(w, response)
}

// SendAPIHandler renders route "/{version}/bot/{token}/send"
func SendAPIHandlerV2(w http.ResponseWriter, r *http.Request) {
	response := &models.QpSendResponseV2{}

	server, err := GetServer(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// Declare a new request struct.
	request := &models.QpSendRequestV2{}

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	log.Tracef("sending requested: %v", request)
	trackid := GetTrackId(r)
	waMsg, err := whatsapp.ToMessage(request.Recipient, request.Message, trackid)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// setting source msg participant
	if waMsg.FromGroup() && len(waMsg.Participant.Id) == 0 {
		waMsg.Participant.Id = whatsapp.PhoneToWid(server.GetWid())
	}

	// setting wa msg chat title
	if len(waMsg.Chat.Title) == 0 {
		waMsg.Chat.Title = server.GetTitle(waMsg.Chat.Id)
	}

	sendResponse, err := server.SendMessage(waMsg)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	response.Chat.ID = waMsg.Chat.Id
	response.Chat.UserName = waMsg.Chat.Id
	response.Chat.Title = waMsg.Chat.Title
	response.From.ID = server.WId
	response.From.UserName = server.GetNumber()
	response.ID = sendResponse.GetId()

	// Para manter a compatibilidade
	response.PreviusV1 = models.QPSendResult{
		Source:    server.GetWid(),
		Recipient: waMsg.Chat.Id,
		MessageId: sendResponse.GetId(),
	}

	metrics.MessagesSent.Inc()
	RespondInterface(w, response)
}

// Renders route GET "/{version}/bot/{token}/receive"
func ReceiveAPIHandlerV2(w http.ResponseWriter, r *http.Request) {
	response := models.QpReceiveResponseV2{}

	server, err := GetServer(r)
	if err != nil {
		metrics.MessageReceiveErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// append server to response
	response.Bot = *models.ToQpServerV2(server.QpServer)

	// Evitando tentativa de download de anexos sem o bot estar devidamente sincronizado
	status := server.GetStatus()
	if status != whatsapp.Ready {
		err = &ApiServerNotReadyException{Wid: server.WId, Status: status}
		metrics.MessageReceiveErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	queryValues := r.URL.Query()
	timestamp := queryValues.Get("timestamp")

	messages, err := GetMessagesToAPIV2(server, timestamp)
	if err != nil {
		metrics.MessageReceiveErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// append messages to response
	response.Messages = messages

	// metrics
	metrics.MessagesReceived.Add(float64(len(messages)))
	RespondInterface(w, response)
}

// NOT TESTED ----------------------------------
// NOT TESTED ----------------------------------
// NOT TESTED ----------------------------------
// NOT TESTED ----------------------------------

// Usado para envio de documentos, anexos, separados do texto, em caso de imagem, aceita um caption (titulo)
func SendDocumentAPIHandlerV2(w http.ResponseWriter, r *http.Request) {

	// setting default reponse type as json
	w.Header().Set("Content-Type", "application/json")

	server, err := GetServerRespondOnError(w, r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		return
	}

	// Declare a new Person struct.
	var request models.QPSendDocumentRequestV2

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		RespondServerError(server, w, err)
		return
	}

	if request.Attachment == (models.QPAttachmentV1{}) {
		metrics.MessageSendErrors.Inc()
		RespondServerError(server, w, fmt.Errorf("attachment not found"))
		return
	}

	trackid := GetTrackId(r)
	waMsg, err := whatsapp.ToMessage(request.Recipient, request.Message, trackid)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		return
	}

	attach, err := models.ToWhatsappAttachment(&request.Attachment)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		RespondServerError(server, w, err)
		return
	}

	waMsg.Attachment = attach
	waMsg.Type = whatsapp.GetMessageType(attach.Mimetype)

	sendResponse, err := server.SendMessage(waMsg)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		RespondServerError(server, w, err)
		return
	}

	response := models.QpSendResponseV2{}
	response.Chat.ID = waMsg.Chat.Id
	response.Chat.UserName = waMsg.Chat.Id
	response.Chat.Title = server.GetTitle(waMsg.Chat.Id)
	response.From.ID = server.WId
	response.From.UserName = server.GetNumber()
	response.ID = sendResponse.GetId()

	// Para manter a compatibilidade
	response.PreviusV1 = models.QPSendResult{
		Source:    server.GetWid(),
		Recipient: waMsg.Chat.Id,
		MessageId: sendResponse.GetId(),
	}

	metrics.MessagesSent.Inc()
	RespondSuccess(w, response)
}

// AttachmentHandler renders route POST "/v1/bot/{token}/attachment"
func AttachmentAPIHandlerV2(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	server, err := models.GetServerFromToken(token)
	if err != nil {
		RespondNoContent(w, fmt.Errorf("Token '%s' not found", token))
		return
	}

	// Evitando tentativa de download de anexos sem o bot estar devidamente sincronizado
	status := server.GetStatus()
	if status != whatsapp.Ready {
		RespondNotReady(w, &ApiServerNotReadyException{Wid: server.GetWid(), Status: status})
		return
	}

	// Declare a new Person struct.
	var p models.QPAttachmentV1

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err = json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		RespondServerError(server, w, err)
	}

	ss := strings.Split(p.Url, "/")
	id := ss[len(ss)-1]

	att, err := server.Download(id, false)
	if err != nil {
		RespondServerError(server, w, err)
		return
	}

	if len(att.FileName) > 0 {
		w.Header().Set("Content-Disposition", "attachment; filename="+att.FileName)
	}

	if len(att.Mimetype) > 0 {
		w.Header().Set("Content-Type", att.Mimetype)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(*att.GetContent())
}
