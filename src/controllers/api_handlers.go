package controllers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	metrics "github.com/sufficit/sufficit-quepasa/metrics"
	models "github.com/sufficit/sufficit-quepasa/models"
)

const CurrentAPIVersion string = "v4"

func RegisterAPIControllers(r chi.Router) {
	aliases := []string{"/current", "", "/" + CurrentAPIVersion}
	for _, endpoint := range aliases {

		// CONTROL METHODS ************************
		// ----------------------------------------
		r.Get(endpoint+"/info", InformationControllerV3)
		r.Get(endpoint+"/scan", ScannerController)

		// ----------------------------------------
		// CONTROL METHODS ************************

		// SENDING MSG ----------------------------
		// ----------------------------------------

		// used to dispatch alert msgs via url, triggers on monitor systems like zabbix
		r.Get(endpoint+"/send", SendAny)

		r.Post(endpoint+"/send", SendAny)
		r.Post(endpoint+"/send/{chatid}", SendAny)
		r.Post(endpoint+"/sendtext", SendText)
		r.Post(endpoint+"/sendtext/{chatid}", SendText)

		// SENDING MSG ATTACH ---------------------

		// deprecated, discard/remove on next version
		r.Post(endpoint+"/senddocument", SendDocumentAPIHandlerV2)

		r.Post(endpoint+"/sendurl", SendDocumentFromUrl)
		r.Post(endpoint+"/sendbinary/{chatid}/{filename}/{text}", SendDocumentFromBinary)
		r.Post(endpoint+"/sendbinary/{chatid}/{filename}", SendDocumentFromBinary)
		r.Post(endpoint+"/sendbinary/{chatid}", SendDocumentFromBinary)
		r.Post(endpoint+"/sendbinary", SendDocumentFromBinary)
		r.Post(endpoint+"/sendencoded", SendDocumentFromEncoded)

		// ----------------------------------------
		// SENDING MSG ----------------------------

		r.Get(endpoint+"/receive", ReceiveAPIHandler)
		r.Post(endpoint+"/attachment", AttachmentAPIHandlerV2)

		r.Get(endpoint+"/download/{messageid}", DownloadController)
		r.Get(endpoint+"/download", DownloadController)

		// PICTURE INFO | DATA --------------------
		// ----------------------------------------

		r.Post(endpoint+"/picinfo", PictureController)
		r.Get(endpoint+"/picinfo/{chatid}/{pictureid}", PictureController)
		r.Get(endpoint+"/picinfo/{chatid}", PictureController)
		r.Get(endpoint+"/picinfo", PictureController)

		r.Post(endpoint+"/picdata", PictureController)
		r.Get(endpoint+"/picdata/{chatid}/{pictureid}", PictureController)
		r.Get(endpoint+"/picdata/{chatid}", PictureController)
		r.Get(endpoint+"/picdata", PictureController)

		// ----------------------------------------
		// PICTURE INFO | DATA --------------------

		r.Post(endpoint+"/webhook", WebhookController)
		r.Get(endpoint+"/webhook", WebhookController)
		r.Delete(endpoint+"/webhook", WebhookController)

		// INVITE METHODS ************************
		// ----------------------------------------

		r.Get(endpoint+"/invite", InviteController)
		r.Get(endpoint+"/invite/{chatid}", InviteController)

		// ----------------------------------------
		// INVITE METHODS ************************
	}
}

func ScannerController(w http.ResponseWriter, r *http.Request) {
	// setting default reponse type as json
	w.Header().Set("Content-Type", "application/json")

	response := &models.QpInfoResponse{}

	server, err := GetServer(r)
	if err != nil {
		metrics.MessageSendErrors.Inc()
		response.ParseError(err)
		RespondServerError(server, w, response)
		return
	}

	response.ParseSuccess(*server)
	RespondSuccess(w, response)
}
