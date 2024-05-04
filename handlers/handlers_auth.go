package handlers

import (
	"auth-golang-cookies/models"
	"auth-golang-cookies/utils"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"net/http"
	"os"
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
		"userId":   foundUser.ID.String(),
	}
	onlineUserDataJSON, err := json.Marshal(onlineUserData)

	// Create a Redis key specifically for tracking loggedIn Users in real-time
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

	// removing online user tracking - for current user
	onlineKey := "onlineUser:" + sessionID
	err = lac.RedisClient.Del(c, onlineKey).Err()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to remove user from online users list",
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

func (lac *LocalApiConfig) HandlerFetchActiveUsers(c *gin.Context) {
	// Fetch all keys for online users
	keys, err := lac.RedisClient.Keys(c, "onlineUser:*").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch online users from Redis: " + err.Error(),
		})
		return
	}

	if len(keys) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":     "No online users found.",
			"onlineUsers": []interface{}{},
		})
		return
	}

	// Use pipelining to fetch all user data at once
	pipe := lac.RedisClient.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.Get(c, key)
	}
	_, err = pipe.Exec(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch user data from Redis: " + err.Error(),
		})
		return
	}

	// Prepare a slice to hold user data
	onlineUsers := make([]map[string]interface{}, 0, len(keys))

	// Collect user data from pipeline results
	for _, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			continue // Optionally handle errors differently
		}

		var userData map[string]interface{}
		err = json.Unmarshal([]byte(data), &userData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch user data from Redis Pipeline: " + err.Error(),
			})
			return
		}

		onlineUsers = append(onlineUsers, userData)
	}

	// Send the collected online user data as JSON
	c.JSON(http.StatusOK, gin.H{
		"message":     "OK",
		"onlineUsers": onlineUsers,
	})
}
