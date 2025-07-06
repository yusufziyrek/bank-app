package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrEmailAlreadyRegistered = errors.New("email is already registered")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrInactiveAccount        = errors.New("user account is inactive")
)

type UserService interface {
	GetAllUsers(ctx context.Context) ([]model.User, error)
	GetUserByID(ctx context.Context, userID int64) (model.User, error)
	GetUserByEmail(ctx context.Context, email string) (model.User, error)
	CreateUser(ctx context.Context, user *model.User) error
	UpdateUserEmail(ctx context.Context, userID int64, newEmail string) error
	UpdateUserPassword(ctx context.Context, userID int64, newPassword string) error
	UpdateUserActiveStatus(ctx context.Context, userID int64, isActive bool) error
	DeleteUserByID(ctx context.Context, userID int64) error
	AuthenticateUser(ctx context.Context, email, password string) (model.User, error)
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) GetAllUsers(ctx context.Context) ([]model.User, error) {
	users, err := s.userRepo.GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("service: failed to get all users: %w", err)
	}
	return users, nil
}

func (s *userService) GetUserByID(ctx context.Context, userID int64) (model.User, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return model.User{}, fmt.Errorf("service: failed to get user by ID: %w", err)
	}
	return user, nil
}

func (s *userService) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return model.User{}, fmt.Errorf("service: failed to get user by email: %w", err)
	}
	return user, nil
}

func (s *userService) CreateUser(ctx context.Context, user *model.User) error {
	_, err := s.userRepo.GetUserByEmail(ctx, user.Email)
	if err == nil {
		return fmt.Errorf("service: email %s is already registered", user.Email)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("service: error checking existing email: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("service: failed to hash password: %w", err)
	}
	user.PasswordHash = string(hashedPassword)

	if user.Role == "" {
		user.Role = "user"
	}
	user.IsActive = true

	if err := s.userRepo.AddUser(ctx, user); err != nil {
		return fmt.Errorf("service: failed to create user: %w", err)
	}
	return nil
}

func (s *userService) UpdateUserEmail(ctx context.Context, userID int64, newEmail string) error {
	existingUser, err := s.userRepo.GetUserByEmail(ctx, newEmail)
	if err == nil && existingUser.ID != userID {
		return fmt.Errorf("service: new email %s is already in use by another user", newEmail)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("service: error checking new email availability: %w", err)
	}

	if err := s.userRepo.UpdateUserEmail(ctx, userID, newEmail); err != nil {
		return fmt.Errorf("service: failed to update user email: %w", err)
	}
	return nil
}

func (s *userService) UpdateUserPassword(ctx context.Context, userID int64, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("service: failed to hash new password: %w", err)
	}
	if err := s.userRepo.UpdateUserPassword(ctx, userID, string(hashedPassword)); err != nil {
		return fmt.Errorf("service: failed to update user password: %w", err)
	}
	return nil
}

func (s *userService) UpdateUserActiveStatus(ctx context.Context, userID int64, isActive bool) error {
	if err := s.userRepo.UpdateUserActiveStatus(ctx, userID, isActive); err != nil {
		return fmt.Errorf("service: failed to update user active status: %w", err)
	}
	return nil
}

func (s *userService) DeleteUserByID(ctx context.Context, userID int64) error {
	if err := s.userRepo.DeleteUserByID(ctx, userID); err != nil {
		return fmt.Errorf("service: failed to delete user: %w", err)
	}
	return nil
}

func (s *userService) AuthenticateUser(ctx context.Context, email, password string) (model.User, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return model.User{}, fmt.Errorf("service: authentication failed: invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return model.User{}, fmt.Errorf("service: authentication failed: invalid credentials")
	}

	if !user.IsActive {
		return model.User{}, fmt.Errorf("service: authentication failed: user account is inactive")
	}

	return user, nil
}
