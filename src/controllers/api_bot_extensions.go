package controllers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	models "github.com/sufficit/sufficit-quepasa/models"
)

/*
<summary>
	Find a whatsapp server by token passed on Url Path parameters
</summary>
*/
func GetServer(r *http.Request) (server *models.QPWhatsappServer, err error) {
	token := GetToken(r)
	return models.GetServerFromToken(token)
}

//<summary>Find a whatsapp server by token passed on Url Path parameters</summary>
func GetServerRespondOnError(w http.ResponseWriter, r *http.Request) (server *models.QPWhatsappServer, err error) {
	token := chi.URLParam(r, "token")
	server, err = models.GetServerFromToken(token)
	if err != nil {
		RespondNotFound(w, fmt.Errorf("token '%s' not found", token))
	}
	return
}
