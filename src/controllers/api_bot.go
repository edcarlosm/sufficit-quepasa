package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	metrics "github.com/sufficit/sufficit-quepasa/metrics"
	models "github.com/sufficit/sufficit-quepasa/models"
	whatsapp "github.com/sufficit/sufficit-quepasa/whatsapp"
)

// ReceiveAPIHandler renders route GET "/{version}/bot/{token}/receive"
func ReceiveAPIHandler(w http.ResponseWriter, r *http.Request) {
	response := &models.QpReceiveResponse{}

	server, err := GetServer(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// Evitando tentativa de download de anexos sem o bot estar devidamente sincronizado
	status := server.GetStatus()
	if status != whatsapp.Ready {
		metrics.MessageReceiveErrors.Inc()
		err = &ApiServerNotReadyException{Wid: server.GetWid(), Status: status}
		response.ParseError(err)
		RespondInterfaceCode(w, response, http.StatusServiceUnavailable)
		return
	}

	queryValues := r.URL.Query()
	paramTimestamp := queryValues.Get("timestamp")
	timestamp, err := GetTimestamp(paramTimestamp)
	if err != nil {
		metrics.MessageReceiveErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	response.Total = uint64(server.Handler.GetTotal())

	messages := GetMessages(server, timestamp)
	metrics.MessagesReceived.Add(float64(len(messages)))

	response.Bot = *server.Bot
	response.Messages = messages

	if timestamp > 0 {
		response.ParseSuccess(fmt.Sprintf("getting with timestamp: %v", timestamp))
	} else {
		response.ParseSuccess("getting without filter")
	}
	RespondInterface(w, response)
}

// SendAPIHandler renders route "/v3/bot/{token}/send"
func SendAny(w http.ResponseWriter, r *http.Request) {
	response := &models.QpSendResponse{}

	server, err := GetServer(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// Declare a new request struct.
	request := &models.QpSendAnyRequest{}

	// Getting ChatId parameter
	err = request.EnsureValidChatId(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	switch os := r.Method; os {
	case http.MethodPost:
		// Try to decode the request body into the struct. If there is an error,
		// respond to the client with the error message and a 400 status code.
		err = json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			jsonErr := fmt.Errorf("invalid json body: %s", err.Error())
			response.ParseError(jsonErr)
			RespondInterface(w, response)
			return
		}

	case http.MethodGet:
		if r.URL.Query().Has("text") {
			request.Text = r.URL.Query().Get("text")
		}

		if r.URL.Query().Has("url") {
			request.Url = r.URL.Query().Get("url")
		}
	}

	// override trackid if passed throw any other way
	trackid := GetTrackId(r)
	if len(trackid) > 0 {
		request.TrackId = trackid
	}

	if len(request.Url) > 0 {
		// base 64 content to byte array
		err = request.GenerateUrlContent()
		if err != nil {
			metrics.MessageSendErrors.Inc()
			response.ParseError(err)
			RespondInterface(w, response)
			return
		}

		SendDocument(server, response, &request.QpSendRequest, w)
	} else if len(request.Content) > 0 {
		// base 64 content to byte array
		err = request.GenerateEmbbedContent()
		if err != nil {
			metrics.MessageSendErrors.Inc()
			response.ParseError(err)
			RespondInterface(w, response)
			return
		}

		SendDocument(server, response, &request.QpSendRequest, w)
	} else {
		// text msg

		if len(request.Text) == 0 {
			metrics.MessageSendErrors.Inc()
			err = fmt.Errorf("text not found, do not send empty messages")
			response.ParseError(err)
			RespondInterface(w, response)
			return
		}

		Send(server, response, &request.QpSendRequest, w, nil)
	}
}

// SendAPIHandler renders route "/v3/bot/{token}/sendtext"
func SendText(w http.ResponseWriter, r *http.Request) {
	response := &models.QpSendResponse{}

	server, err := GetServer(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// Declare a new request struct.
	request := &models.QpSendRequest{}

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err = json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	if len(request.Text) == 0 {
		metrics.MessageSendErrors.Inc()
		err = fmt.Errorf("text not found, do not send empty messages")
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// Getting ChatId parameter
	err = request.EnsureValidChatId(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// override trackid if passed throw any other way
	trackid := GetTrackId(r)
	if len(trackid) > 0 {
		request.TrackId = trackid
	}
	Send(server, response, request, w, nil)
}

/*
<summary>
	Renders route POST "/{version}/bot/{token}/sendbinary/{chatid}/{fileName}/{text}"

	Any of then, at this order of priority
	Path parameters: {chatid}
	Path parameters: {fileName}
	Path parameters: {text} only images
	Url parameters: ?chatid={chatId}
	Url parameters: ?fileName={fileName}
	Url parameters: ?text={text} only images
	Header parameters: X-QUEPASA-CHATID = {chatId}
	Header parameters: X-QUEPASA-FILENAME = {fileName}
	Header parameters: X-QUEPASA-TEXT = {text} only images
</summary>
*/
func SendDocumentFromBinary(w http.ResponseWriter, r *http.Request) {
	response := &models.QpSendResponse{}

	server, err := GetServer(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// Declare a new request struct.
	request := &models.QpSendRequest{}

	// Getting ChatId parameter
	err = request.EnsureValidChatId(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	content, err := io.ReadAll(r.Body)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(fmt.Errorf("attachment content missing or read error"))
		RespondInterface(w, response)
		return
	}

	request.Content = content

	// Getting FileName parameter
	fileName := chi.URLParam(r, "fileName")
	if len(fileName) == 0 && r.URL.Query().Has("fileName") {
		fileName = r.URL.Query().Get("fileName")
	} else if len(fileName) == 0 {
		fileName = r.Header.Get("X-QUEPASA-FILENAME")
	}

	// Setting filename
	request.FileName = fileName

	// Getting textLabel parameter
	text := chi.URLParam(r, "text")
	if len(text) == 0 && r.URL.Query().Has("text") {
		text = r.URL.Query().Get("text")
	} else if len(text) == 0 {
		text = r.Header.Get("X-QUEPASA-TEXT")
	}

	request.Text = text

	// override trackid if passed throw any other way
	trackid := GetTrackId(r)
	if len(trackid) > 0 {
		request.TrackId = trackid
	}

	SendDocument(server, response, request, w)
}

/*
<summary>
	Renders route POST "/{version}/bot/{token}/sendencoded"

	Body parameter: {chatId}
	Body parameter: {fileName}
	Body parameter: {text} only images
	Body parameter: {content}
</summary>
*/
func SendDocumentFromEncoded(w http.ResponseWriter, r *http.Request) {
	response := &models.QpSendResponse{}

	server, err := GetServer(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// Declare a new request struct.
	request := &models.QpSendRequestEncoded{}

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// Getting ChatId parameter
	err = request.EnsureValidChatId(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// base 64 content to byte array
	err = request.GenerateContent()
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// override trackid if passed throw any other way
	trackid := GetTrackId(r)
	if len(trackid) > 0 {
		request.TrackId = trackid
	}
	SendDocument(server, response, &request.QpSendRequest, w)
}

/*
<summary>
	Renders route POST "/{version}/bot/{token}/sendurl"

	Body parameter: {url}
	Body parameter: {chatId}
	Body parameter: {fileName}
	Body parameter: {text} only images
</summary>
*/
func SendDocumentFromUrl(w http.ResponseWriter, r *http.Request) {
	response := &models.QpSendResponse{}

	server, err := GetServer(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// Declare a new request struct.
	request := &models.QpSendRequestUrl{}

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// Getting ChatId parameter
	err = request.EnsureValidChatId(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// url download content to byte array
	err = request.GenerateContent()
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// override trackid if passed throw any other way
	trackid := GetTrackId(r)
	if len(trackid) > 0 {
		request.TrackId = trackid
	}

	SendDocument(server, response, &request.QpSendRequest, w)
}

func Send(server *models.QPWhatsappServer, response *models.QpSendResponse, request *models.QpSendRequest, w http.ResponseWriter, attach *whatsapp.WhatsappAttachment) {
	waMsg, err := request.ToWhatsappMessage()
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	if attach != nil {
		waMsg.Attachment = attach
		waMsg.Type = whatsapp.GetMessageType(attach.Mimetype)
		server.Log.Debugf("send attachment of type: %v and mime: %s and length: %v and filename: %s", waMsg.Type, attach.Mimetype, attach.FileLength, attach.FileName)
	} else {
		waMsg.Type = whatsapp.TextMessageType
	}

	sendResponse, err := server.SendMessage(waMsg)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// success
	metrics.MessagesSent.Inc()

	result := &models.QpSendResponseMessage{}
	result.Wid = server.GetWid()
	result.Id = sendResponse.GetID()
	result.ChatId = waMsg.Chat.ID
	result.TrackId = waMsg.TrackId

	response.ParseSuccess(result)
	RespondInterface(w, response)
}

func SendDocument(server *models.QPWhatsappServer, response *models.QpSendResponse, request *models.QpSendRequest, w http.ResponseWriter) {
	attach, err := request.ToWhatsappAttachment()
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	Send(server, response, request, w, attach)
}
