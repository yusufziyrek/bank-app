package controller

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/yusufziyrek/bank-app/common/app"
	"github.com/yusufziyrek/bank-app/internal/controller/dto"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/service"
)

type AuthController struct {
	svc       service.UserService
	jwtSecret string
	jwtTTL    time.Duration
}

func NewAuthController(svc service.UserService, cfg *app.ConfigurationManager) *AuthController {
	return &AuthController{
		svc:       svc,
		jwtSecret: cfg.JwtSecret,
		jwtTTL:    time.Duration(cfg.JwtTTL) * time.Minute,
	}
}

func (a *AuthController) Register(c echo.Context) error {
	var req dto.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return sendError(c, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON", err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	user := model.User{FullName: req.FullName, Email: req.Email, PasswordHash: req.Password}
	if err := a.svc.CreateUser(c.Request().Context(), &user); err != nil {
		if err == service.ErrEmailAlreadyRegistered {
			return sendError(c, http.StatusConflict, "EMAIL_EXISTS", err.Error(), "")
		}
		return sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not register", err.Error())
	}

	token, exp, err := a.issueToken(user)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, "TOKEN_ERROR", "Token creation failed", err.Error())
	}
	return c.JSON(http.StatusCreated, dto.AuthResponse{Token: token, ExpiresAt: exp, User: dto.UserResponseFromModel(user)})
}

func (a *AuthController) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return sendError(c, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON", err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	user, err := a.svc.AuthenticateUser(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials || err == service.ErrInactiveAccount {
			return sendError(c, http.StatusUnauthorized, "AUTH_FAILED", err.Error(), "")
		}
		return sendError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not login", err.Error())
	}

	token, exp, err := a.issueToken(user)
	if err != nil {
		return sendError(c, http.StatusInternalServerError, "TOKEN_ERROR", "Token creation failed", err.Error())
	}
	return c.JSON(http.StatusOK, dto.AuthResponse{Token: token, ExpiresAt: exp, User: dto.UserResponseFromModel(user)})
}

func (a *AuthController) issueToken(u model.User) (string, time.Time, error) {
	exp := time.Now().Add(a.jwtTTL)
	claims := jwt.MapClaims{"sub": u.ID, "exp": exp.Unix(), "role": u.Role}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString([]byte(a.jwtSecret))
	return s, exp, err
}
