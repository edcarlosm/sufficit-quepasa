package main

import (
	"github.com/joho/godotenv"
	controllers "github.com/sufficit/sufficit-quepasa/controllers"
	library "github.com/sufficit/sufficit-quepasa/library"
	models "github.com/sufficit/sufficit-quepasa/models"
	whatsmeow "github.com/sufficit/sufficit-quepasa/whatsmeow"

	log "github.com/sirupsen/logrus"
)

// @title chi-swagger example APIs
// @version 1.0
// @description chi-swagger example APIs
// @BasePath /
func main() {

	// Carregando variaveis de ambiente apartir de arquivo .env
	godotenv.Load()

	if models.ENV.DEBUGJsonMessages() {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	library.EnsureMimesMapping()

	// Verifica se é necessario realizar alguma migração de base de dados
	err := models.MigrateToLatest()
	if err != nil {
		log.Fatalf("Database migration error: %s", err.Error())
	}

	whatsmeow.WhatsmeowService.Start()

	// Inicializando serviço de controle do whatsapp
	// De forma assíncrona
	err = models.QPWhatsappStart()
	if err != nil {
		panic(err.Error())
	}

	controllers.QPWebServerStart()
}
