package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/carsongro/chirpy/internal/database"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func (cfg *apiConfig) PostUserUpgrade(w http.ResponseWriter, r *http.Request) {
	db := cfg.db

	polkaKey := r.Header.Get("Authorization")
	polkaKey, found := strings.CutPrefix(polkaKey, "Apikey ")
	if !found || polkaKey != cfg.polkaKey {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserId int `json:"user_id"`
		} `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil || params.Event != "user.upgraded" {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	user, err := db.GetUser(params.Data.UserId)
	if err != nil {
		respondWithError(w, 404, "User not found")
		return
	}

	_, err = db.UpdateUser(user.Id, user.Email, user.Password, true)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	respondWithJSON(w, 200, "")
}

func (cfg *apiConfig) PostUserHandler(w http.ResponseWriter, r *http.Request) {
	db := cfg.db

	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	newUser, err := db.CreateUser(params.Email, string(hashedPassword))
	if err != nil {
		respondWithError(w, 500, "Something went wrong: ")
		return
	}

	type newUserResponse struct {
		Id          int    `json:"id"`
		Email       string `json:"email"`
		IsChirpyRed bool   `json:"is_chirpy_red"`
	}

	respondWithJSON(w, 201, newUserResponse{
		Id:          newUser.Id,
		Email:       newUser.Email,
		IsChirpyRed: false,
	})
}

func (cfg *apiConfig) PostLoginHandler(w http.ResponseWriter, r *http.Request) {
	db := cfg.db

	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	users, err := db.GetUsers()
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	var user database.User
	for _, dbUser := range users {
		if dbUser.Email == params.Email {
			user = dbUser
		}
	}

	if user == (database.User{}) {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password))
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy_access",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Second * time.Duration(3600))),
		Subject:   strconv.Itoa(user.Id),
	})

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy_refresh",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Second * time.Duration(86400*60))),
		Subject:   strconv.Itoa(user.Id),
	})

	tokenString, err := token.SignedString([]byte(cfg.jwtSecret))
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	refreshTokenString, err := refreshToken.SignedString([]byte(cfg.jwtSecret))
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	type userResponse struct {
		Id           int    `json:"id"`
		Email        string `json:"email"`
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
		IsChirpyRed  bool   `json:"is_chirpy_red"`
	}

	respondWithJSON(w, 200, userResponse{
		Id:           user.Id,
		Email:        user.Email,
		Token:        tokenString,
		RefreshToken: refreshTokenString,
		IsChirpyRed:  user.IsChirpyRed,
	})
}

func (cfg *apiConfig) PutUsersHandler(w http.ResponseWriter, r *http.Request) {
	db := cfg.db

	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	jwtToken := r.Header.Get("Authorization")
	jwtToken, found := strings.CutPrefix(jwtToken, "Bearer ")
	if !found {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	type userClaims struct {
		jwt.RegisteredClaims
	}
	var claims userClaims
	token, err := jwt.ParseWithClaims(jwtToken, &claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(cfg.jwtSecret), nil
	})
	if err != nil || !token.Valid || claims.Issuer == "chirpy_refresh" {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	id, err := strconv.Atoi(claims.Subject)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	oldUser, err := db.GetUser(id)
	if err != nil {
		respondWithError(w, 404, "User not found")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	updatedUser, err := db.UpdateUser(id, params.Email, string(hashedPassword), oldUser.IsChirpyRed)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	type userResponse struct {
		Id          int    `json:"id"`
		Email       string `json:"email"`
		IsChirpyRed bool   `json:"is_chirpy_red"`
	}

	respondWithJSON(w, 200, userResponse{
		Id:          updatedUser.Id,
		Email:       updatedUser.Email,
		IsChirpyRed: updatedUser.IsChirpyRed,
	})
}

func (cfg *apiConfig) PostRefreshHandler(w http.ResponseWriter, r *http.Request) {
	db := cfg.db

	jwtToken := r.Header.Get("Authorization")
	jwtToken, found := strings.CutPrefix(jwtToken, "Bearer ")
	if !found {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	type userClaims struct {
		jwt.RegisteredClaims
	}
	var claims userClaims
	token, err := jwt.ParseWithClaims(jwtToken, &claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(cfg.jwtSecret), nil
	})
	if err != nil || !token.Valid || claims.Issuer != "chirpy_refresh" {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	revokedTokens, err := db.GetRevokedTokens()
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	if _, ok := revokedTokens[token.Raw]; ok {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy_access",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Second * time.Duration(3600))),
		Subject:   claims.Subject,
	})

	tokenString, err := newToken.SignedString([]byte(cfg.jwtSecret))
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	type tokenResponse struct {
		Token string `json:"token"`
	}

	respondWithJSON(w, 200, tokenResponse{
		Token: tokenString,
	})
}

func (cfg *apiConfig) PostRevokeHandler(w http.ResponseWriter, r *http.Request) {
	db := cfg.db

	jwtToken := r.Header.Get("Authorization")
	jwtToken, found := strings.CutPrefix(jwtToken, "Bearer ")
	if !found {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	type userClaims struct {
		jwt.RegisteredClaims
	}
	var claims userClaims
	token, err := jwt.ParseWithClaims(jwtToken, &claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(cfg.jwtSecret), nil
	})
	if err != nil || !token.Valid || claims.Issuer != "chirpy_refresh" {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	err = db.UpdateRevokedTokens(token.Raw)
	if err != nil {
		respondWithError(w, 500, "something went wrong")
		return
	}

	respondWithJSON(w, 200, "")
}
