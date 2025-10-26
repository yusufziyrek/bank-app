package controller

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/yusufziyrek/bank-app/internal/controller/dto"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/service"
)

type TransactionController struct {
	svc       service.TransactionService
	accountSv service.AccountService
}

func NewTransactionController(svc service.TransactionService, accountSvc service.AccountService) *TransactionController {
	return &TransactionController{svc: svc, accountSv: accountSvc}
}

func (t *TransactionController) GetAll(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, ErrUnauthorized, "User not authenticated", err.Error())
	}
	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	// Fetch user's accounts to filter transactions by account ownership
	accounts, err := t.accountSv.GetAccountsByUser(ctx, userID)
	if err != nil {
		return handleServiceError(c, err, "fetch user accounts")
	}
	accountIDs := make(map[int64]struct{}, len(accounts))
	for _, a := range accounts {
		accountIDs[a.ID] = struct{}{}
	}

	transactions, err := t.svc.GetAllTransactions(ctx)
	if err != nil {
		return handleServiceError(c, err, "fetch transactions")
	}

	var userTransactions []dto.TransactionResponse
	for _, tr := range transactions {
		if _, ok := accountIDs[tr.AccountID]; ok {
			userTransactions = append(userTransactions, dto.TransactionResponse{
				ID:          tr.ID,
				AccountID:   tr.AccountID,
				ToAccountID: tr.ToAccountID,
				Amount:      tr.Amount,
				Type:        tr.Type,
				Description: tr.Description,
				CreatedAt:   tr.CreatedAt.Format(time.RFC3339),
			})
		}
	}
	resp := dto.TransactionsResponse{
		Transactions: userTransactions,
		Count:        len(userTransactions),
	}
	return c.JSON(http.StatusOK, resp)
}

func (t *TransactionController) GetByID(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	tr, err := t.svc.GetTransactionByID(ctx, id)
	if err != nil {
		return handleServiceError(c, err, "fetch transaction")
	}

	resp := dto.TransactionResponse{
		ID:          tr.ID,
		AccountID:   tr.AccountID,
		Amount:      tr.Amount,
		Type:        tr.Type,
		Description: tr.Description,
		CreatedAt:   tr.CreatedAt.Format(time.RFC3339),
	}
	return c.JSON(http.StatusOK, resp)
}

func (t *TransactionController) Create(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err.Error())
	}
	var req dto.CreateTransactionRequest
	if ok := bindAndValidate(c, &req); !ok {
		return nil
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	// Verify the account belongs to the authenticated user
	acc, err := t.accountSv.GetAccountByID(ctx, req.AccountID)
	if err != nil {
		return handleServiceError(c, err, "verify account")
	}
	if acc.UserID != userID {
		return sendError(c, http.StatusForbidden, "FORBIDDEN", "Hesap size ait değil", "")
	}

	if req.Type == "transfer" && req.Description == "" {
		return sendError(c, http.StatusBadRequest, "DESCRIPTION_REQUIRED", "Açıklama alanı zorunludur", "")
	}

	if req.Amount <= 0 || req.Amount > 10000.0 {
		return sendError(c, http.StatusBadRequest, "AMOUNT_LIMIT", "İşlem tutarı 0'dan büyük ve 10.000'den küçük olmalı", "")
	}

	tr := &model.Transaction{
		AccountID:   req.AccountID,
		ToAccountID: req.ToAccountID,
		Amount:      req.Amount,
		Type:        req.Type,
		Description: req.Description,
	}
	if err := t.svc.CreateTransaction(ctx, tr, userID); err != nil {
		return handleServiceError(c, err, "create transaction")
	}

	resp := dto.TransactionResponse{
		ID:          tr.ID,
		AccountID:   tr.AccountID,
		ToAccountID: tr.ToAccountID,
		Amount:      tr.Amount,
		Type:        tr.Type,
		Description: tr.Description,
		CreatedAt:   tr.CreatedAt.Format(time.RFC3339),
	}
	return c.JSON(http.StatusCreated, resp)
}

func (t *TransactionController) Update(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}

	var req dto.UpdateTransactionRequest
	if ok := bindAndValidate(c, &req); !ok {
		return nil
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	tr := &model.Transaction{
		ID:          id,
		ToAccountID: req.ToAccountID,
		Amount:      req.Amount,
		Type:        req.Type,
		Description: req.Description,
	}
	if err := t.svc.UpdateTransaction(ctx, tr); err != nil {
		return handleServiceError(c, err, "update transaction")
	}
	return c.NoContent(http.StatusNoContent)
}

func (t *TransactionController) Delete(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	if err := t.svc.DeleteTransaction(ctx, id); err != nil {
		return handleServiceError(c, err, "delete transaction")
	}
	return c.NoContent(http.StatusNoContent)
}
