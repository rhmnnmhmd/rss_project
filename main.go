package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"rhmnnmhmd/rss_project/internal/database"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	
	portString := os.Getenv("PORT")

	if portString == "" {
		log.Fatal("PORT is not found in the environment")
	}

	dbUrl := os.Getenv("DB_URL")

	if dbUrl == "" {
		log.Fatal("DB_URL is not found in the environment")
	}

	connection, err := sql.Open("postgres", dbUrl)

	if err != nil {
		log.Fatalf("Error opening database connection: %s", err)
	}

	apiConfig := apiConfig{
		DB: database.New(connection),
	}

	go startScraping(apiConfig.DB, 10, time.Minute)

	router := chi.NewRouter()

	corsOption := cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{"Link"},
		AllowCredentials: false,
		MaxAge: 300,
	}

	router.Use(cors.Handler(corsOption))
	
	v1Router := chi.NewRouter()
	v1Router.Get("/healthz", handlerReadiness)
	v1Router.Get("/err", handlerErr)
	v1Router.Post("/users", apiConfig.handlerCreateUser)
	v1Router.Get("/users", apiConfig.middlewareAuth(apiConfig.handlerGetUser))
	v1Router.Post("/feeds", apiConfig.middlewareAuth(apiConfig.handlerCreateFeed))
	v1Router.Get("/feeds", apiConfig.handlerGetFeeds)
	v1Router.Post("/feed_follows", apiConfig.middlewareAuth(apiConfig.handlerCreateFeedFollow))
	v1Router.Get("/feed_follows", apiConfig.middlewareAuth(apiConfig.handlerGetFeedFollows))
	v1Router.Delete("/feed_follows/{feedFollowID}", apiConfig.middlewareAuth(apiConfig.handlerDeleteFeedFollow))
	v1Router.Get("/posts", apiConfig.middlewareAuth(apiConfig.handlerGetPostsForUser))

	router.Mount("/v1", v1Router)

	server := &http.Server{
		Addr:    ":" + portString,
		Handler: router,
	}

	log.Printf("Server started on port %s", portString)
	err = server.ListenAndServe()

	if err != nil {
		log.Fatalf("Error starting server: %s", err)
	}
}

type apiConfig struct {
	DB *database.Queries
}