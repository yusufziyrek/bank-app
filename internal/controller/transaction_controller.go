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
	admin := isAdmin(c)

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	var transactions []model.Transaction
	if admin {
		transactions, err = t.svc.GetAllTransactions(ctx)
	} else {
		accounts, acctErr := t.accountSv.GetAccountsByUser(ctx, userID)
		if acctErr != nil {
			return handleServiceError(c, acctErr, "fetch user accounts")
		}
		if len(accounts) == 0 {
			return c.JSON(http.StatusOK, dto.TransactionsResponse{Transactions: []dto.TransactionResponse{}, Count: 0})
		}
		ids := make([]int64, 0, len(accounts))
		for _, a := range accounts {
			ids = append(ids, a.ID)
		}
		transactions, err = t.svc.GetTransactionsByAccountIDs(ctx, ids)
	}
	if err != nil {
		return handleServiceError(c, err, "fetch transactions")
	}

	responses := make([]dto.TransactionResponse, 0, len(transactions))
	for _, tr := range transactions {
		responses = append(responses, dto.TransactionResponse{
			ID:          tr.ID,
			AccountID:   tr.AccountID,
			ToAccountID: tr.ToAccountID,
			Amount:      tr.Amount,
			Type:        tr.Type,
			Description: tr.Description,
			CreatedAt:   tr.CreatedAt.Format(time.RFC3339),
		})
	}
	return c.JSON(http.StatusOK, dto.TransactionsResponse{
		Transactions: responses,
		Count:        len(responses),
	})
}

func (t *TransactionController) GetByID(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, ErrUnauthorized, "User not authenticated", err.Error())
	}
	admin := isAdmin(c)

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	tr, err := t.svc.GetTransactionByID(ctx, id)
	if err != nil {
		return handleServiceError(c, err, "fetch transaction")
	}
	if !admin {
		account, err := t.accountSv.GetAccountByID(ctx, tr.AccountID)
		if err != nil {
			return handleServiceError(c, err, "fetch account")
		}
		if account.UserID != userID {
			return sendError(c, http.StatusForbidden, ErrForbidden, "Bu işlem üzerinde yetkiniz yok", "")
		}
	}

	return c.JSON(http.StatusOK, dto.TransactionResponse{
		ID:          tr.ID,
		AccountID:   tr.AccountID,
		ToAccountID: tr.ToAccountID,
		Amount:      tr.Amount,
		Type:        tr.Type,
		Description: tr.Description,
		CreatedAt:   tr.CreatedAt.Format(time.RFC3339),
	})
}

func (t *TransactionController) Create(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err.Error())
	}
	admin := isAdmin(c)

	var req dto.CreateTransactionRequest
	if ok := bindAndValidate(c, &req); !ok {
		return nil
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	acc, err := t.accountSv.GetAccountByID(ctx, req.AccountID)
	if err != nil {
		return handleServiceError(c, err, "verify account")
	}
	if !admin && acc.UserID != userID {
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
	ownerID := userID
	if admin {
		ownerID = acc.UserID
	}
	if err := t.svc.CreateTransaction(ctx, tr, ownerID); err != nil {
		return handleServiceError(c, err, "create transaction")
	}

	return c.JSON(http.StatusCreated, dto.TransactionResponse{
		ID:          tr.ID,
		AccountID:   tr.AccountID,
		ToAccountID: tr.ToAccountID,
		Amount:      tr.Amount,
		Type:        tr.Type,
		Description: tr.Description,
		CreatedAt:   tr.CreatedAt.Format(time.RFC3339),
	})
}

func (t *TransactionController) Update(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, ErrUnauthorized, "User not authenticated", err.Error())
	}
	admin := isAdmin(c)

	var req dto.UpdateTransactionRequest
	if ok := bindAndValidate(c, &req); !ok {
		return nil
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	existing, err := t.svc.GetTransactionByID(ctx, id)
	if err != nil {
		return handleServiceError(c, err, "fetch transaction")
	}
	if !admin {
		account, err := t.accountSv.GetAccountByID(ctx, existing.AccountID)
		if err != nil {
			return handleServiceError(c, err, "fetch account")
		}
		if account.UserID != userID {
			return sendError(c, http.StatusForbidden, ErrForbidden, "Bu işlem üzerinde yetkiniz yok", "")
		}
	}

	updated := &model.Transaction{
		ID:          id,
		ToAccountID: req.ToAccountID,
		Amount:      req.Amount,
		Type:        req.Type,
		Description: req.Description,
	}
	if err := t.svc.UpdateTransaction(ctx, updated); err != nil {
		return handleServiceError(c, err, "update transaction")
	}
	return c.NoContent(http.StatusNoContent)
}

func (t *TransactionController) Delete(c echo.Context) error {
	id, herr := parseID(c)
	if herr != nil {
		return c.JSON(herr.Code, herr.Message)
	}
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, ErrUnauthorized, "User not authenticated", err.Error())
	}
	admin := isAdmin(c)

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	existing, err := t.svc.GetTransactionByID(ctx, id)
	if err != nil {
		return handleServiceError(c, err, "fetch transaction")
	}
	if !admin {
		account, err := t.accountSv.GetAccountByID(ctx, existing.AccountID)
		if err != nil {
			return handleServiceError(c, err, "fetch account")
		}
		if account.UserID != userID {
			return sendError(c, http.StatusForbidden, ErrForbidden, "Bu işlem üzerinde yetkiniz yok", "")
		}
	}

	if err := t.svc.DeleteTransaction(ctx, id); err != nil {
		return handleServiceError(c, err, "delete transaction")
	}
	return c.NoContent(http.StatusNoContent)
}
