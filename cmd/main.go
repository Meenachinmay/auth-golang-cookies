package main

import (
	"auth-golang-cookies/handlers"
	"auth-golang-cookies/internal/config"
	"auth-golang-cookies/internal/database"
	"database/sql"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
)

func main() {
	// initialize the Redis here
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// Initialize the database here
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading env file.")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not found in env file or empty")
	}

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Cannot connect to database")
	}

	var testQuery int
	err = conn.QueryRow("SELECT 1").Scan(&testQuery)

	if err != nil {
		log.Fatal("Database connection test failed")
	} else {
		log.Println("Connection test query executed successfully. Database connection is working")
	}

	// setting API configuration
	apiConfig := &config.ApiConfig{
		DB:          database.New(conn),
		RedisClient: redisClient,
	}

	localApiConfig := &handlers.LocalApiConfig{
		ApiConfig: apiConfig,
	}

	// Initialize the router
	router := gin.Default()

	// add this line for the time being to allow all the origins, later we will fix it for
	// few origins only by making an array of origin allowed.
	router.Use(cors.Default())

	authorized := router.Group("/")

	authorized.Use(localApiConfig.AuthMiddleware())
	{
		authorized.GET("/health-check", localApiConfig.HandlerCheckReadiness)
	}

	router.POST("/sign-in", localApiConfig.SignInHandler)
	router.POST("/logout", localApiConfig.LogoutHandler)
	router.POST("/signup", localApiConfig.HandlerCreateUser)

	log.Fatal(router.Run(":8080"))
}
