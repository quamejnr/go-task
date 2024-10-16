package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"time"
)

type Task struct {
	db     *sql.DB
	ID     int  `json:"id"`
	Input  int  `json:"input"`
	Output int  `json:"output"`
	IsDone bool `json:"is_done"`
}

func NewTask(db *sql.DB) *Task {
	return &Task{db: db}
}

func (t *Task) Create() error {
	query := "INSERT INTO Tasks (input) VALUES (?) RETURNING id"
	if err := t.db.QueryRow(query, t.Input).Scan(&t.ID); err != nil {
		return err
	}
	return nil
}

func (t *Task) Retrieve(id int) error {
	var IsDone int
	err := t.db.QueryRow(
		"SELECT id, input, output, is_done FROM Tasks WHERE id = ?", id).
		Scan(&t.ID, &t.Input, &t.Output, &IsDone)
	t.IsDone = intToBool(IsDone)

	if err != nil {
		return err
	}
	return nil
}

func (t *Task) Update(id int) error {
	_, err := t.db.Exec("UPDATE Tasks SET output = input * input, is_done = 1 WHERE id = ?", id)
	if err != nil {
		return err
	}
	return nil

}

func intToBool(val int) bool {
	if val == 1 {
		return true
	}
	return false
}

type Auth struct {
	db       *sql.DB
	ID       int       `json:"-"`
	Username string    `json:"-"`
	Password []byte    `json:"-"`
	Token    string    `json:"token"`
	Expiry   time.Time `json:"expires_in"`
}

func NewAuth(db *sql.DB) *Auth {
	return &Auth{db: db}
}

func (a *Auth) Retrieve(username string) error {
	err := a.db.QueryRow(
		"SELECT username, password FROM Auth WHERE username = ?", username).
		Scan(&a.Username, &a.Password)
	if err != nil {
		return err
	}
	return nil
}

func (a *Auth) generateToken() error {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return err
	}
	a.Token = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	a.Expiry = time.Now().Add(1 * time.Hour)
	_, err := a.db.Exec("UPDATE Auth SET token = ?, expiry = ? WHERE username = ?", a.Token, a.Expiry, a.Username)
	if err != nil {
		return err
	}
	return nil
}

