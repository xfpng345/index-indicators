package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"

	"index-indicator-apis/server/app/entity"
	"index-indicator-apis/server/app/models"

	"github.com/markcheno/go-quote"
	"golang.org/x/crypto/bcrypt"
)

//App struct
type App struct {
	DB entity.DB
}

//NewApp return *APP
func NewApp(models *models.Models) *App {
	return &App{
		DB: models,
	}
}

// JSONResponse is a response mssage
type JSONResponse struct {
	Response string `json:"response"`
	Code     int    `json:"code"`
}

func (a *App) resposeStatusCode(w http.ResponseWriter, ResMessage string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	jsonError, err := json.Marshal(JSONResponse{Response: ResMessage, Code: code})
	if err != nil {
		log.Fatal(err)
	}
	w.Write(jsonError)
}

func (a *App) tokenVerifyMiddleWare(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessDetails, err := models.ExtractTokenMetadata(r)
		if err != nil {
			a.resposeStatusCode(w, "unauthorized", http.StatusNotFound)
			return
		}

		// Redisからtokenを検索して見つからない場合はunauthorizedを返す。
		_, authErr := models.FetchAuth(accessDetails)
		if authErr != nil {
			a.resposeStatusCode(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		fn(w, r)
	})
}

// ---------usersHandlers---------
func (a *App) signupHandler(w http.ResponseWriter, r *http.Request) {
	var u entity.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		a.resposeStatusCode(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := u.UserName
	email := u.Email
	pass := u.Password

	if name == "" {
		a.resposeStatusCode(w, "UserName is required", http.StatusBadRequest)
		return
	}
	if email == "" {
		a.resposeStatusCode(w, "Email is required", http.StatusBadRequest)
		return
	}
	if pass == "" {
		a.resposeStatusCode(w, "Password is required", http.StatusBadRequest)
		return
	}

	if err := a.DB.CreateUser(name, email, pass); err != nil {
		a.resposeStatusCode(w, "username or email are duplicated", http.StatusConflict)
		return
	}

	a.resposeStatusCode(w, "success", http.StatusCreated)
	return
}

func (a *App) userDeleteHandler(w http.ResponseWriter, r *http.Request) {
	var u entity.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		a.resposeStatusCode(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(path.Base(r.URL.Path))
	if err != nil {
		a.resposeStatusCode(w, "cloud not find user", http.StatusNotFound)
		return
	}

	err = a.DB.DeleteUser(id, u.Password)
	if err != nil {
		a.resposeStatusCode(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.resposeStatusCode(w, "success", http.StatusOK)
	return
}

func (a *App) userUpdateHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(path.Base(r.URL.Path))
	if err != nil {
		a.resposeStatusCode(w, err.Error(), http.StatusBadRequest)
		return
	}

	foundUser, err := a.DB.FindUserByID(id)
	if err != nil {
		a.resposeStatusCode(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type body struct {
		User struct {
			Password string `json:"password,omitempty"`
		} `json:"user,omitempty"`
		NewUser struct {
			UserName string `json:"user_name,omitempty"`
			Email    string `json:"email,omitempty"`
			Password string `json:"password,omitempty"`
		} `json:"new_user,omitempty"`
	}

	var updateUser body
	if err := json.NewDecoder(r.Body).Decode(&updateUser); err != nil {
		a.resposeStatusCode(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(updateUser.User.Password)); err != nil {
		a.resposeStatusCode(w, err.Error(), http.StatusNotAcceptable)
		return
	}
	if updateUser.NewUser.UserName != "" {
		foundUser.UserName = updateUser.NewUser.UserName
	}
	if updateUser.NewUser.Email != "" {
		foundUser.Email = updateUser.NewUser.Email
	}
	if updateUser.NewUser.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(updateUser.NewUser.Password), 10)
		if err != nil {
			a.resposeStatusCode(w, err.Error(), http.StatusBadRequest)
			return
		}
		foundUser.Password = string(hash)
	}

	if err := a.DB.UpdateUser(foundUser); err != nil {
		a.resposeStatusCode(w, err.Error(), http.StatusBadRequest)
		return
	}

	a.resposeStatusCode(w, "success", http.StatusOK)
	return
}

// ---------authHandlers---------
func (a *App) loginHandler(w http.ResponseWriter, r *http.Request) {
	var user entity.User
	json.NewDecoder(r.Body).Decode(&user)

	searchedUser, err := models.FindUser(user)
	if err != nil {
		a.resposeStatusCode(w, err.Error(), http.StatusNotFound)
		return
	}

	fmt.Println("compare the password")
	if err := bcrypt.CompareHashAndPassword([]byte(searchedUser.Password), []byte(user.Password)); err != nil {
		a.resposeStatusCode(w, err.Error(), http.StatusUnauthorized)
		return
	}
	fmt.Println("password is be valid")

	token, err := models.CreateToken(searchedUser.ID)
	if err != nil {
		a.resposeStatusCode(w, err.Error(), http.StatusUnauthorized)
		return
	}

	saveErr := models.CreateAuth(searchedUser.ID, token)
	if saveErr != nil {
		a.resposeStatusCode(w, err.Error(), http.StatusUnauthorized)
		return
	}

	tokens := map[string]string{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(tokens)
}

func (a *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	accessDetails, err := models.ExtractTokenMetadata(r)
	if err != nil {
		a.resposeStatusCode(w, "not found", http.StatusNotFound)
		return
	}

	deleted, delErr := models.DeleteAuth(accessDetails.AccessUUID)
	if delErr != nil || deleted == 0 {
		a.resposeStatusCode(w, "not found", http.StatusNotFound)
		return
	}

	a.resposeStatusCode(w, "success", http.StatusOK)
}

func (a *App) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {

	mapToken := map[string]string{}
	json.NewDecoder(r.Body).Decode(&mapToken)
	refreshToken := mapToken["refresh_token"]

	tokens, errMsg := models.RefreshAuth(r, refreshToken)
	if errMsg != "" {
		a.resposeStatusCode(w, errMsg, http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(tokens)
}

// ---------fgisHandlers---------
func (a *App) fgiHandler(w http.ResponseWriter, r *http.Request) {
	strLimit := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(strLimit)
	if strLimit == "" || err != nil || limit < 0 || limit > 100 {
		limit = 100
	}
	fgi := models.GetFgis(limit)
	js, err := json.Marshal(fgi)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(js)
}

// ---------quote---------
func (a *App) tickerHandler(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	startDay := r.URL.Query().Get("start")
	endDay := r.URL.Query().Get("end")

	if symbol == "" || startDay == "" || endDay == "" {
		a.resposeStatusCode(w, "some query are empty", http.StatusUnauthorized)
		return
	}

	ticker, _ := quote.NewQuoteFromYahoo(
		symbol, startDay, endDay, quote.Daily, true)

	json.NewEncoder(w).Encode(ticker)

}
