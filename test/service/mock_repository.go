package service

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/service"
)

// MockUserRepository UserRepository için mock implementasyonu
type MockUserRepository struct {
	users  map[int64]*model.User
	emails map[string]*model.User
	mu     sync.RWMutex
	nextID int64
}

// NewMockUserRepository yeni mock repository oluşturur
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:  make(map[int64]*model.User),
		emails: make(map[string]*model.User),
		nextID: 1,
	}
}

// AddTestUser test için kullanıcı ekler
func (m *MockUserRepository) AddTestUser(user *model.User) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if user.ID == 0 {
		user.ID = m.nextID
		m.nextID++
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	m.users[user.ID] = user
	m.emails[user.Email] = user
}

// GetAllUsers tüm kullanıcıları getirir
func (m *MockUserRepository) GetAllUsers(ctx context.Context) ([]model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]model.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, *user)
	}
	return users, nil
}

// GetUserByID ID ile kullanıcı getirir
func (m *MockUserRepository) GetUserByID(ctx context.Context, id int64) (model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[id]
	if !exists {
		return model.User{}, service.ErrUserNotFound
	}
	return *user, nil
}

// GetUserByEmail email ile kullanıcı getirir
func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.emails[email]
	if !exists {
		return model.User{}, service.ErrUserNotFound
	}
	return *user, nil
}

// AddUser kullanıcı ekler
func (m *MockUserRepository) AddUser(ctx context.Context, u *model.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Email kontrolü
	if _, exists := m.emails[u.Email]; exists {
		return service.ErrEmailAlreadyRegistered
	}

	u.ID = m.nextID
	m.nextID++
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()

	m.users[u.ID] = u
	m.emails[u.Email] = u
	return nil
}

// UpdateUserEmail kullanıcı email'ini günceller
func (m *MockUserRepository) UpdateUserEmail(ctx context.Context, id int64, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return service.ErrUserNotFound
	}

	// Email kontrolü
	if _, exists := m.emails[email]; exists {
		return service.ErrEmailAlreadyRegistered
	}

	// Eski email'i sil
	delete(m.emails, user.Email)

	// Yeni email'i ekle
	user.Email = email
	user.UpdatedAt = time.Now()
	m.emails[email] = user

	return nil
}

// UpdateUserPassword kullanıcı şifresini günceller
func (m *MockUserRepository) UpdateUserPassword(ctx context.Context, id int64, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return service.ErrUserNotFound
	}

	user.PasswordHash = hash
	user.UpdatedAt = time.Now()

	return nil
}

// UpdateUserActiveStatus kullanıcı aktiflik durumunu günceller
func (m *MockUserRepository) UpdateUserActiveStatus(ctx context.Context, id int64, isActive bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return service.ErrUserNotFound
	}

	user.IsActive = isActive
	user.UpdatedAt = time.Now()

	return nil
}

// DeleteUserByID kullanıcıyı siler
func (m *MockUserRepository) DeleteUserByID(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return service.ErrUserNotFound
	}

	delete(m.users, id)
	delete(m.emails, user.Email)

	return nil
}

// WithTransaction mock transaction desteği
func (m *MockUserRepository) WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error {
	// Mock için basit implementasyon - gerçek transaction simülasyonu
	return fn(nil) // nil tx ile çağır
}

// Clear tüm test verilerini temizler
func (m *MockUserRepository) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.users = make(map[int64]*model.User)
	m.emails = make(map[string]*model.User)
	m.nextID = 1
}

// InsertRefreshToken refresh token ekler
func (m *MockUserRepository) InsertRefreshToken(ctx context.Context, rt *model.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rt.ID = m.nextID
	m.nextID++
	rt.CreatedAt = time.Now()

	return nil
}

// GetRefreshToken refresh token getirir
func (m *MockUserRepository) GetRefreshToken(ctx context.Context, token string) (model.RefreshToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Mock için basit implementasyon - her zaman geçerli token döner
	return model.RefreshToken{
		ID:        1,
		UserID:    1,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}, nil
}

// DeleteRefreshToken refresh token siler
func (m *MockUserRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Mock için basit implementasyon
	return nil
}

// DeleteUserRefreshTokens kullanıcının tüm refresh token'larını siler
func (m *MockUserRepository) DeleteUserRefreshTokens(ctx context.Context, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Mock için basit implementasyon
	return nil
}
