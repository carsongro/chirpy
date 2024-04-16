package database

import (
	"encoding/json"
	"net/http"
)

func (db *DB) PostUserHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	newUser, err := db.CreateUser(params.Email)
	if err != nil {
		respondWithError(w, 500, "Something went wrong: ")
		return
	}

	respondWithJSON(w, 201, newUser)
}
