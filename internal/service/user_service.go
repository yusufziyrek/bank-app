package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

const refreshTokenLength = 64
const refreshTokenTTL = 7 * 24 * time.Hour // 7 gün
const redisOpTimeout = 300 * time.Millisecond

type UserService interface {
	GetAllUsers(ctx context.Context) ([]model.User, error)
	GetUserByID(ctx context.Context, id int64) (model.User, error)
	CreateUser(ctx context.Context, u *model.User) error
	UpdateUserEmail(ctx context.Context, id int64, email string) error
	UpdateUserPassword(ctx context.Context, id int64, pwd string) error
	UpdateUserActiveStatus(ctx context.Context, id int64, isActive bool) error
	DeleteUserByID(ctx context.Context, id int64) error
	AuthenticateUser(ctx context.Context, email, pwd string) (model.User, error)
	GenerateRefreshToken(ctx context.Context, userID int64) (string, time.Time, error)
	ValidateRefreshToken(ctx context.Context, token string) (int64, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID int64) error
}

type userService struct {
	repo     repository.UserRepository
	cache    *redis.Client
	cacheTTL time.Duration
}

func NewUserService(r repository.UserRepository) UserService {
	return newUserService(r, nil, 0)
}

func NewUserServiceWithCache(r repository.UserRepository, cache *redis.Client, ttl time.Duration) UserService {
	return newUserService(r, cache, ttl)
}

func newUserService(r repository.UserRepository, cache *redis.Client, ttl time.Duration) UserService {
	service := &userService{repo: r, cache: cache}
	if cache != nil {
		if ttl <= 0 {
			ttl = 5 * time.Minute
		}
		service.cacheTTL = ttl
	}
	return service
}

func (s *userService) cacheKey(id int64) string {
	return fmt.Sprintf("user:%d", id)
}

func (s *userService) getUserFromCache(ctx context.Context, key string) (model.User, bool) {
	if s.cache == nil {
		return model.User{}, false
	}
	cacheCtx, cancel := context.WithTimeout(ctx, redisOpTimeout)
	defer cancel()

	data, err := s.cache.Get(cacheCtx, key).Result()
	if err != nil {
		return model.User{}, false
	}
	var user model.User
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return model.User{}, false
	}
	return user, true
}

func (s *userService) setUserCache(ctx context.Context, key string, user model.User) {
	if s.cache == nil {
		return
	}
	payload, err := json.Marshal(user)
	if err != nil {
		return
	}
	cacheCtx, cancel := context.WithTimeout(context.Background(), redisOpTimeout)
	defer cancel()
	_ = s.cache.Set(cacheCtx, key, payload, s.cacheTTL).Err()
}

func (s *userService) invalidateUserCache(ctx context.Context, id int64) {
	if s.cache == nil {
		return
	}
	cacheCtx, cancel := context.WithTimeout(context.Background(), redisOpTimeout)
	defer cancel()
	_ = s.cache.Del(cacheCtx, s.cacheKey(id)).Err()
}

func (s *userService) GetAllUsers(ctx context.Context) ([]model.User, error) {
	return s.repo.GetAllUsers(ctx)
}

func (s *userService) GetUserByID(ctx context.Context, id int64) (model.User, error) {
	if cached, ok := s.getUserFromCache(ctx, s.cacheKey(id)); ok {
		return cached, nil
	}

	u, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return u, ErrUserNotFound
		}
		return u, fmt.Errorf("service:GetUserByID: %w", err)
	}
	s.setUserCache(ctx, s.cacheKey(id), u)
	return u, nil
}

func (s *userService) CreateUser(ctx context.Context, u *model.User) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(u.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("service:hash: %w", err)
	}
	u.PasswordHash = string(hashed)
	if u.Role == "" {
		u.Role = "user"
	}
	u.IsActive = true

	if err := s.repo.AddUser(ctx, u); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailAlreadyRegistered
		}
		return fmt.Errorf("service:AddUser: %w", err)
	}
	s.invalidateUserCache(ctx, u.ID)
	return nil
}

func (s *userService) UpdateUserEmail(ctx context.Context, id int64, email string) error {
	if err := s.repo.UpdateUserEmail(ctx, id, email); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailAlreadyRegistered
		}
		return fmt.Errorf("service:UpdateEmail: %w", err)
	}
	s.invalidateUserCache(ctx, id)
	return nil
}

func (s *userService) UpdateUserPassword(ctx context.Context, id int64, pwd string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("service:hashPwd: %w", err)
	}
	if err := s.repo.UpdateUserPassword(ctx, id, string(hashed)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("service:UpdatePwd: %w", err)
	}
	s.invalidateUserCache(ctx, id)
	return nil
}

func (s *userService) UpdateUserActiveStatus(ctx context.Context, id int64, active bool) error {
	if err := s.repo.UpdateUserActiveStatus(ctx, id, active); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("service:UpdateStatus: %w", err)
	}
	s.invalidateUserCache(ctx, id)
	return nil
}

func (s *userService) DeleteUserByID(ctx context.Context, id int64) error {
	if err := s.repo.DeleteUserByID(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("service:DeleteUser: %w", err)
	}
	s.invalidateUserCache(ctx, id)
	return nil
}

func (s *userService) AuthenticateUser(ctx context.Context, email, pwd string) (model.User, error) {
	u, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrInvalidCredentials
		}
		return model.User{}, fmt.Errorf("service:AuthenticateUser: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(pwd)); err != nil {
		return model.User{}, ErrInvalidCredentials
	}

	if !u.IsActive {
		return model.User{}, ErrInactiveAccount
	}

	return u, nil
}

func (s *userService) GenerateRefreshToken(ctx context.Context, userID int64) (string, time.Time, error) {
	b := make([]byte, refreshTokenLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", time.Time{}, err
	}
	token := base64.URLEncoding.EncodeToString(b)
	expiresAt := time.Now().Add(refreshTokenTTL)
	rt := &model.RefreshToken{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	if err := s.repo.InsertRefreshToken(ctx, rt); err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (s *userService) ValidateRefreshToken(ctx context.Context, token string) (int64, error) {
	rt, err := s.repo.GetRefreshToken(ctx, token)
	if err != nil {
		return 0, ErrInvalidCredentials
	}
	if time.Now().After(rt.ExpiresAt) {
		_ = s.repo.DeleteRefreshToken(ctx, token)
		return 0, ErrInvalidCredentials
	}
	return rt.UserID, nil
}

func (s *userService) RevokeRefreshToken(ctx context.Context, token string) error {
	return s.repo.DeleteRefreshToken(ctx, token)
}

func (s *userService) RevokeAllUserRefreshTokens(ctx context.Context, userID int64) error {
	return s.repo.DeleteUserRefreshTokens(ctx, userID)
}
