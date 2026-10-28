package main

import (
	"log"
	"net/http"
	"task-api/internal/handler"
)

func main() {
	mux := http.NewServeMux()

	taskHandler := handler.NewTaskHandler()

	mux.HandleFunc("GET /health", handler.HealthHandler)
	mux.HandleFunc("POST /tasks", taskHandler.CreateTask)
	mux.HandleFunc("GET /tasks", taskHandler.GetTask)
	mux.HandleFunc("GET /tasks/{id}", taskHandler.GetTaskByID)
	mux.HandleFunc("PUT /tasks/{id}", taskHandler.UpdateTaskByID)
	mux.HandleFunc("DELETE /tasks/{id}", taskHandler.DeleteTaskByID)

	log.Println("Starting server on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
