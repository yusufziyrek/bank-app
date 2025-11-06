package controller

import (
	"net/http"
	"strconv"
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

const (
	jwtIssuer   = "bank-app"
	jwtAudience = "bank-app-clients"
)

type jwtCustomClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
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
	if ok := bindAndValidate(c, &req); !ok {
		return nil
	}

	user := model.User{FullName: req.FullName, Email: req.Email, PasswordHash: req.Password}
	if err := a.svc.CreateUser(c.Request().Context(), &user); err != nil {
		return handleServiceError(c, err, "register")
	}
	token, exp, err := a.issueToken(user)
	if err != nil {
		return handleServiceError(c, err, "token issue")
	}
	refreshToken, refreshExp, err := a.svc.GenerateRefreshToken(c.Request().Context(), user.ID)
	if err != nil {
		return handleServiceError(c, err, "refresh token issue")
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
	if ok := bindAndValidate(c, &req); !ok {
		return nil
	}

	user, err := a.svc.AuthenticateUser(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		return handleServiceError(c, err, "login")
	}
	token, exp, err := a.issueToken(user)
	if err != nil {
		return handleServiceError(c, err, "token issue")
	}
	refreshToken, refreshExp, err := a.svc.GenerateRefreshToken(c.Request().Context(), user.ID)
	if err != nil {
		return handleServiceError(c, err, "refresh token issue")
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
	if ok := bindAndValidate(c, &req); !ok {
		return nil
	}
	ctx := c.Request().Context()
	storedToken, err := a.svc.ValidateRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return handleServiceError(c, err, "refresh token validate")
	}
	user, err := a.svc.GetUserByID(ctx, storedToken.UserID)
	if err != nil {
		return handleServiceError(c, err, "refresh token user")
	}
	if !user.IsActive {
		return sendError(c, http.StatusUnauthorized, ErrUnauthorized, "Hesabınız pasif durumda", "")
	}
	token, exp, err := a.issueToken(user)
	if err != nil {
		return handleServiceError(c, err, "token issue")
	}
	newRefreshToken, refreshExp, err := a.svc.GenerateRefreshToken(ctx, user.ID)
	if err != nil {
		return handleServiceError(c, err, "refresh token rotate")
	}
	return c.JSON(http.StatusOK, dto.RefreshResponse{
		Token:        token,
		ExpiresAt:    exp,
		RefreshToken: newRefreshToken,
		RefreshExp:   refreshExp,
	})
}

func (a *AuthController) issueToken(u model.User) (string, time.Time, error) {
	exp := time.Now().Add(a.jwtTTL)
	now := time.Now()
	claims := jwtCustomClaims{
		Role: u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(u.ID, 10),
			Issuer:    jwtIssuer,
			Audience:  []string{jwtAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString([]byte(a.jwtSecret))
	return s, exp, err
}
