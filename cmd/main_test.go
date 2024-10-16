package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTask(t *testing.T) {
	db, err := openDB("../test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := &application{
		logger:    log.New(io.Discard, "", 0),
		taskQueue: make(chan int, 100),
		db:        db,
	}

	body := []byte(`{"input": 5}`)
	req, err := http.NewRequest(http.MethodPost, "/tasks", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	app.createTask(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var task Task
	err = json.Unmarshal(rr.Body.Bytes(), &task)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if task.Input != 5 {
		t.Errorf("Expected task input to be 5, got %d", task.Input)
	}

	if task.IsDone {
		t.Errorf("Expected task to be not done")
	}

	// Check if the task was added to the queue
	select {
	case id := <-app.taskQueue:
		if id != task.ID {
			t.Errorf("Expected task ID %d in queue, got %d", task.ID, id)
		}
	default:
		t.Error("Expected task to be added to queue")
	}
}

func TestGetTask(t *testing.T) {
	db, err := openDB("../test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := &application{
		logger: log.New(io.Discard, "", 0),
		db:     db,
	}

	m := NewTask(db)
	m.Input = 5
	m.Create()

	req, err := http.NewRequest(http.MethodGet, "/tasks/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetPathValue("id", "1")

	rr := httptest.NewRecorder()
	app.getTask(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var task Task
	err = json.Unmarshal(rr.Body.Bytes(), &task)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if task.ID != 1 {
		t.Errorf("Expected task ID to be 1, got %d", task.ID)
	}

	if task.Input != 5 {
		t.Errorf("Expected task input to be 5, got %d", task.Input)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	db, err := openDB("../test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app := &application{
		logger: log.New(io.Discard, "", 0),
		db:     db,
	}

	req, err := http.NewRequest(http.MethodGet, "/tasks/999", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.SetPathValue("id", "999")

	rr := httptest.NewRecorder()
	app.getTask(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

}
