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

// Renders route GET "/{version}/bot/{token}/receive"
func ReceiveAPIHandlerV2(w http.ResponseWriter, r *http.Request) {

	// setting default reponse type as json
	w.Header().Set("Content-Type", "application/json")

	server, err := GetServer(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		RespondServerError(server, w, err)
		return
	}

	// Evitando tentativa de download de anexos sem o bot estar devidamente sincronizado
	status := server.GetStatus()
	if status != whatsapp.Ready {
		RespondNotReady(w, &ApiServerNotReadyException{Wid: server.GetWid(), Status: status})
		return
	}

	queryValues := r.URL.Query()
	timestamp := queryValues.Get("timestamp")

	messages, err := GetMessagesToAPIV2(server, timestamp)
	if err != nil {
		metrics.MessageReceiveErrors.Inc()
		RespondServerError(server, w, err)
		return
	}

	metrics.MessagesReceived.Add(float64(len(messages)))

	out := models.QPFormReceiveResponseV2{
		Bot:      *models.ToQPBotV2(server.Bot),
		Messages: messages,
	}

	RespondSuccess(w, out)
}

// SendAPIHandler renders route "/v2/bot/{token}/send"
func SendTextAPIHandlerV2(w http.ResponseWriter, r *http.Request) {

	// setting default reponse type as json
	w.Header().Set("Content-Type", "application/json")

	server, err := GetServer(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		RespondServerError(server, w, err)
		return
	}

	// Declare a new Person struct.
	var request models.QPSendRequest

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		RespondServerError(server, w, err)
		return
	}

	log.Tracef("sending requested: %v", request)
	trackid := GetTrackId(r)
	waMsg, err := whatsapp.ToMessage(request.Recipient, request.Message, trackid)
	if err != nil {
		metrics.MessageSendErrors.Inc()
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
		RespondServerError(server, w, err)
		return
	}

	response := models.QPSendResponseV2{}
	response.Chat.ID = waMsg.Chat.Id
	response.Chat.UserName = waMsg.Chat.Id
	response.Chat.Title = waMsg.Chat.Title
	response.From.ID = server.Bot.ID
	response.From.UserName = server.Bot.GetNumber()
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

	response := models.QPSendResponseV2{}
	response.Chat.ID = waMsg.Chat.Id
	response.Chat.UserName = waMsg.Chat.Id
	response.Chat.Title = server.GetTitle(waMsg.Chat.Id)
	response.From.ID = server.Bot.ID
	response.From.UserName = server.Bot.GetNumber()
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
		RespondNotFound(w, fmt.Errorf("Token '%s' not found", token))
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
