package main

import (
	"log"
	"net/http"
	"task-api/internal/auth"
	"task-api/internal/config"
	"task-api/internal/database"
	"task-api/internal/handler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	taskHandler := handler.NewTaskHandler(db)

	authHandler := handler.NewAuthHandler(db, cfg.JWTSecret)
	authMiddleware := auth.Middleware(cfg.JWTSecret)

	mux.HandleFunc("GET /health", handler.HealthHandler)
	mux.Handle("POST /tasks", authMiddleware(http.HandlerFunc(taskHandler.CreateTask)))
	mux.Handle("GET /tasks", authMiddleware(http.HandlerFunc(taskHandler.GetTask)))
	mux.Handle("GET /tasks/{id}", authMiddleware(http.HandlerFunc(taskHandler.GetTaskByID)))
	mux.Handle("PUT /tasks/{id}", authMiddleware(http.HandlerFunc(taskHandler.UpdateTaskByID)))
	mux.Handle("DELETE /tasks/{id}", authMiddleware(http.HandlerFunc(taskHandler.DeleteTaskByID)))
	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)

	log.Printf("Starting server on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}

}
