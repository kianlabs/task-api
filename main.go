package main

import (
	"log"
	"net/http"
	"os"
	"task-api/internal/auth"
	"task-api/internal/database"
	"task-api/internal/handler"
)

func main() {
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	taskHandler := handler.NewTaskHandler(db)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-key-change-in-prod"
	}
	authHandler := handler.NewAuthHandler(db, jwtSecret)
	authMiddleware := auth.Middleware(jwtSecret)

	mux.HandleFunc("GET /health", handler.HealthHandler)
	mux.Handle("POST /tasks", authMiddleware(http.HandlerFunc(taskHandler.CreateTask)))
	mux.Handle("GET /tasks", authMiddleware(http.HandlerFunc(taskHandler.GetTask)))
	mux.Handle("GET /tasks/{id}", authMiddleware(http.HandlerFunc(taskHandler.GetTaskByID)))
	mux.Handle("PUT /tasks/{id}", authMiddleware(http.HandlerFunc(taskHandler.UpdateTaskByID)))
	mux.Handle("DELETE /tasks/{id}", authMiddleware(http.HandlerFunc(taskHandler.DeleteTaskByID)))
	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)

	log.Println("Starting server on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}

}
