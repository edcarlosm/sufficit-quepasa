package controllers

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	models "github.com/sufficit/sufficit-quepasa/models"
)

func RedirectToLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, FormLoginEndpoint, http.StatusFound)
}

// Google chrome bloqueou wss, portanto retornaremos sempre ws apatir de agora
func WebSocketProtocol() string {
	protocol := "ws"
	isSecure, _ := models.GetEnvBool("WEBSOCKETSSL", false)
	if isSecure {
		protocol = "wss"
	}

	return protocol
}

//
// Cycle
//

// CycleHandler renders route POST "/bot/cycle"
func FormCycleController(w http.ResponseWriter, r *http.Request) {
	_, server, err := GetUserAndServer(w, r)
	if err != nil {
		// retorno já tratado pela funcao
		return
	}

	err = server.CycleToken()
	if err != nil {
		RespondServerError(server, w, err)
		return
	}

	http.Redirect(w, r, FormAccountEndpoint, http.StatusFound)
}

// DebugHandler renders route POST "/bot/debug"
func FormDebugController(w http.ResponseWriter, r *http.Request) {
	_, server, err := GetUserAndServer(w, r)
	if err != nil {
		// retorno já tratado pela funcao
		return
	}

	err = server.ToggleDevel()
	if err != nil {
		RespondServerError(server, w, err)
		return
	}

	http.Redirect(w, r, FormAccountEndpoint, http.StatusFound)
}

// ToggleHandler renders route POST "/bot/toggle"
func FormToggleController(w http.ResponseWriter, r *http.Request) {
	_, server, err := GetUserAndServer(w, r)
	if err != nil {
		// retorno já tratado pela funcao
		return
	}

	response := &models.QpResponse{}

	err = server.Toggle()
	if err != nil {
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	http.Redirect(w, r, FormAccountEndpoint, http.StatusFound)
}

// ToggleHandler renders route POST "/bot/toggle"
func FormToggleBroadcastController(w http.ResponseWriter, r *http.Request) {
	_, server, err := GetUserAndServer(w, r)
	if err != nil {
		// retorno já tratado pela funcao
		return
	}

	err = server.ToggleBroadcast()
	if err != nil {
		RespondServerError(server, w, err)
		return
	}

	http.Redirect(w, r, FormAccountEndpoint, http.StatusFound)
}

func FormToggleGroupsController(w http.ResponseWriter, r *http.Request) {
	_, server, err := GetUserAndServer(w, r)
	if err != nil {
		// retorno já tratado pela funcao
		return
	}

	err = server.ToggleGroups()
	if err != nil {
		RespondServerError(server, w, err)
		return
	}

	http.Redirect(w, r, FormAccountEndpoint, http.StatusFound)
}

//
// Verify
//

// VerifyFormHandler renders route GET "/bot/verify" ?mode={sd|md}
func VerifyFormHandler(w http.ResponseWriter, r *http.Request) {
	data := models.QPFormVerifyData{
		PageTitle:   "Verify To Add or Update",
		Protocol:    WebSocketProtocol(),
		Host:        r.Host,
		Destination: FormAccountEndpoint,
	}

	templates := template.Must(template.ParseFiles(
		"views/layouts/main.tmpl",
		"views/bot/verify.tmpl",
	))
	templates.ExecuteTemplate(w, "main", data)
}

var done chan interface{}
var interrupt chan os.Signal
var upgrader = websocket.Upgrader{}

// VerifyHandler renders route GET "/bot/verify/ws"
func VerifyHandler(w http.ResponseWriter, r *http.Request) {
	user, err := models.GetUser(r)
	if err != nil {
		log.Print("Connection upgrade error (not logged): ", err)
		RedirectToLogin(w, r)
		return
	}

	done = make(chan interface{})          // Channel to indicate that the receiverHandler is done
	interrupt = make(chan os.Signal)       // Channel to listen for interrupt signal to terminate gracefully
	signal.Notify(interrupt, os.Interrupt) // Notify the interrupt channel for SIGINT

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Connection upgrade error: ", err)
		return
	}

	defer conn.Close()
	go receiveWebSocketHandler(user, conn)

	// Our main loop for the client
	// We send our relevant packets here
	for {
		select {
		case <-time.After(time.Duration(1) * time.Millisecond * 1000):
			// Send an echo packet every second
			err := conn.WriteMessage(websocket.TextMessage, []byte("echo"))
			if err != nil {
				//log.Println("Error during writing to websocket:", err)
				return
			}

		case <-interrupt:
			// We received a SIGINT (Ctrl + C). Terminate gracefully...
			log.Println("Received SIGINT interrupt signal. Closing all pending connections")

			// Close our websocket connection
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Error during closing websocket:", err)
				return
			}

			select {
			case <-done:
				log.Println("Receiver Channel Closed! Exiting....")
			case <-time.After(time.Duration(1) * time.Second):
				log.Println("Timeout in closing receiving channel. Exiting....")
			}
			return
		}
	}
}

