package handlers

import (
	"auth-golang-cookies/models"
	"auth-golang-cookies/utils"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"os"
	"time"
)

type Claims struct {
	Email string `json:"email"`
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

	expirationTime := time.Now().Add(10 * time.Minute)
	claims := &Claims{
		Email: userToAuth.Email,
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

	c.SetCookie("session_id", sessionID, int(time.Until(expirationTime).Seconds()), "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"expires": expirationTime,
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

	c.JSON(http.StatusOK, gin.H{
		"error": "Logged out successfully",
	})
}

func (lac *LocalApiConfig) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Unautorized - no session",
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
				"error": "Failed to decode the sessiondata",
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
