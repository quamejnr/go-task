package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

func (app *application) ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(map[string]string{"status": "available"})
}

func (app *application) createTask(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		app.logger.Printf("error reading body %s", err.Error())
		http.Error(w, "body contains malformed JSON", http.StatusBadRequest)
		return
	}
	var i struct {
		Input int
	}
	err = json.Unmarshal(body, &i)
	if err != nil {
		app.logger.Printf("error reading body %s", err.Error())
		http.Error(w, "`input` field is required", http.StatusBadRequest)
		return
	}

	task := NewTask(app.db)
	task.Input = i.Input
	task.Create()

	// add to task queue
	app.taskQueue <- task.ID

	res, err := json.Marshal(task)
	if err != nil {
		app.logger.Printf("error marshalling response %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(res))
}

func (app *application) getTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		app.logger.Printf("error converting path value %s", err.Error())
		http.Error(w, "something went wrong. Kindly contact customer support", http.StatusInternalServerError)
		return
	}

	t := NewTask(app.db)
	if err := t.Retrieve(id); err != nil {
		if err == sql.ErrNoRows {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"detail": "Not Found"}`, http.StatusNotFound)
			return
		} else {
			app.logger.Printf("error retrieving rows in db %s", err.Error())
			http.Error(w, "something went wrong. Kindly contact customer support", http.StatusInternalServerError)
			return
		}
	}

	res, err := json.Marshal(t)
	if err != nil {
		app.logger.Printf("error marshalling response %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(res))

}

func (app *application) getToken(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		app.logger.Printf("error reading body %s", err.Error())
		http.Error(w, "body contains malformed JSON", http.StatusBadRequest)
		return
	}

	var i struct {
		Username, Password string
	}
	err = json.Unmarshal(body, &i)
	if err != nil {
		app.logger.Printf("error reading body %s", err.Error())
		http.Error(w, "`username` and `password` fields are required", http.StatusBadRequest)
		return
	}

	a := NewAuth(app.db)
	if err := a.Retrieve(i.Username); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "username is incorrect", http.StatusUnauthorized)
			return
		} else {
			app.logger.Printf("error retrieving rows in db %s", err.Error())
			http.Error(w, "something went wrong. Kindly contact customer support", http.StatusInternalServerError)
			return
		}
	}
	err = bcrypt.CompareHashAndPassword(a.Password, []byte(i.Password))
	if err != nil {
		app.logger.Printf("password authentication failed %s", err.Error())
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	if err := a.generateToken(); err != nil {
		app.logger.Printf("error generating token %s", err)
		http.Error(w, "something went wrong. Kindly contact customer support", http.StatusInternalServerError)
		return
	}

	res, err := json.Marshal(a)
	if err != nil {
		app.logger.Printf("error marshalling response %s", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(res))
}
