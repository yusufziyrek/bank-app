package controller

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/yusufziyrek/bank-app/internal/controller/dto"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/service"
)

type AuthController struct {
	svc       service.UserService
	jwtSecret string
	jwtTTL    time.Duration
}

func NewAuthController(svc service.UserService, jwtSecret string, jwtTTL time.Duration) *AuthController {
	return &AuthController{
		svc:       svc,
		jwtSecret: jwtSecret,
		jwtTTL:    jwtTTL,
	}
}

func (a *AuthController) Register(c echo.Context) error {
	var req dto.CreateUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	user := model.User{FullName: req.FullName, Email: req.Email, PasswordHash: req.Password}
	if err := a.svc.CreateUser(c.Request().Context(), &user); err != nil {
		return handleServiceError(c, err, "register")
	}
	token, exp, err := a.issueToken(user)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, "TOKEN_ERROR", "Token creation failed", err.Error())
	}
	refreshToken, refreshExp, err := a.svc.GenerateRefreshToken(c.Request().Context(), user.ID)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, "REFRESH_TOKEN_ERROR", "Refresh token creation failed", err.Error())
	}
	return c.JSON(http.StatusCreated, dto.AuthResponse{
		Token:        token,
		ExpiresAt:    exp,
		RefreshToken: refreshToken,
		RefreshExp:   refreshExp,
		User:         dto.UserResponseFromModel(user),
	})
}

func (a *AuthController) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	user, err := a.svc.AuthenticateUser(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return handleServiceError(c, err, "login")
	}
	token, exp, err := a.issueToken(user)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, "TOKEN_ERROR", "Token creation failed", err.Error())
	}
	refreshToken, refreshExp, err := a.svc.GenerateRefreshToken(c.Request().Context(), user.ID)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, "REFRESH_TOKEN_ERROR", "Refresh token creation failed", err.Error())
	}
	return c.JSON(http.StatusOK, dto.AuthResponse{
		Token:        token,
		ExpiresAt:    exp,
		RefreshToken: refreshToken,
		RefreshExp:   refreshExp,
		User:         dto.UserResponseFromModel(user),
	})
}

func (a *AuthController) Refresh(c echo.Context) error {
	var req dto.RefreshRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	userID, err := a.svc.ValidateRefreshToken(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Refresh token invalid or expired", "")
	}
	user, err := a.svc.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		return handleServiceError(c, err, "refresh token user")
	}
	token, exp, err := a.issueToken(user)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, "TOKEN_ERROR", "Token creation failed", err.Error())
	}
	return c.JSON(http.StatusOK, dto.RefreshResponse{
		Token:     token,
		ExpiresAt: exp,
	})
}

func (a *AuthController) issueToken(u model.User) (string, time.Time, error) {
	exp := time.Now().Add(a.jwtTTL)
	claims := jwt.MapClaims{"sub": u.ID, "exp": exp.Unix(), "role": u.Role}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString([]byte(a.jwtSecret))
	return s, exp, err
}
