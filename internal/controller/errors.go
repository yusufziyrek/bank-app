package controller

const (
	ErrUnauthorized           = "UNAUTHORIZED"
	ErrInvalidUserID          = "INVALID_USER_ID"
	ErrInvalidCardID          = "INVALID_CARD_ID"
	ErrTokenError             = "TOKEN_ERROR"
	ErrRefreshTokenError      = "REFRESH_TOKEN_ERROR"
	ErrInternalServerError    = "INTERNAL_SERVER_ERROR"
	ErrUserNotFound           = "USER_NOT_FOUND"
	ErrEmailExists            = "EMAIL_EXISTS"
	ErrAuthFailed             = "AUTH_FAILED"
	ErrMaxAccountsReached     = "MAX_ACCOUNTS_REACHED"
	ErrValidationError        = "VALIDATION_ERROR"
	ErrInvalidBody            = "INVALID_BODY"
	ErrAccountNotFound        = "ACCOUNT_NOT_FOUND"
	ErrCardAlreadyExists      = "CARD_ALREADY_EXISTS"
	ErrForbidden              = "FORBIDDEN"
	ErrTransactionNotFound    = "TRANSACTION_NOT_FOUND"
	ErrInsufficientFunds      = "INSUFFICIENT_FUNDS"
	ErrInvalidTransactionType = "INVALID_TRANSACTION_TYPE"
)
