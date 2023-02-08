package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	library "github.com/sufficit/sufficit-quepasa/library"
	metrics "github.com/sufficit/sufficit-quepasa/metrics"
	models "github.com/sufficit/sufficit-quepasa/models"
	whatsapp "github.com/sufficit/sufficit-quepasa/whatsapp"
)

const APIVersion3 string = "v3"

var ControllerPrefixV3 string = fmt.Sprintf("/%s/bot/{token}", APIVersion3)

func RegisterAPIV3Controllers(r chi.Router) {
	r.Get(ControllerPrefixV3, InformationControllerV3)

	// SENDING MSG ----------------------------
	// ----------------------------------------

	// used to dispatch alert msgs via url, triggers on monitor systems like zabbix
	r.Get(ControllerPrefixV3+"/send", SendAny)

	r.Post(ControllerPrefixV3+"/send", SendAny)
	r.Post(ControllerPrefixV3+"/send/{chatid}", SendAny)
	r.Post(ControllerPrefixV3+"/sendtext", SendText)
	r.Post(ControllerPrefixV3+"/sendtext/{chatid}", SendText)

	// SENDING MSG ATTACH ---------------------

	// deprecated, discard/remove on next version
	r.Post(ControllerPrefixV3+"/senddocument", SendDocumentAPIHandlerV2)

	r.Post(ControllerPrefixV3+"/sendurl", SendDocumentFromUrl)
	r.Post(ControllerPrefixV3+"/sendbinary/{chatid}/{fileName}/{text}", SendDocumentFromBinary)
	r.Post(ControllerPrefixV3+"/sendbinary/{chatid}/{fileName}", SendDocumentFromBinary)
	r.Post(ControllerPrefixV3+"/sendbinary/{chatid}", SendDocumentFromBinary)
	r.Post(ControllerPrefixV3+"/sendbinary", SendDocumentFromBinary)
	r.Post(ControllerPrefixV3+"/sendencoded", SendDocumentFromEncoded)

	// ----------------------------------------
	// SENDING MSG ----------------------------

	r.Get(ControllerPrefixV3+"/receive", ReceiveAPIHandler)
	r.Post(ControllerPrefixV3+"/attachment", AttachmentAPIHandlerV2)

	r.Get(ControllerPrefixV3+"/download/{messageId}", DownloadControllerV3)
	r.Get(ControllerPrefixV3+"/download", DownloadControllerV3)

	// PICTURE INFO | DATA --------------------
	// ----------------------------------------

	r.Post(ControllerPrefixV3+"/picinfo", PictureController)
	r.Get(ControllerPrefixV3+"/picinfo/{chatid}/{pictureid}", PictureController)
	r.Get(ControllerPrefixV3+"/picinfo/{chatid}", PictureController)
	r.Get(ControllerPrefixV3+"/picinfo", PictureController)

	r.Post(ControllerPrefixV3+"/picdata", PictureController)
	r.Get(ControllerPrefixV3+"/picdata/{chatid}/{pictureid}", PictureController)
	r.Get(ControllerPrefixV3+"/picdata/{chatid}", PictureController)
	r.Get(ControllerPrefixV3+"/picdata", PictureController)

	// ----------------------------------------
	// PICTURE INFO | DATA --------------------

	r.Post(ControllerPrefixV3+"/webhook", WebhookController)
	r.Get(ControllerPrefixV3+"/webhook", WebhookController)
	r.Delete(ControllerPrefixV3+"/webhook", WebhookController)

	// INVITE METHODS ************************
	// ----------------------------------------

	r.Get(ControllerPrefixV3+"/invite/{chatid}", InviteController)

	// ----------------------------------------
	// INVITE METHODS ************************
}

//region CONTROLLER - INFORMATION

// InformationController renders route GET "/{version}/info"
func InformationControllerV3(w http.ResponseWriter, r *http.Request) {
	// setting default reponse type as json
	w.Header().Set("Content-Type", "application/json")

	response := &models.QpInfoResponse{}

	server, err := GetServer(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	response.ParseSuccess(*server)
	RespondSuccess(w, response)
}

//endregion

//endregion
//region CONTROLLER - DOWNLOAD MESSAGE ATTACHMENT

func GetDownloadPrefixV3(token string) (path string) {
	path = ControllerPrefixV3 + "/download/{messageId}"
	path = strings.Replace(path, "{token}", token, -1)
	path = strings.Replace(path, "{messageId}", "", -1)
	return
}

/*
<summary>
	Renders route GET "/{{version}}/bot/{{token}}/download/{messageId}"

	Any of then, at this order of priority
	Path parameters: {messageId}
	Url parameters: ?messageId={messageId} || ?id={messageId}
	Header parameters: X-QUEPASA-MESSAGEID = {messageId}
</summary>
*/
func DownloadControllerV3(w http.ResponseWriter, r *http.Request) {

	response := &models.QpResponse{}

	server, err := GetServer(r)
	if err != nil {
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// Evitando tentativa de download de anexos sem o bot estar devidamente sincronizado
	status := server.GetStatus()
	if status != whatsapp.Ready {
		err := &ApiServerNotReadyException{Wid: server.GetWid(), Status: status}
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	// Default parameters
	messageId := chi.URLParam(r, "messageId")
	if strings.Contains(messageId, "message") || (len(messageId) == 0 && r.URL.Query().Has("id")) {
		messageId = r.URL.Query().Get("id")
	} else if len(messageId) == 0 && r.URL.Query().Has("messageId") {
		messageId = r.URL.Query().Get("messageId")
	} else if len(messageId) == 0 {
		messageId = r.Header.Get("X-QUEPASA-MESSAGEID")
	}

	if len(messageId) == 0 {
		metrics.MessageSendErrors.Inc()
		err := fmt.Errorf("empty message id")
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	att, err := server.Download(messageId)
	if err != nil {
		response.ParseError(err)
		RespondInterface(w, response)
		return
	}

	var fileName string

	// If filename not setted
	if len(att.FileName) == 0 {
		exten, _ := library.GetExtensionFromMimeType(att.Mimetype)
		if len(exten) > 0 {

			// Generate from mime type and message id
			fileName = fmt.Sprint("; filename=", messageId, ".", exten)
		}
	} else {
		fileName = fmt.Sprint("; filename=", att.FileName)
	}

	// setting header filename
	w.Header().Set("Content-Disposition", fmt.Sprint("attachment", fileName))

	w.WriteHeader(http.StatusOK)
	w.Write(*att.GetContent())
}

//endregion
