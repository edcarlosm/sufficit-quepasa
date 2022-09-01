package controllers

import (
	"html/template"
	"net/http"

	. "github.com/sufficit/sufficit-quepasa/metrics"
	. "github.com/sufficit/sufficit-quepasa/models"
	. "github.com/sufficit/sufficit-quepasa/whatsapp"
)

// FormReceiveController renders route GET "/bot/{botID}/receive"
func FormReceiveController(w http.ResponseWriter, r *http.Request) {
	data := QPFormReceiveData{PageTitle: "Receive", FormAccountEndpoint: FormAccountEndpoint}

	server, err := GetServerFromRequest(r)
	if err != nil {
		data.ErrorMessage = err.Error()
	} else {
		data.Number = server.GetWid()
		data.Token = server.Bot.Token
		data.DownloadPrefix = GetDownloadPrefix(server.Bot.Token)
	}

	// Evitando tentativa de download de anexos sem o bot estar devidamente sincronizado
	status := server.GetStatus()
	if status != Ready {
		RespondNotReady(w, &ApiServerNotReadyException{Wid: server.GetWid(), Status: status})
		return
	}

	queryValues := r.URL.Query()
	paramTimestamp := queryValues.Get("timestamp")
	timestamp, err := GetTimestamp(paramTimestamp)
	if err != nil {
		MessageReceiveErrors.Inc()
		RespondServerError(server, w, err)
		return
	}

	messages := GetMessages(server, timestamp)
	data.Messages = messages

	MessagesReceived.Add(float64(len(messages)))

	templates := template.Must(template.ParseFiles(
		"views/layouts/main.tmpl",
		"views/bot/receive.tmpl"))
	templates.ExecuteTemplate(w, "main", data)
}
