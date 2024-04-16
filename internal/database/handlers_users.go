package database

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func (db *DB) PostUserHandler(w http.ResponseWriter, r *http.Request) {
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

func (db *DB) PostLoginHandler(w http.ResponseWriter, r *http.Request) {
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

	dbStructure, err := db.loadDB()
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	var user User
	for _, dbUser := range dbStructure.Users {
		if dbUser.Email == params.Email {
			user = dbUser
		}
	}

	if user == (User{}) {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password))
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	type newUserResponse struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
	}

	respondWithJSON(w, 200, newUserResponse{
		Id:    user.Id,
		Email: user.Email,
	})
}
