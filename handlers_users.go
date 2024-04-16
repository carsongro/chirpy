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
		Id    int    `json:"id"`
		Email string `json:"email"`
	}

	respondWithJSON(w, 201, newUserResponse{
		Id:    newUser.Id,
		Email: newUser.Email,
	})
}

func (cfg *apiConfig) PostLoginHandler(w http.ResponseWriter, r *http.Request) {
	db := cfg.db

	type parameters struct {
		Password         string `json:"password"`
		Email            string `json:"email"`
		ExpiresInSeconds *int   `json:"expires_in_seconds"`
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

	expiresIn := 86400
	if params.ExpiresInSeconds != nil {
		expiresIn = *params.ExpiresInSeconds
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Second * time.Duration(expiresIn))),
		Subject:   strconv.Itoa(user.Id),
	})

	tokenString, err := token.SignedString([]byte(cfg.jwtSecret))
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	type userResponse struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
		Token string `json:"token"`
	}

	respondWithJSON(w, 200, userResponse{
		Id:    user.Id,
		Email: user.Email,
		Token: tokenString,
	})
}

func (cfg *apiConfig) PustUsersHandler(w http.ResponseWriter, r *http.Request) {
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
	if err != nil || !token.Valid {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	id, err := strconv.Atoi(claims.Subject)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	updatedUser, err := db.UpdateUser(id, params.Email, string(hashedPassword))
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	type userResponse struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
	}

	respondWithJSON(w, 200, userResponse{
		Id:    updatedUser.Id,
		Email: updatedUser.Email,
	})
}
