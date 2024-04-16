package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/carsongro/chirpy/internal/database"
	"github.com/golang-jwt/jwt/v5"
)

func (cfg *apiConfig) GetChirpHandler(w http.ResponseWriter, r *http.Request) {
	db := cfg.db

	chirps, err := db.GetChirps()
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	id, err := strconv.Atoi(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 404, "chirp not found")
		return
	}

	var chirp database.Chirp
	for _, dbChirp := range chirps {
		if dbChirp.Id == id {
			chirp = dbChirp
		}
	}

	respondWithJSON(w, 200, chirp)
}

func (cfg *apiConfig) GetChirpsHandler(w http.ResponseWriter, r *http.Request) {
	db := cfg.db

	chirps, err := db.GetChirps()
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	respondWithJSON(w, 200, chirps)
}

func (cfg *apiConfig) PostChirpHandler(w http.ResponseWriter, r *http.Request) {
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
	if err != nil || !token.Valid || claims.Issuer != "chirpy_access" {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	authorId, err := strconv.Atoi(claims.Subject)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	badWords := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}

	words := strings.Split(params.Body, " ")

	for i, word := range words {
		if badWords[strings.ToLower(word)] {
			words[i] = "****"
		}
	}

	cleanedBody := strings.Join(words, " ")

	newChirp, err := db.CreateChirp(cleanedBody, authorId)
	if err != nil {
		respondWithError(w, 500, "Something went wrong: ")
		return
	}

	respondWithJSON(w, 201, newChirp)
}

func (cfg *apiConfig) DeleteChirpHandler(w http.ResponseWriter, r *http.Request) {
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
	if err != nil || !token.Valid || claims.Issuer != "chirpy_access" {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	authorId, err := strconv.Atoi(claims.Subject)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	chirps, err := db.GetChirps()
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	id, err := strconv.Atoi(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 404, "chirp not found")
		return
	}

	var chirp database.Chirp
	for _, dbChirp := range chirps {
		if dbChirp.Id == id {
			chirp = dbChirp
		}
	}

	if chirp.AuthorId != authorId {
		respondWithError(w, 403, "Unauthorized")
		return
	}

	err = db.DeleteChirp(id)
	if err != nil {
		respondWithJSON(w, 500, "Something went wrong")
	}

	respondWithJSON(w, 200, "")
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type error struct {
		Error string `json:"error"`
	}

	respondWithJSON(w, code, error{
		Error: msg,
	})
}
