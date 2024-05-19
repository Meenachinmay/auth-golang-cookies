package handlers

import (
	"auth-golang-cookies/models"
	"auth-golang-cookies/utils"
	"context"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Claims struct {
	Email  string    `json:"email"`
	UserId uuid.UUID `json:"userId"`
	jwt.StandardClaims
}

type JWTOutput struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expires"`
}

type SessionData struct {
	Token  string    `json:"token"`
	UserId uuid.UUID `json:"userId"`
}

func (lac *LocalApiConfig) SignInHandler(c *gin.Context) {
	var userToAuth models.UserToAuth

	if err := c.ShouldBindJSON(&userToAuth); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// insert validation here
	validationErrors := utils.ValidateUserToAuth(userToAuth)
	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": validationErrors,
		})
		return
	}

	// fetch the user here from the database to check if user is exists or not
	foundUser, err := lac.DB.FindUserByEmail(c, userToAuth.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "No user found",
		})
		return
	}

	if foundUser.Password != userToAuth.Password {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "password is invalid",
		})
		return
	}

	expirationTime := time.Now().Add(60 * time.Minute)
	claims := &Claims{
		Email:  userToAuth.Email,
		UserId: foundUser.ID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	sessionID := uuid.New().String()

	sessionData := map[string]interface{}{
		"token":  tokenString,
		"userId": foundUser.ID,
	}

	sessionDataJSON, err := json.Marshal(sessionData)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to encode the session data into the session data",
		})
		return
	}

	err = lac.RedisClient.Set(c, sessionID, sessionDataJSON, time.Until(expirationTime)).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save the session data to the redis",
		})
		return
	}

	onlineUserData := map[string]interface{}{
		"username": foundUser.Name,
		"userId":   foundUser.ID,
	}

	onlineUserDataJSON, err := json.Marshal(onlineUserData)

	// create a Redis key sepcifically for tracking loggedin Users in real-time
	onlineKey := "onlineUser:" + sessionID

	err = lac.RedisClient.Set(c, onlineKey, onlineUserDataJSON, time.Until(expirationTime)).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to mark user as online" + err.Error(),
		})
		return
	}

	c.SetCookie("session_id", sessionID, int(time.Until(expirationTime).Seconds()), "/", "localhost", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"expires": expirationTime,
		"token":   tokenString,
		"userId":  foundUser.ID,
	})
}

func (lac *LocalApiConfig) LogoutHandler(c *gin.Context) {
	// Retrieve the session from the cookies first
	sessionID, err := c.Cookie("session_id")

	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized request",
		})
		return
	}

	err = lac.RedisClient.Del(c, sessionID).Err()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "failed to end session",
		})
		return
	}

	c.SetCookie("session_id", "", -1, "/", "", false, true)

	// remove the onlineUser key from the redis to handle conflict and data repeatition
	onlineKey := "onlineUser:" + sessionID
	err = lac.RedisClient.Del(c, onlineKey).Err()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "failed to remove online user from redis" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"error": "Logged out successfully",
	})
}

func (lac *LocalApiConfig) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized - no session",
			})
			return
		}

		sessionDataJSON, err := lac.RedisClient.Get(c, sessionID).Result()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired session",
			})
			return
		}

		var sessionData SessionData
		err = json.Unmarshal([]byte(sessionDataJSON), &sessionData)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Failed to decode the session data",
			})
			return
		}

		//
		token, err := jwt.ParseWithClaims(sessionData.Token, &Claims{},
			func(token *jwt.Token) (interface{}, error) {
				return []byte(os.Getenv("JWT_SECRET")), nil
			})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			return
		}

		c.Set("userId", sessionData.UserId)
		c.Next()
	}
}

func (lac *LocalApiConfig) HandlerAuthRoute(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Authenticated routes are working",
	})
}

func (lac *LocalApiConfig) HandlerFetchOnlineUsers(c *gin.Context) {
	// fetch all the keys for online users
	keys, err := lac.RedisClient.Keys(c, "onlineUser:*").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch the keys from redis" + err.Error(),
		})
		return
	}

	if len(keys) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":     "No online users found",
			"onlineUsers": nil,
		})
		return
	}

	// use redis pipeline to fetch all user's data at once
	pipe := lac.RedisClient.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.Get(c, key)
	}
	_, err = pipe.Exec(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch user's data from redis pipe" + err.Error(),
		})
		return
	}

	// prepare a slice to hold user data
	onlineUsers := make([]map[string]interface{}, 0, len(keys))

	// get the data from the pipe to the slice
	for _, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			continue // you can handle the error here accordingly, i am leaving for later work
		}

		var userData map[string]interface{}
		err = json.Unmarshal([]byte(data), &userData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to fetch unmarshal the data got from redis" + err.Error(),
			})
			return
		}

		onlineUsers = append(onlineUsers, userData)
	}

	// send to client
	c.JSON(http.StatusOK, gin.H{
		"message":     "OK",
		"onlineUsers": onlineUsers,
	})
}

