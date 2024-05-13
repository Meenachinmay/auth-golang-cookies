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
	"github.com/pusher/pusher-http-go/v5"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
	"time"
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

	// initialize pusher here
	pusherClient := &pusher.Client{
		AppID:   os.Getenv("PUSHER_APP_ID"),
		Key:     os.Getenv("PUSHER_APP_KEY"),
		Secret:  os.Getenv("PUSHER_APP_SECRET"),
		Cluster: "ap3",
		Secure:  false,
	}

	// setting API configuration
	apiConfig := &config.ApiConfig{
		DB:           database.New(conn),
		RedisClient:  redisClient,
		PusherClient: pusherClient,
	}

	localApiConfig := &handlers.LocalApiConfig{
		ApiConfig: apiConfig,
	}

	// Initialize the router
	router := gin.Default()

	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "*"}, // Specify the exact origin of your Next.js app
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true, // Important: Must be true when credentials are included
		MaxAge:           12 * time.Hour,
	}))

	authorized := router.Group("/")

	authorized.Use(localApiConfig.AuthMiddleware())
	{
		authorized.GET("/health-check", localApiConfig.HandlerCheckReadiness)
		authorized.GET("/auth-route", localApiConfig.HandlerAuthRoute)
		authorized.POST("/notify-subscribed", localApiConfig.HandlerNotifySubscribed)
		authorized.GET("/fetch-online-users", localApiConfig.HandlerFetchOnlineUsers)
		authorized.GET("/check-ws", localApiConfig.HandlerCheckWS)
		authorized.POST("/send-message", localApiConfig.HandlerSendMessage)
		authorized.POST("/logout", localApiConfig.LogoutHandler)
	}

	router.POST("/sign-in", localApiConfig.SignInHandler)
	router.POST("/pusher/auth", localApiConfig.HandlerPusherAuth)
	router.POST("/signup", localApiConfig.HandlerCreateUser)
	router.POST("/send-email", localApiConfig.HandlerPasswordReset)

	log.Fatal(router.Run(":8080"))
}
