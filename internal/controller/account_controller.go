package controller

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/yusufziyrek/bank-app/internal/controller/dto"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/service"
)

type AccountController struct {
	svc     service.AccountService
	cardSvc service.CardService
}

func NewAccountController(svc service.AccountService, cardSvc service.CardService) *AccountController {
	return &AccountController{svc: svc, cardSvc: cardSvc}
}

func (a *AccountController) GetAll(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, ErrUnauthorized, "User not authenticated", err.Error())
	}
	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	accounts, err := a.svc.GetAllAccounts(ctx)
	if err != nil {
		return handleServiceError(c, err, "fetch accounts")
	}

	var userAccounts []dto.AccountResponse
	for _, acc := range accounts {
		if acc.UserID == userID {
			cards, _ := a.cardSvc.GetCardsByAccount(ctx, acc.ID)
			userAccounts = append(userAccounts, dto.AccountResponse{
				ID:            acc.ID,
				UserID:        acc.UserID,
				AccountNumber: acc.AccountNumber,
				Balance:       acc.Balance,
				CreatedAt:     acc.CreatedAt.Format(time.RFC3339),
				UpdatedAt:     acc.UpdatedAt.Format(time.RFC3339),
				Cards:         dto.MapCardsToDTO(cards),
			})
		}
	}
	resp := dto.AccountsResponse{
		Accounts: userAccounts,
		Count:    len(userAccounts),
	}
	return c.JSON(http.StatusOK, resp)
}

func (a *AccountController) GetByID(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, ErrUnauthorized, "User not authenticated", err.Error())
	}
	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	// parse account ID from path
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}

	acc, err := a.svc.GetAccountByID(ctx, id)
	if err != nil {
		return handleServiceError(c, err, "fetch account")
	}
	if acc.UserID != userID {
		return sendError(c, http.StatusForbidden, "FORBIDDEN", "Bu hesaba erişemezsiniz", "")
	}
	cards, _ := a.cardSvc.GetCardsByAccount(ctx, acc.ID)
	resp := dto.AccountResponse{
		ID:            acc.ID,
		UserID:        acc.UserID,
		AccountNumber: acc.AccountNumber,
		Balance:       acc.Balance,
		CreatedAt:     acc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     acc.UpdatedAt.Format(time.RFC3339),
		Cards:         dto.MapCardsToDTO(cards),
	}
	return c.JSON(http.StatusOK, resp)
}

func (a *AccountController) MyAccounts(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, ErrUnauthorized, "User not authenticated", err.Error())
	}
	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	accounts, err := a.svc.GetAccountsByUser(ctx, userID)
	if err != nil {
		return handleServiceError(c, err, "fetch user accounts")
	}

	var resp []dto.AccountResponse
	for _, acc := range accounts {
		cards, _ := a.cardSvc.GetCardsByAccount(ctx, acc.ID)
		resp = append(resp, dto.AccountResponse{
			ID:            acc.ID,
			UserID:        acc.UserID,
			AccountNumber: acc.AccountNumber,
			Balance:       acc.Balance,
			CreatedAt:     acc.CreatedAt.Format(time.RFC3339),
			UpdatedAt:     acc.UpdatedAt.Format(time.RFC3339),
			Cards:         dto.MapCardsToDTO(cards),
		})
	}
	return c.JSON(http.StatusOK, dto.AccountsResponse{
		Accounts: resp,
		Count:    len(resp),
	})
}

func (a *AccountController) Create(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err.Error())
	}
	var req dto.CreateAccountRequest
	if ok := bindAndValidate(c, &req); !ok {
		return nil
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	acc := &model.Account{
		UserID:  userID,
		Balance: req.Balance,
	}
	if err := a.svc.CreateAccount(ctx, acc); err != nil {
		return handleServiceError(c, err, "create account")
	}

	resp := dto.AccountResponse{
		ID:            acc.ID,
		UserID:        acc.UserID,
		AccountNumber: acc.AccountNumber,
		Balance:       acc.Balance,
		CreatedAt:     acc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     acc.UpdatedAt.Format(time.RFC3339),
	}
	return c.JSON(http.StatusCreated, resp)
}

func (a *AccountController) Update(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}

	var req dto.UpdateAccountRequest
	if ok := bindAndValidate(c, &req); !ok {
		return nil
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	acc := &model.Account{
		ID:      id,
		Balance: req.Balance,
	}
	if err := a.svc.UpdateAccount(ctx, acc); err != nil {
		return handleServiceError(c, err, "update account")
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *AccountController) Delete(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	if err := a.svc.DeleteAccount(ctx, id); err != nil {
		return handleServiceError(c, err, "delete account")
	}
	return c.NoContent(http.StatusNoContent)
}