func receiveWebSocketHandler(user models.QPUser, connection *websocket.Conn) {
	defer close(done)
	for {
		mt, msg, err := connection.ReadMessage()
		if err != nil {
			log.Errorf("error in receive: %s", err.Error())
			return
		}

		if strings.EqualFold(string(msg), "start:sd") || strings.EqualFold(string(msg), "start:md") {
			out := make(chan []byte)
			go func() {
				defer close(out)
				err = connection.WriteMessage(mt, <-out)
				if err != nil {
					log.Println("Write message error: ", err)
				}
			}()

			multidevice := strings.EqualFold(string(msg), "start:md")

			// Exibindo código QR
			err := models.SignInWithQRCode(user, multidevice, out)
			if err != nil {
				if strings.Contains(err.Error(), "timed out") {
					err = connection.WriteMessage(mt, []byte("timeout"))
					if err != nil {
						// log.Println("Write message error after timeout: ", err)
						return
					}
				} else {
					log.Println("SignInWithQRCode Unknown Error:", err)
				}
			} else {
				err = connection.WriteMessage(websocket.TextMessage, []byte("complete"))
				if err != nil {
					log.Errorf("error on write complete message after qrcode verified: %s", err.Error())
				}
				return
			}
		} else {
			log.Warnf("received unknown msg from websocket: %s", msg)
		}
	}
}

//
// Delete
//

// DeleteHandler renders route POST "/form/delete"
func FormDeleteController(w http.ResponseWriter, r *http.Request) {
	_, server, err := GetUserAndServer(w, r)
	if err != nil {
		// retorno já tratado pela funcao
		return
	}

	server.Log.Warnf("delete requested by form !")
	if err := models.WhatsappService.Delete(server); err != nil {
		RespondServerError(server, w, err)
		return
	}

	http.Redirect(w, r, FormAccountEndpoint, http.StatusFound)
}

//
// Helpers
//

// Facilitador que traz usuario e servidor para quem esta autenticado
func GetUserAndServer(w http.ResponseWriter, r *http.Request) (user models.QPUser, server *models.QPWhatsappServer, err error) {
	user, err = models.GetUser(r)
	if err != nil {
		RedirectToLogin(w, r)
		return
	}

	r.ParseForm()
	server, err = GetServerFromAuthenticatedRequest(user, r)
	if err != nil {
		RespondServerError(server, w, err)
		return
	}

	return
}

// Returns bot from http form request using E164 id
func GetBotFromRequest(r *http.Request) (models.QPBot, error) {
	var bot models.QPBot
	user, err := models.GetUser(r)
	if err != nil {
		return bot, err
	}

	botID := chi.URLParam(r, "id")
	return models.WhatsappService.DB.Bot.FindForUser(user.ID, botID)
}

// Returns bot from http form request using E164 id
func GetServerFromRequest(r *http.Request) (*models.QPWhatsappServer, error) {
	wid := chi.URLParam(r, "id")
	return models.GetServerFromID(wid)
}

// Search for a server ID from an authenticated request
func GetServerFromAuthenticatedRequest(user models.QPUser, r *http.Request) (server *models.QPWhatsappServer, err error) {
	serverid := r.Form.Get("botID")
	server, ok := models.GetServersForUser(user)[serverid]
	if !ok {
		err = fmt.Errorf("server not found")
	}
	return
}

func GetDownloadPrefix(token string) (path string) {
	path = "/download?token={token}&cache=false&messageid={messageid}"
	path = strings.Replace(path, "{token}", token, -1)
	path = strings.Replace(path, "{messageid}", "", -1)
	return
}
