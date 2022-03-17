package whatsapp

import (
	log "github.com/sirupsen/logrus"
)

type IWhatsappConnection interface {
	// Returns Connection Version (beta|multi|single)
	GetVersion() string

	GetStatus() WhatsappConnectionState

	// Retorna o ID do controlador whatsapp
	GetWid() (string, error)
	GetTitle(Wid string) string

	Connect() error
	Disconnect() error
	Delete() error
	GetWhatsAppQRChannel(chan<- string) error

	UpdateHandler(IWhatsappHandlers)
	EnsureHandlers() error

	// Download message attachment if exists
	DownloadData(IWhatsappMessage) ([]byte, error)

	// Download message attachment if exists and informations
	Download(IWhatsappMessage) (WhatsappAttachment, error)

	// Default send message method
	Send(WhatsappMessage) (IWhatsappSendResponse, error)

	// Define the log level for this connection
	UpdateLog(*log.Entry)

	// Release all resources
	Dispose()
}
