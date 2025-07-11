package infrastructure

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/repository"
)

// TestUserRepository UserRepository için test suite'i
func TestUserRepository(t *testing.T) {
	// Test veritabanı bağlantısı
	ctx := context.Background()

	// Test için PostgreSQL bağlantısı (gerçek test ortamında test container kullanılmalı)
	pool, err := pgxpool.New(ctx, "postgres://postgres:1234@localhost:5432/bankapp_test?sslmode=disable")
	if err != nil {
		t.Skipf("Test database connection failed: %v", err)
	}
	defer pool.Close()

	// Test veritabanını başlat
	err = InitializeTestDatabase(ctx, pool)
	require.NoError(t, err)

	// Test sonunda temizlik
	t.Cleanup(func() {
		ClearTestDatabase(ctx, pool)
	})

	repo := repository.NewUserRepository(pool)

	t.Run("GetAllUsers", func(t *testing.T) {
		users, err := repo.GetAllUsers(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, users)
		assert.Len(t, users, 3) // 3 test kullanıcısı

		// Kullanıcıların doğru alanları kontrol et
		for _, user := range users {
			assert.NotZero(t, user.ID)
			assert.NotEmpty(t, user.FullName)
			assert.NotEmpty(t, user.Email)
			assert.NotEmpty(t, user.PasswordHash)
			assert.NotZero(t, user.CreatedAt)
			assert.NotZero(t, user.UpdatedAt)
		}
	})

	t.Run("GetUserByID_Success", func(t *testing.T) {
		// Önce test kullanıcısını al
		testUser, err := GetTestUserByEmail(ctx, pool, "test1@example.com")
		require.NoError(t, err)

		user, err := repo.GetUserByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, user.ID)
		assert.Equal(t, testUser.Email, user.Email)
		assert.Equal(t, testUser.FullName, user.FullName)
	})

	t.Run("GetUserByID_NotFound", func(t *testing.T) {
		user, err := repo.GetUserByID(ctx, 99999)
		assert.Error(t, err)
		assert.ErrorIs(t, err, pgx.ErrNoRows)
		assert.Empty(t, user)
	})

	t.Run("GetUserByEmail_Success", func(t *testing.T) {
		user, err := repo.GetUserByEmail(ctx, "test1@example.com")
		require.NoError(t, err)
		assert.Equal(t, "test1@example.com", user.Email)
		assert.Equal(t, "Test User 1", user.FullName)
	})

	t.Run("GetUserByEmail_NotFound", func(t *testing.T) {
		user, err := repo.GetUserByEmail(ctx, "nonexistent@example.com")
		assert.Error(t, err)
		assert.ErrorIs(t, err, pgx.ErrNoRows)
		assert.Empty(t, user)
	})

	t.Run("AddUser_Success", func(t *testing.T) {
		newUser := &model.User{
			FullName:     "New Test User",
			Email:        "newuser@example.com",
			PasswordHash: "hashedpassword",
			Role:         "user",
			IsActive:     true,
		}

		err := repo.AddUser(ctx, newUser)
		require.NoError(t, err)
		assert.NotZero(t, newUser.ID)

		// Kullanıcının gerçekten eklendiğini kontrol et
		addedUser, err := repo.GetUserByID(ctx, newUser.ID)
		require.NoError(t, err)
		assert.Equal(t, newUser.Email, addedUser.Email)
		assert.Equal(t, newUser.FullName, addedUser.FullName)
	})

	t.Run("AddUser_DuplicateEmail", func(t *testing.T) {
		duplicateUser := &model.User{
			FullName:     "Duplicate User",
			Email:        "test1@example.com", // Zaten var olan email
			PasswordHash: "hashedpassword",
			Role:         "user",
			IsActive:     true,
		}

		err := repo.AddUser(ctx, duplicateUser)
		assert.Error(t, err)
		// PostgreSQL unique constraint violation
		assert.Contains(t, err.Error(), "23505")
	})

	t.Run("UpdateUserEmail_Success", func(t *testing.T) {
		// Önce test kullanıcısını al
		testUser, err := GetTestUserByEmail(ctx, pool, "test2@example.com")
		require.NoError(t, err)

		newEmail := "updated@example.com"
		err = repo.UpdateUserEmail(ctx, testUser.ID, newEmail)
		require.NoError(t, err)

		// Güncellemenin başarılı olduğunu kontrol et
		updatedUser, err := repo.GetUserByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, newEmail, updatedUser.Email)
		assert.True(t, updatedUser.UpdatedAt.After(testUser.UpdatedAt))
	})

	t.Run("UpdateUserEmail_UserNotFound", func(t *testing.T) {
		err := repo.UpdateUserEmail(ctx, 99999, "newemail@example.com")
		assert.Error(t, err)
		assert.ErrorIs(t, err, pgx.ErrNoRows)
	})

	t.Run("UpdateUserEmail_DuplicateEmail", func(t *testing.T) {
		// Önce test kullanıcısını al
		testUser, err := GetTestUserByEmail(ctx, pool, "test1@example.com")
		require.NoError(t, err)

		// Zaten var olan bir email'e güncelle
		err = repo.UpdateUserEmail(ctx, testUser.ID, "test2@example.com")
		assert.Error(t, err)
		// PostgreSQL unique constraint violation
		assert.Contains(t, err.Error(), "23505")
	})

	t.Run("UpdateUserPassword_Success", func(t *testing.T) {
		// Önce test kullanıcısını al
		testUser, err := GetTestUserByEmail(ctx, pool, "test1@example.com")
		require.NoError(t, err)

		newPasswordHash := "newhashedpassword"
		err = repo.UpdateUserPassword(ctx, testUser.ID, newPasswordHash)
		require.NoError(t, err)

		// Güncellemenin başarılı olduğunu kontrol et
		updatedUser, err := repo.GetUserByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, newPasswordHash, updatedUser.PasswordHash)
		assert.True(t, updatedUser.UpdatedAt.After(testUser.UpdatedAt))
	})

	t.Run("UpdateUserPassword_UserNotFound", func(t *testing.T) {
		err := repo.UpdateUserPassword(ctx, 99999, "newpassword")
		assert.Error(t, err)
		assert.ErrorIs(t, err, pgx.ErrNoRows)
	})

	t.Run("UpdateUserActiveStatus_Success", func(t *testing.T) {
		// Önce test kullanıcısını al
		testUser, err := GetTestUserByEmail(ctx, pool, "test1@example.com")
		require.NoError(t, err)

		newStatus := false
		err = repo.UpdateUserActiveStatus(ctx, testUser.ID, newStatus)
		require.NoError(t, err)

		// Güncellemenin başarılı olduğunu kontrol et
		updatedUser, err := repo.GetUserByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, newStatus, updatedUser.IsActive)
		assert.True(t, updatedUser.UpdatedAt.After(testUser.UpdatedAt))
	})

	t.Run("UpdateUserActiveStatus_UserNotFound", func(t *testing.T) {
		err := repo.UpdateUserActiveStatus(ctx, 99999, false)
		assert.Error(t, err)
		assert.ErrorIs(t, err, pgx.ErrNoRows)
	})

	t.Run("DeleteUserByID_Success", func(t *testing.T) {
		// Önce yeni bir kullanıcı ekle
		newUser := &model.User{
			FullName:     "User to Delete",
			Email:        "delete@example.com",
			PasswordHash: "hashedpassword",
			Role:         "user",
			IsActive:     true,
		}

		err := repo.AddUser(ctx, newUser)
		require.NoError(t, err)

		// Kullanıcıyı sil
		err = repo.DeleteUserByID(ctx, newUser.ID)
		require.NoError(t, err)

		// Kullanıcının gerçekten silindiğini kontrol et
		_, err = repo.GetUserByID(ctx, newUser.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, pgx.ErrNoRows)
	})

	t.Run("DeleteUserByID_UserNotFound", func(t *testing.T) {
		err := repo.DeleteUserByID(ctx, 99999)
		assert.Error(t, err)
		assert.ErrorIs(t, err, pgx.ErrNoRows)
	})
}

