package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/yusufziyrek/bank-app/internal/controller/dto"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/service"
)

type CardController struct {
	svc       service.CardService
	accountSv service.AccountService
}

func NewCardController(svc service.CardService, accountSvc service.AccountService) *CardController {
	return &CardController{svc: svc, accountSv: accountSvc}
}

func (cc *CardController) GetAll(c echo.Context) error {
	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	cards, err := cc.svc.GetAllCards(ctx)
	if err != nil {
		return handleServiceError(c, err, "fetch cards")
	}
	return c.JSON(http.StatusOK, dto.CardsResponseFromModels(cards))
}

func (cc *CardController) GetByID(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return sendError(c, http.StatusBadRequest,
			ErrInvalidCardID, "Invalid card ID", "ID must be a positive integer")
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	card, err := cc.svc.GetCardByID(ctx, id)
	if err != nil {
		return handleServiceError(c, err, "fetch card")
	}
	return c.JSON(http.StatusOK, dto.CardResponseFromModel(card))
}

func (cc *CardController) GetMyCards(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized,
			ErrUnauthorized, "User not authenticated", err.Error())
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	cards, err := cc.svc.GetCardsByUser(ctx, userID)
	if err != nil {
		return handleServiceError(c, err, "fetch my cards")
	}
	return c.JSON(http.StatusOK, dto.CardsResponseFromModels(cards))
}

func (cc *CardController) Create(c echo.Context) error {
	userID, err := getUserIDFromToken(c)
	if err != nil {
		return sendError(c, http.StatusUnauthorized, ErrUnauthorized, "User not authenticated", err.Error())
	}

	var req dto.CreateCardRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	// Hesabın gerçekten giriş yapan kullanıcıya ait olup olmadığını kontrol et
	account, err := cc.accountSv.GetAccountByID(ctx, req.AccountID)
	if err != nil {
		return sendError(c, http.StatusBadRequest, "ACCOUNT_NOT_FOUND", "Hesap bulunamadı", err.Error())
	}
	if account.UserID != userID {
		return sendError(c, http.StatusForbidden, "FORBIDDEN", "Bu hesaba kart ekleyemezsiniz", "")
	}

	defaultExpiry := time.Now().AddDate(5, 0, 0)

	domainCard := model.Card{
		AccountID:  req.AccountID,
		CardNumber: req.CardNumber,
		CVV:        req.CVV,
		ExpiryDate: defaultExpiry,
	}

	if err := cc.svc.CreateCard(ctx, &domainCard); err != nil {
		return handleServiceError(c, err, "create card")
	}
	return c.JSON(http.StatusCreated, dto.CardResponseFromModel(domainCard))
}

func (cc *CardController) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return sendError(c, http.StatusBadRequest,
			"INVALID_CARD_ID", "Invalid card ID", "ID must be a positive integer")
	}

	var req dto.UpdateCardRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	updated := model.Card{ID: id}
	if req.CardNumber != "" {
		updated.CardNumber = req.CardNumber
	}
	if req.CVV != "" {
		updated.CVV = req.CVV
	}
	if req.ExpiryDate != nil {
		updated.ExpiryDate = *req.ExpiryDate
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	if err := cc.svc.UpdateCard(ctx, &updated); err != nil {
		return handleServiceError(c, err, "update card")
	}
	return c.NoContent(http.StatusNoContent)
}

func (cc *CardController) UpdateStatus(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return sendError(c, http.StatusBadRequest,
			"INVALID_CARD_ID", "Invalid card ID", "ID must be a positive integer")
	}

	var req dto.UpdateCardStatusRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	if err := cc.svc.UpdateCardStatus(ctx, id, req.IsActive); err != nil {
		return handleServiceError(c, err, "update card status")
	}
	return c.NoContent(http.StatusNoContent)
}

func (cc *CardController) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return sendError(c, http.StatusBadRequest,
			"INVALID_CARD_ID", "Invalid card ID", "ID must be a positive integer")
	}

	ctx, cancel := withTimeout(c.Request().Context())
	defer cancel()

	if err := cc.svc.DeleteCard(ctx, id); err != nil {
		return handleServiceError(c, err, "delete card")
	}
	return c.NoContent(http.StatusNoContent)
}
