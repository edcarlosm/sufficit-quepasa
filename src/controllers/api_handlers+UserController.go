package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	models "github.com/sufficit/sufficit-quepasa/models"
)

//region CONTROLLER - User

func UserController(w http.ResponseWriter, r *http.Request) {

	// setting default reponse type as json
	w.Header().Set("Content-Type", "application/json")

	response := &models.QpInfoResponse{}

	// reading body to avoid converting to json if empty
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// Declare a new Person struct.
	var user *models.QpUser

	if len(body) > 0 {

		// Try to decode the request body into the struct. If there is an error,
		// respond to the client with the error message and a 400 status code.
		err = json.Unmarshal(body, &user)
		if err != nil {
			jsonError := fmt.Errorf("error converting body to json: %v", err.Error())
			response.ParseError(jsonError)
			RespondInterface(w, response)
			return
		}
	}

	// creating an empty webhook, to filter or clear it all
	if user == nil || len(user.Username) == 0 {
		jsonErr := fmt.Errorf("invalid user body: %s", err.Error())
		response.ParseError(jsonErr)
		RespondInterface(w, response)
		return
	}

	// searching user
	user, err = models.WhatsappService.DB.Users.Find(user.Username)
	if err != nil {
		jsonError := fmt.Errorf("user not found: %v", err.Error())
		response.ParseError(jsonError)
		RespondInterface(w, response)
		return
	}

	server, err := GetServer(r)
	if err != nil {
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	server.User = user.Username
	err = server.Save()
	if err != nil {
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	response.PatchSuccess(server, "server attached for user: "+user.Username)
	RespondSuccess(w, response)
	return
}

//endregion
