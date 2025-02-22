package controllers

import (
	"errors"
	"html/template"
	"net/http"
	"os"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth"
	log "github.com/sirupsen/logrus"
	. "github.com/sufficit/sufficit-quepasa/models"
)

var FormLoginEndpoint string = "/login"
var FormSetupEndpoint string = "/setup"
var FormLogoutEndpoint string = "/logout"

func RegisterFormControllers(r chi.Router) {
	r.Get("/", IndexHandler)
	r.Get(FormLoginEndpoint, LoginFormHandler)
	r.Post(FormLoginEndpoint, LoginHandler)
	r.Get(FormSetupEndpoint, SetupFormHandler)
	r.Post(FormSetupEndpoint, SetupHandler)
	r.Get(FormLogoutEndpoint, LogoutHandler)
}

// LoginFormHandler renders route GET "/login"
func LoginFormHandler(w http.ResponseWriter, r *http.Request) {
	data := QPFormLoginData{PageTitle: "Login"}

	templates := template.Must(template.ParseFiles("views/layouts/main.tmpl", "views/login.tmpl"))
	templates.ExecuteTemplate(w, "main", data)
}

// LoginHandler renders route POST "/login"
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	email := r.Form.Get("email")
	password := r.Form.Get("password")

	if email == "" || password == "" {
		RespondUnauthorized(w, errors.New("Missing username or password"))
		return
	}

	user, err := WhatsappService.GetUser(email, password)
	if err != nil {
		RespondUnauthorized(w, errors.New("Incorrect username or password"))
		return
	}

	tokenAuth := jwtauth.New("HS256", []byte(os.Getenv("SIGNING_SECRET")), nil)
	claims := jwt.MapClaims{"user_id": user.ID}
	jwtauth.SetIssuedNow(claims)
	jwtauth.SetExpiryIn(claims, 24*time.Hour)
	_, tokenString, err := tokenAuth.Encode(claims)
	if err != nil {
		RespondErrorCode(w, errors.New("Cannot encode token to save"), 500)
		return
	}

	cookie := &http.Cookie{
		Name:     "jwt",
		Value:    tokenString,
		MaxAge:   60 * 60 * 24,
		Path:     "/",
		HttpOnly: true,
	}

	log.Debugf("setting cookie and redirecting to: %v", FormAccountEndpoint)
	http.SetCookie(w, cookie)
	http.Redirect(w, r, FormAccountEndpoint, http.StatusFound)
}
