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

	GetWhatsAppQRChannel(chan<- string) error

	// Get group invite link
	GetInvite(groupId string) (string, error)

	// Get info to download profile picture
	GetProfilePicture(wid string, knowingId string) (*WhatsappProfilePicture, error)

	UpdateHandler(IWhatsappHandlers)

	// Download message attachment if exists
	DownloadData(IWhatsappMessage) ([]byte, error)

	// Download message attachment if exists and informations
	Download(IWhatsappMessage) (*WhatsappAttachment, error)

	// Default send message method
	Send(*WhatsappMessage) (IWhatsappSendResponse, error)

	// Define the log level for this connection
	UpdateLog(*log.Entry)

	/*
		<summary>
			Disconnect if connected
			Cleanup Handlers
			Dispose resources
			Does not erase permanent data !
		</summary>
	*/
	Dispose()

	/*
		<summary>
			Erase permanent data + Dispose !
		</summary>
	*/
	Delete() error

	IsInterfaceNil() bool

	// Is connected and logged, valid verification
	IsValid() bool
}