func (lac *LocalApiConfig) HandlerPasswordReset(c *gin.Context) {

	var emailType models.EmailType

	if err := c.ShouldBindJSON(&emailType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to parse email type",
		})
		return
	}

	res, err := lac.HandlerSendEmail(emailType)
	if err != nil {
		log.Printf("error sending email: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to send email",
		})
		return
	}

	if res.StatusCode >= 300 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "email service responded with an error: " + res.Body,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Email sent successfully",
		"result":  res,
	})
}

func (lac *LocalApiConfig) SignInHandlerKafka(data []byte) RespondToKafkaConsumerMessage {
	var userToAuth models.UserToAuth

	if err := json.Unmarshal(data, &userToAuth); err != nil {
		return RespondToKafkaConsumerMessage{
			Status:  http.StatusBadRequest,
			Message: gin.H{"error": err.Error()},
		}
	}

	// insert validation here
	validationErrors := utils.ValidateUserToAuth(userToAuth)
	if len(validationErrors) > 0 {
		return RespondToKafkaConsumerMessage{
			Status:  http.StatusBadRequest,
			Message: gin.H{"error": strings.Join(validationErrors, ", ")},
		}
	}

	// fetch the user here from the database to check if user is existing or not
	foundUser, err := lac.DB.FindUserByEmail(context.Background(), userToAuth.Email)
	if err != nil {
		return RespondToKafkaConsumerMessage{
			Status:  http.StatusInternalServerError,
			Message: gin.H{"error": "user not found" + err.Error()},
		}
	}

	if foundUser.Password != userToAuth.Password {
		return RespondToKafkaConsumerMessage{
			Status:  http.StatusUnauthorized,
			Message: gin.H{"error": "passwords don't match"},
		}
	}

	expirationTime := time.Now().Add(60 * time.Minute)
	claims := &Claims{
		Email:  userToAuth.Email,
		UserId: foundUser.ID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))

	if err != nil {
		return RespondToKafkaConsumerMessage{
			Status: http.StatusInternalServerError,
			Message: gin.H{
				"error": "failed to create token" + err.Error(),
			},
		}
	}

	sessionID := uuid.New().String()

	sessionData := map[string]interface{}{
		"token":  tokenString,
		"userId": foundUser.ID,
	}

	sessionDataJSON, err := json.Marshal(sessionData)

	if err != nil {
		return RespondToKafkaConsumerMessage{
			Status:  http.StatusInternalServerError,
			Message: gin.H{"error": "failed to encode session data" + err.Error()},
		}
	}

	err = lac.RedisClient.Set(context.Background(), sessionID, sessionDataJSON, time.Until(expirationTime)).Err()
	if err != nil {
		return RespondToKafkaConsumerMessage{
			Status:  http.StatusInternalServerError,
			Message: gin.H{"error": "failed to save session data to redis" + err.Error()},
		}
	}

	onlineUserData := map[string]interface{}{
		"username": foundUser.Name,
		"userId":   foundUser.ID,
	}

	onlineUserDataJSON, err := json.Marshal(onlineUserData)

	// create a Redis key specifically for tracking logged Users in real-time
	onlineKey := "onlineUser:" + sessionID

	err = lac.RedisClient.Set(context.Background(), onlineKey, onlineUserDataJSON, time.Until(expirationTime)).Err()
	if err != nil {
		return RespondToKafkaConsumerMessage{
			Status:  http.StatusInternalServerError,
			Message: gin.H{"error": "failed to mark user online" + err.Error()},
		}
	}

	//c.SetCookie("session_id", sessionID, int(time.Until(expirationTime).Seconds()), "/", "localhost", false, true)

	return RespondToKafkaConsumerMessage{
		Status: http.StatusOK,
		Message: map[string]interface{}{
			"message":   "Login successful",
			"expires":   expirationTime,
			"token":     tokenString,
			"userId":    foundUser.ID,
			"sessionId": sessionID,
		},
	}
}
