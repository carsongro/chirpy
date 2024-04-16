package database

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
)

func (db *DB) GetChirpsHandler(w http.ResponseWriter, r *http.Request) {
	chirps, err := db.GetChirps()
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	respondWithJSON(w, 200, chirps)
}

func (db *DB) PostChirpHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
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

	newChirp, err := db.CreateChirp(cleanedBody)
	if err != nil {
		respondWithError(w, 500, "Something went wrong: ")
		return
	}

	respondWithJSON(w, 201, newChirp)
}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {
	db := DB{
		path: path,
		mux:  &sync.RWMutex{},
	}
	err := db.ensureDB()
	if err != nil {
		return nil, err
	}

	return &db, nil
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	newId := len(dbStructure.Chirps) + 1

	newChirp := Chirp{
		Id:   newId,
		Body: body,
	}

	dbStructure.Chirps[newId] = newChirp

	err = db.writeDB(dbStructure)
	if err != nil {
		return Chirp{}, err
	}

	return newChirp, nil
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()

	dbStructure, err := db.loadDB()
	if err != nil {
		return []Chirp{}, err
	}

	chirps := make([]Chirp, 0, len(dbStructure.Chirps))
	for _, val := range dbStructure.Chirps {
		chirps = append(chirps, val)
	}

	sort.Slice(chirps, func(i, j int) bool { return chirps[i].Id < chirps[j].Id })
	return chirps, nil
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB() error {
	db.mux.Lock()
	defer db.mux.Unlock()

	err := os.WriteFile(db.path, []byte{}, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

// loadDB reads the database file into memory
func (db *DB) loadDB() (DBStructure, error) {
	file, err := os.ReadFile(db.path)
	if err != nil {
		return DBStructure{}, err
	}

	dbStructure := DBStructure{
		Chirps: make(map[int]Chirp),
	}

	if len(file) == 0 {
		return dbStructure, nil
	}

	err = json.Unmarshal(file, &dbStructure.Chirps)
	if err != nil {
		return DBStructure{}, err
	}

	if dbStructure.Chirps == nil {
		dbStructure.Chirps = make(map[int]Chirp)
	}

	return dbStructure, nil
}

// writeDB writes the database file to disk
func (db *DB) writeDB(dbStructure DBStructure) error {
	db.mux.Lock()
	defer db.mux.Unlock()

	newData, err := json.Marshal(dbStructure.Chirps)
	if err != nil {
		return err
	}

	err = os.WriteFile(db.path, newData, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
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
