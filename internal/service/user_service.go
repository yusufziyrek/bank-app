package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

type cachedUser struct {
	ID           int64     `json:"id"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func cachedFromModel(u model.User) cachedUser {
	return cachedUser{
		ID:           u.ID,
		FullName:     u.FullName,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func (c cachedUser) toModel() model.User {
	return model.User{
		ID:           c.ID,
		FullName:     c.FullName,
		Email:        c.Email,
		PasswordHash: c.PasswordHash,
		Role:         c.Role,
		IsActive:     c.IsActive,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

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
	return fmt.Sprintf("user:id:%d", id)
}

func (s *userService) cacheKeyEmail(email string) string {
	return fmt.Sprintf("user:email:%s", strings.ToLower(email))
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
	var cached cachedUser
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return model.User{}, false
	}
	return cached.toModel(), cached.PasswordHash != ""
}

func (s *userService) setUserCache(user model.User) {
	if s.cache == nil || user.ID == 0 {
		return
	}
	payload, err := json.Marshal(cachedFromModel(user))
	if err != nil {
		return
	}
	cacheCtx, cancel := context.WithTimeout(context.Background(), redisOpTimeout)
	defer cancel()
	keys := []string{s.cacheKey(user.ID)}
	if user.Email != "" {
		keys = append(keys, s.cacheKeyEmail(user.Email))
	}
	for _, key := range keys {
		_ = s.cache.Set(cacheCtx, key, payload, s.cacheTTL).Err()
	}
}

func (s *userService) invalidateUserCache(ctx context.Context, id int64) {
	if s.cache == nil {
		return
	}
	cacheCtx, cancel := context.WithTimeout(context.Background(), redisOpTimeout)
	defer cancel()
	key := s.cacheKey(id)
	var emailKey string
	if data, err := s.cache.Get(cacheCtx, key).Result(); err == nil {
		var cached cachedUser
		if json.Unmarshal([]byte(data), &cached) == nil && cached.Email != "" {
			emailKey = s.cacheKeyEmail(cached.Email)
		}
	}
	_ = s.cache.Del(cacheCtx, key).Err()
	if emailKey != "" {
		_ = s.cache.Del(cacheCtx, emailKey).Err()
	}
}

func (s *userService) invalidateUserEmailCache(email string) {
	if s.cache == nil || email == "" {
		return
	}
	cacheCtx, cancel := context.WithTimeout(context.Background(), redisOpTimeout)
	defer cancel()
	_ = s.cache.Del(cacheCtx, s.cacheKeyEmail(email)).Err()
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
	s.setUserCache(u)
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
	s.setUserCache(*u)
	return nil
}

func (s *userService) UpdateUserEmail(ctx context.Context, id int64, email string) error {
	existing, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("service:UpdateEmail:get: %w", err)
	}

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
	s.invalidateUserEmailCache(existing.Email)
	s.invalidateUserEmailCache(email)
	return nil
}

func (s *userService) UpdateUserPassword(ctx context.Context, id int64, pwd string) error {
	existing, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("service:UpdatePwd:get: %w", err)
	}

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
	s.invalidateUserEmailCache(existing.Email)
	return nil
}

func (s *userService) UpdateUserActiveStatus(ctx context.Context, id int64, active bool) error {
	existing, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("service:UpdateStatus:get: %w", err)
	}

	if err := s.repo.UpdateUserActiveStatus(ctx, id, active); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("service:UpdateStatus: %w", err)
	}
	s.invalidateUserCache(ctx, id)
	s.invalidateUserEmailCache(existing.Email)
	return nil
}

func (s *userService) DeleteUserByID(ctx context.Context, id int64) error {
	existing, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("service:DeleteUser:get: %w", err)
	}

	if err := s.repo.DeleteUserByID(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("service:DeleteUser: %w", err)
	}
	s.invalidateUserCache(ctx, id)
	s.invalidateUserEmailCache(existing.Email)
	return nil
}

func (s *userService) AuthenticateUser(ctx context.Context, email, pwd string) (model.User, error) {
	if cached, ok := s.getUserFromCache(ctx, s.cacheKeyEmail(email)); ok {
		if err := bcrypt.CompareHashAndPassword([]byte(cached.PasswordHash), []byte(pwd)); err != nil {
			return model.User{}, ErrInvalidCredentials
		}
		if !cached.IsActive {
			return model.User{}, ErrInactiveAccount
		}
		return cached, nil
	}

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

	s.setUserCache(u)
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
