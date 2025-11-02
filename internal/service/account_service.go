package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/repository"
)

type AccountService interface {
	GetAllAccounts(ctx context.Context) ([]model.Account, error)
	GetAccountByID(ctx context.Context, id int64) (model.Account, error)
	GetAccountsByUser(ctx context.Context, userID int64) ([]model.Account, error)
	CreateAccount(ctx context.Context, a *model.Account) error
	UpdateAccount(ctx context.Context, a *model.Account) error
	DeleteAccount(ctx context.Context, id int64) error
	NotifyAccountBalanceChanged(ctx context.Context, account model.Account)
}

type accountService struct {
	repo     repository.AccountRepository
	cache    *redis.Client
	cacheTTL time.Duration
}

func NewAccountService(r repository.AccountRepository) AccountService {
	return newAccountService(r, nil, 0)
}

func NewAccountServiceWithCache(r repository.AccountRepository, cache *redis.Client, ttl time.Duration) AccountService {
	return newAccountService(r, cache, ttl)
}

func newAccountService(r repository.AccountRepository, cache *redis.Client, ttl time.Duration) AccountService {
	svc := &accountService{repo: r, cache: cache}
	if cache != nil {
		if ttl <= 0 {
			ttl = 5 * time.Minute
		}
		svc.cacheTTL = ttl
	}
	return svc
}

