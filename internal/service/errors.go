package service

import "errors"

var (
	ErrAccountNotFound        = errors.New("account not found")
	ErrMaxAccountsExceeded    = errors.New("max accounts per user exceeded")
	ErrMaxCardsPerAccount     = errors.New("max cards per account exceeded")
	ErrCardNotFound           = errors.New("card not found")
	ErrCardAlreadyExists      = errors.New("card already exists")
	ErrTransactionNotFound    = errors.New("transaction not found")
	ErrInsufficientFunds      = errors.New("insufficient funds")
	ErrInvalidTransactionType = errors.New("invalid transaction type")
	ErrUserNotFound           = errors.New("user not found")
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrInactiveAccount        = errors.New("inactive account")
)
