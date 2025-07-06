package dto

import "time"

type CreateUserRequest struct {
	FullName string `json:"full_name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UpdateUserEmailRequest struct {
	NewEmail string `json:"new_email" validate:"required,email"`
}

type UpdateUserPasswordRequest struct {
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type UpdateUserStatusRequest struct {
	IsActive bool `json:"is_active"`
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
