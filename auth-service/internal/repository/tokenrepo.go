package repository

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

type RedisTokenRepository struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisTokenRepository(client *redis.Client) *RedisTokenRepository {
	return &RedisTokenRepository{
		client: client,
		ctx:    context.Background(),
	}
}

func (r *RedisTokenRepository) SaveRefreshToken(userID, token string, ttl time.Duration) error {
	return r.client.Set(r.ctx, "refresh:"+token, userID, ttl).Err()
}

func (r *RedisTokenRepository) GetUserIDByRefreshToken(token string) (string, error) {
	userID, err := r.client.Get(r.ctx, "refresh:"+token).Result()
	if err == redis.Nil {
		return "", ErrInvalidRefreshToken
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (r *RedisTokenRepository) DeleteRefreshToken(token string) error {
	return r.client.Del(r.ctx, "refresh:"+token).Err()
}
