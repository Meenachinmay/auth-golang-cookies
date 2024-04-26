package config

import (
	"auth-golang-cookies/internal/database"
	"github.com/redis/go-redis/v9"
)

type ApiConfig struct {
	DB          *database.Queries
	RedisClient *redis.Client
}
