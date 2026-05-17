package auth

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

type IAuthService interface {
	SendCode(ctx context.Context, email string) error
}

type AuthService struct {
	client  AuthClient
	logger  *slog.Logger
	limiter *redis_rate.Limiter
	rdb     *redis.Client
}

func NewService(authClient AuthClient, limiter *redis_rate.Limiter, rdb *redis.Client, logger *slog.Logger) IAuthService {
	return &AuthService{
		client:  authClient,
		logger:  logger,
		limiter: limiter,
		rdb:     rdb,
	}
}

func (s *AuthService) SendCode(ctx context.Context, email string) error {
	limit := redis_rate.Limit{
		Rate:   1,
		Burst:  1,
		Period: time.Minute / 10,
	}
	res, err := s.limiter.Allow(ctx, "sendcode:"+email, limit)
	if err != nil || res.Allowed != 1 {
		return fmt.Errorf("Too many reqests")
	}

	n := rand.Intn(999999)
	code := fmt.Sprintf(strconv.Itoa(n))

	if err := s.rdb.Set(ctx, "code:"+email, code, 10000*time.Minute).Err(); err != nil {
		s.logger.Error("redis set failed", "email", email, "error", err)
		return fmt.Errorf("Internal error")
	}

	return nil
}
