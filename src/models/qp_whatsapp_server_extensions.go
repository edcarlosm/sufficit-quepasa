package models

import (
	"crypto/tls"
	"errors"
	"net/http"

	whatsapp "github.com/sufficit/sufficit-quepasa/whatsapp"
)

// Encaminha msg ao WebHook específicado
func PostToWebHookFromServer(server *QPWhatsappServer, message *whatsapp.WhatsappMessage) (err error) {
	wid := server.GetWid()

	// Ignorando certificado ao realizar o post
	// Não cabe a nós a segurança do cliente
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	for _, element := range server.Webhooks {
		if !message.FromInternal || (element.ForwardInternal && (len(element.TrackId) == 0 || element.TrackId != message.TrackId)) {
			element.Post(wid, message)
		}
	}

	return
}

//region FIND|SEARCH WHATSAPP SERVER
var ErrServerNotFound error = errors.New("the requested whatsapp server was not found")

func GetServerFromID(source string) (server *QPWhatsappServer, err error) {
	server, ok := WhatsappService.Servers[source]
	if !ok {
		err = ErrServerNotFound
		return
	}
	return
}

func GetServerFromBot(source QPBot) (server *QPWhatsappServer, err error) {
	return GetServerFromID(source.ID)
}

func GetServerFromToken(token string) (server *QPWhatsappServer, err error) {
	for _, item := range WhatsappService.Servers {
		if item.Bot != nil && item.Bot.Token == token {
			server = item
			break
		}
	}
	if server == nil {
		err = ErrServerNotFound
	}
	return
}

func GetServersForUserID(userid string) (servers map[string]*QPWhatsappServer) {
	return WhatsappService.GetServersForUser(userid)
}

func GetServersForUser(user QPUser) (servers map[string]*QPWhatsappServer) {
	return GetServersForUserID(user.ID)
}

//endregion
