package models

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const (
	ENVIRONMENT         = "APP_ENV"
	MIGRATIONS          = "MIGRATIONS"
	TITLE               = "APP_TITLE"
	DEBUG_REQUESTS      = "DEBUGREQUESTS"
	DEBUG_JSON_MESSAGES = "DEBUGJSONMESSAGES"
)

type Environment struct{}

var ENV Environment

func (_ *Environment) IsDevelopment() bool {
	environment, _ := GetEnvStr(ENVIRONMENT)
	if strings.ToLower(environment) == "development" {
		return true
	} else {
		return false
	}
}

// MIGRATIONS => Path to database migrations folder
func (_ *Environment) Migrate() bool {
	migrations, _ := GetEnvStr(MIGRATIONS)
	boolMigrations, err := strconv.ParseBool(migrations)
	if err == nil {
		return boolMigrations
	} else {
		return true
	}
}

// MIGRATIONS => Path to database migrations folder
func (_ *Environment) MigrationPath() string {
	migrations, _ := GetEnvStr(MIGRATIONS)
	_, err := strconv.ParseBool(migrations)
	if err != nil {
		return migrations
	} else {
		return "" // indicates that should use default path
	}
}

func (_ *Environment) AppTitle() string {
	title, _ := GetEnvStr(TITLE)
	return title
}

func (_ *Environment) DEBUGRequests() bool {

	if ENV.IsDevelopment() {
		environment, err := GetEnvBool(DEBUG_REQUESTS, true)
		if err == nil {
			return environment
		}
	}

	return false
}

func (_ *Environment) DEBUGJsonMessages() bool {

	if ENV.IsDevelopment() {
		environment, err := GetEnvBool(DEBUG_JSON_MESSAGES, true)
		if err == nil {
			return environment
		}
	}

	return false
}

var ErrEnvVarEmpty = errors.New("getenv: environment variable empty")

func GetEnvBool(key string, value bool) (bool, error) {
	result := value
	s, err := GetEnvStr(key)
	if err == nil {
		trying, err := strconv.ParseBool(s)
		if err == nil {
			result = trying
		}
	}
	return result, err
}

func GetEnvStr(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return v, ErrEnvVarEmpty
	}
	return v, nil
}