// TestUserRepositoryIntegration entegrasyon testleri
func TestUserRepositoryIntegration(t *testing.T) {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://postgres:1234@localhost:5432/bankapp_test?sslmode=disable")
	if err != nil {
		t.Skipf("Test database connection failed: %v", err)
	}
	defer pool.Close()

	err = InitializeTestDatabase(ctx, pool)
	require.NoError(t, err)

	t.Cleanup(func() {
		ClearTestDatabase(ctx, pool)
	})

	repo := repository.NewUserRepository(pool)

	t.Run("FullUserLifecycle", func(t *testing.T) {
		// 1. Kullanıcı oluştur
		newUser := &model.User{
			FullName:     "Lifecycle Test User",
			Email:        "lifecycle@example.com",
			PasswordHash: "initialpassword",
			Role:         "user",
			IsActive:     true,
		}

		err := repo.AddUser(ctx, newUser)
		require.NoError(t, err)
		assert.NotZero(t, newUser.ID)

		// 2. Kullanıcıyı getir
		retrievedUser, err := repo.GetUserByID(ctx, newUser.ID)
		require.NoError(t, err)
		assert.Equal(t, newUser.Email, retrievedUser.Email)

		// 3. Email güncelle
		err = repo.UpdateUserEmail(ctx, newUser.ID, "updated@example.com")
		require.NoError(t, err)

		// 4. Şifre güncelle
		err = repo.UpdateUserPassword(ctx, newUser.ID, "newpassword")
		require.NoError(t, err)

		// 5. Durum güncelle
		err = repo.UpdateUserActiveStatus(ctx, newUser.ID, false)
		require.NoError(t, err)

		// 6. Güncellenmiş kullanıcıyı kontrol et
		updatedUser, err := repo.GetUserByID(ctx, newUser.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated@example.com", updatedUser.Email)
		assert.Equal(t, "newpassword", updatedUser.PasswordHash)
		assert.False(t, updatedUser.IsActive)

		// 7. Kullanıcıyı sil
		err = repo.DeleteUserByID(ctx, newUser.ID)
		require.NoError(t, err)

		// 8. Kullanıcının silindiğini kontrol et
		_, err = repo.GetUserByID(ctx, newUser.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, pgx.ErrNoRows)
	})
}