type cachedAccount struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	AccountNumber string    `json:"account_number"`
	Balance       float64   `json:"balance"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func cachedAccountFromModel(a model.Account) cachedAccount {
	return cachedAccount{
		ID:            a.ID,
		UserID:        a.UserID,
		AccountNumber: a.AccountNumber,
		Balance:       a.Balance,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

func (c cachedAccount) toModel() model.Account {
	return model.Account{
		ID:            c.ID,
		UserID:        c.UserID,
		AccountNumber: c.AccountNumber,
		Balance:       c.Balance,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

func (s *accountService) cacheKeyAccount(id int64) string {
	return fmt.Sprintf("account:id:%d", id)
}

func (s *accountService) cacheKeyUserAccounts(userID int64) string {
	return fmt.Sprintf("account:user:%d", userID)
}

func (s *accountService) getAccountFromCache(ctx context.Context, key string) (model.Account, bool) {
	if s.cache == nil {
		return model.Account{}, false
	}
	cacheCtx, cancel := context.WithTimeout(ctx, redisOpTimeout)
	defer cancel()

	data, err := s.cache.Get(cacheCtx, key).Result()
	if err != nil {
		return model.Account{}, false
	}
	var cached cachedAccount
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return model.Account{}, false
	}
	return cached.toModel(), true
}

func (s *accountService) setAccountCache(a model.Account) {
	if s.cache == nil || a.ID == 0 {
		return
	}
	payload, err := json.Marshal(cachedAccountFromModel(a))
	if err != nil {
		return
	}
	cacheCtx, cancel := context.WithTimeout(context.Background(), redisOpTimeout)
	defer cancel()
	_ = s.cache.Set(cacheCtx, s.cacheKeyAccount(a.ID), payload, s.cacheTTL).Err()
}

func (s *accountService) invalidateAccountCache(id int64) {
	if s.cache == nil {
		return
	}
	cacheCtx, cancel := context.WithTimeout(context.Background(), redisOpTimeout)
	defer cancel()
	_ = s.cache.Del(cacheCtx, s.cacheKeyAccount(id)).Err()
}

func (s *accountService) getAccountsByUserFromCache(ctx context.Context, userID int64) ([]model.Account, bool) {
	if s.cache == nil {
		return nil, false
	}
	cacheCtx, cancel := context.WithTimeout(ctx, redisOpTimeout)
	defer cancel()
	data, err := s.cache.Get(cacheCtx, s.cacheKeyUserAccounts(userID)).Result()
	if err != nil {
		return nil, false
	}
	var cached []cachedAccount
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, false
	}
	accounts := make([]model.Account, len(cached))
	for i, item := range cached {
		accounts[i] = item.toModel()
	}
	return accounts, true
}

func (s *accountService) setAccountsByUserCache(userID int64, accounts []model.Account) {
	if s.cache == nil {
		return
	}
	cached := make([]cachedAccount, len(accounts))
	for i, a := range accounts {
		cached[i] = cachedAccountFromModel(a)
	}
	payload, err := json.Marshal(cached)
	if err != nil {
		return
	}
	cacheCtx, cancel := context.WithTimeout(context.Background(), redisOpTimeout)
	defer cancel()
	_ = s.cache.Set(cacheCtx, s.cacheKeyUserAccounts(userID), payload, s.cacheTTL).Err()
}

func (s *accountService) invalidateAccountsByUserCache(userID int64) {
	if s.cache == nil {
		return
	}
	cacheCtx, cancel := context.WithTimeout(context.Background(), redisOpTimeout)
	defer cancel()
	_ = s.cache.Del(cacheCtx, s.cacheKeyUserAccounts(userID)).Err()
}

func (s *accountService) GetAllAccounts(ctx context.Context) ([]model.Account, error) {
	return s.repo.GetAllAccounts(ctx)
}

func (s *accountService) GetAccountByID(ctx context.Context, id int64) (model.Account, error) {
	if acc, ok := s.getAccountFromCache(ctx, s.cacheKeyAccount(id)); ok {
		return acc, nil
	}
	a, err := s.repo.GetAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return a, ErrAccountNotFound
		}
		return a, fmt.Errorf("service:GetAccountByID: %w", err)
	}
	s.setAccountCache(a)
	return a, nil
}

func (s *accountService) GetAccountsByUser(ctx context.Context, userID int64) ([]model.Account, error) {
	if accounts, ok := s.getAccountsByUserFromCache(ctx, userID); ok {
		return accounts, nil
	}
	accounts, err := s.repo.GetAccountsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("service:GetAccountsByUser: %w", err)
	}
	s.setAccountsByUserCache(userID, accounts)
	for _, acc := range accounts {
		s.setAccountCache(acc)
	}
	return accounts, nil
}

func (s *accountService) CreateAccount(ctx context.Context, a *model.Account) error {
	existing, err := s.repo.GetAccountsByUser(ctx, a.UserID)
	if err != nil {
		return fmt.Errorf("service:getAccountsByUser: %w", err)
	}
	if len(existing) >= 3 {
		return ErrMaxAccountsExceeded
	}

	num, err := generateAccountNumber()
	if err != nil {
		return fmt.Errorf("service:generateAccountNumber: %w", err)
	}
	a.AccountNumber = num

	if err := s.repo.AddAccount(ctx, a); err != nil {
		return fmt.Errorf("service:AddAccount: %w", err)
	}
	s.setAccountCache(*a)
	s.invalidateAccountsByUserCache(a.UserID)
	return nil
}

func (s *accountService) UpdateAccount(ctx context.Context, a *model.Account) error {
	acc, err := s.repo.GetAccountByID(ctx, a.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("service:GetAccountByID: %w", err)
	}
	acc.Balance = a.Balance
	if err := s.repo.UpdateAccount(ctx, &acc); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("service:UpdateAccount: %w", err)
	}
	s.setAccountCache(acc)
	s.invalidateAccountsByUserCache(acc.UserID)
	return nil
}

func (s *accountService) DeleteAccount(ctx context.Context, id int64) error {
	acc, err := s.repo.GetAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("service:GetAccountByID: %w", err)
	}

	if err := s.repo.DeleteAccount(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("service:DeleteAccount: %w", err)
	}
	s.invalidateAccountCache(id)
	s.invalidateAccountsByUserCache(acc.UserID)
	return nil
}

func (s *accountService) NotifyAccountBalanceChanged(ctx context.Context, account model.Account) {
	if account.ID == 0 {
		return
	}
	s.invalidateAccountsByUserCache(account.UserID)
	s.setAccountCache(account)
}

func generateAccountNumber() (string, error) {
	b := make([]byte, 10)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("TR%020d", new(big.Int).SetBytes(b)), nil
}
