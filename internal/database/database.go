package database

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"
	"time"
)

func (db *DB) GetRevokedTokens() (map[string]time.Time, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return map[string]time.Time{}, err
	}

	return dbStructure.RevokedTokens, nil
}

func (db *DB) UpdateRevokedTokens(token string) error {
	dbStructure, err := db.loadDB()
	if err != nil {
		return err
	}

	dbStructure.RevokedTokens[token] = time.Now().UTC()

	err = db.writeDB(dbStructure)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) UpdateUser(Id int, email, password string) (User, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	newUser := User{
		Id:       Id,
		Email:    email,
		Password: password,
	}

	_, ok := dbStructure.Users[Id]
	if !ok {
		return User{}, errors.New("failed to update user")
	}

	dbStructure.Users[Id] = newUser

	err = db.writeDB(dbStructure)
	if err != nil {
		return User{}, err
	}

	return newUser, nil
}

// CreateUser creates a new user and saves it to disk
func (db *DB) CreateUser(email string, password string) (User, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	for _, user := range dbStructure.Users {
		if email == user.Email {
			return User{}, errors.New("a user with this email already exists")
		}
	}

	newId := len(dbStructure.Users) + 1

	newUser := User{
		Id:       newId,
		Email:    email,
		Password: password,
	}

	dbStructure.Users[newId] = newUser

	err = db.writeDB(dbStructure)
	if err != nil {
		return User{}, err
	}

	return newUser, nil
}

// CreateUser creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string, authorId int) (Chirp, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	newId := len(dbStructure.Chirps) + 1

	newChirp := Chirp{
		Id:       newId,
		Body:     body,
		AuthorId: authorId,
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

func (db *DB) DeleteChirp(id int) error {
	dbStructure, err := db.loadDB()
	if err != nil {
		return err
	}

	if _, ok := dbStructure.Chirps[id]; !ok {
		return errors.New("chirp not found")
	}

	delete(dbStructure.Chirps, id)

	err = db.writeDB(dbStructure)
	if err != nil {
		return err
	}

	return nil
}

// GetUsers returns all users in the database
func (db *DB) GetUsers() ([]User, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()

	dbStructure, err := db.loadDB()
	if err != nil {
		return []User{}, err
	}

	users := make([]User, 0, len(dbStructure.Users))
	for _, val := range dbStructure.Users {
		users = append(users, val)
	}

	sort.Slice(users, func(i, j int) bool { return users[i].Id < users[j].Id })
	return users, nil
}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string, makeNew bool) (*DB, error) {
	db := DB{
		path: path,
		mux:  &sync.RWMutex{},
	}
	err := db.ensureDB(makeNew)
	if err != nil {
		return nil, err
	}

	return &db, nil
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB(makeNew bool) error {
	db.mux.Lock()
	defer db.mux.Unlock()
	if makeNew {
		err := os.WriteFile(db.path, []byte{}, os.ModePerm)
		if err != nil {
			return err
		}
	} else {
		if _, err := os.Stat(db.path); err != nil {
			err := os.WriteFile(db.path, []byte{}, os.ModePerm)
			if err != nil {
				return err
			}
		}
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
		Chirps:        make(map[int]Chirp),
		Users:         make(map[int]User),
		RevokedTokens: make(map[string]time.Time),
	}

	if len(file) == 0 {
		return dbStructure, nil
	}

	err = json.Unmarshal(file, &dbStructure)
	if err != nil {
		return DBStructure{}, err
	}

	if dbStructure.Chirps == nil {
		dbStructure.Chirps = make(map[int]Chirp)
		dbStructure.Users = make(map[int]User)
		dbStructure.RevokedTokens = make(map[string]time.Time)
	}

	return dbStructure, nil
}

// writeDB writes the database file to disk
func (db *DB) writeDB(dbStructure DBStructure) error {
	db.mux.Lock()
	defer db.mux.Unlock()

	newData, err := json.Marshal(dbStructure)
	if err != nil {
		return err
	}

	err = os.WriteFile(db.path, newData, os.ModeAppend)
	if err != nil {
		return err
	}

	return nil
}
