package dto

import (
	"time"

	"github.com/yusufziyrek/bank-app/internal/model"
)

type CreateUserRequest struct {
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=100"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UpdateUserEmailRequest struct {
	NewEmail string `json:"new_email" validate:"required,email,max=255"`
}

type UpdateUserPasswordRequest struct {
	NewPassword string `json:"new_password" validate:"required,min=8,max=100"`
}

type UpdateUserStatusRequest struct {
	IsActive bool `json:"is_active" validate:"required"`
}

type UserResponse struct {
	ID        int64     `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UsersResponse struct {
	Users []UserResponse `json:"users"`
	Count int            `json:"count"`
}

type AuthResponse struct {
	Token        string       `json:"token"`
	ExpiresAt    time.Time    `json:"expires_at"`
	RefreshToken string       `json:"refresh_token"`
	RefreshExp   time.Time    `json:"refresh_expires_at"`
	User         UserResponse `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func UserResponseFromModel(u model.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		FullName:  u.FullName,
		Email:     u.Email,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func UsersResponseFromModels(users []model.User) UsersResponse {
	resp := make([]UserResponse, len(users))
	for i, u := range users {
		resp[i] = UserResponseFromModel(u)
	}
	return UsersResponse{
		Users: resp,
		Count: len(resp),
	}
}
