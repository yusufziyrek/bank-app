package service

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/repository"
	"github.com/yusufziyrek/bank-app/internal/service"
	"github.com/yusufziyrek/bank-app/test/infrastructure"
	"golang.org/x/crypto/bcrypt"
)

// TestUserService UserService için test suite'i
func TestUserService(t *testing.T) {
	ctx := context.Background()

	// Test veritabanı bağlantısı
	pool, err := pgxpool.New(ctx, "postgres://postgres:1234@localhost:5432/bankapp_test?sslmode=disable")
	if err != nil {
		t.Skipf("Test database connection failed: %v", err)
	}
	defer pool.Close()

	// Test veritabanını başlat
	err = infrastructure.InitializeTestDatabase(ctx, pool)
	require.NoError(t, err)

	// Test sonunda temizlik
	t.Cleanup(func() {
		infrastructure.ClearTestDatabase(ctx, pool)
	})

	repo := repository.NewUserRepository(pool)
	svc := service.NewUserService(repo)

	t.Run("GetAllUsers", func(t *testing.T) {
		users, err := svc.GetAllUsers(ctx)
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
		testUser, err := infrastructure.GetTestUserByEmail(ctx, pool, "test1@example.com")
		require.NoError(t, err)

		user, err := svc.GetUserByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, user.ID)
		assert.Equal(t, testUser.Email, user.Email)
		assert.Equal(t, testUser.FullName, user.FullName)
	})

	t.Run("GetUserByID_NotFound", func(t *testing.T) {
		user, err := svc.GetUserByID(ctx, 99999)
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
		assert.Empty(t, user)
	})

	t.Run("CreateUser_Success", func(t *testing.T) {
		newUser := &model.User{
			FullName:     "New Service User",
			Email:        "newservice@example.com",
			PasswordHash: "plainpassword", // Service'te hash'lenecek
			Role:         "",              // Service'te "user" olarak set edilecek
			IsActive:     false,           // Service'te true olarak set edilecek
		}

		err := svc.CreateUser(ctx, newUser)
		require.NoError(t, err)
		assert.NotZero(t, newUser.ID)
		assert.Equal(t, "user", newUser.Role)
		assert.True(t, newUser.IsActive)

		// Şifrenin hash'lendiğini kontrol et
		err = bcrypt.CompareHashAndPassword([]byte(newUser.PasswordHash), []byte("plainpassword"))
		assert.NoError(t, err)

		// Kullanıcının gerçekten eklendiğini kontrol et
		addedUser, err := svc.GetUserByID(ctx, newUser.ID)
		require.NoError(t, err)
		assert.Equal(t, newUser.Email, addedUser.Email)
		assert.Equal(t, newUser.FullName, addedUser.FullName)
	})

	t.Run("CreateUser_DuplicateEmail", func(t *testing.T) {
		duplicateUser := &model.User{
			FullName:     "Duplicate Service User",
			Email:        "test1@example.com", // Zaten var olan email
			PasswordHash: "plainpassword",
		}

		err := svc.CreateUser(ctx, duplicateUser)
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrEmailAlreadyRegistered)
	})

	t.Run("CreateUser_PasswordHashing", func(t *testing.T) {
		newUser := &model.User{
			FullName:     "Password Test User",
			Email:        "passwordtest@example.com",
			PasswordHash: "testpassword123",
		}

		err := svc.CreateUser(ctx, newUser)
		require.NoError(t, err)

		// Şifrenin hash'lendiğini kontrol et
		assert.NotEqual(t, "testpassword123", newUser.PasswordHash)
		err = bcrypt.CompareHashAndPassword([]byte(newUser.PasswordHash), []byte("testpassword123"))
		assert.NoError(t, err)
	})

	t.Run("UpdateUserEmail_Success", func(t *testing.T) {
		// Önce test kullanıcısını al
		testUser, err := infrastructure.GetTestUserByEmail(ctx, pool, "test2@example.com")
		require.NoError(t, err)

		newEmail := "serviceupdated@example.com"
		err = svc.UpdateUserEmail(ctx, testUser.ID, newEmail)
		require.NoError(t, err)

		// Güncellemenin başarılı olduğunu kontrol et
		updatedUser, err := svc.GetUserByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, newEmail, updatedUser.Email)
		assert.True(t, updatedUser.UpdatedAt.After(testUser.UpdatedAt))
	})

	t.Run("UpdateUserEmail_UserNotFound", func(t *testing.T) {
		err := svc.UpdateUserEmail(ctx, 99999, "newemail@example.com")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("UpdateUserEmail_DuplicateEmail", func(t *testing.T) {
		// Önce test kullanıcısını al
		testUser, err := infrastructure.GetTestUserByEmail(ctx, pool, "test1@example.com")
		require.NoError(t, err)

		// Zaten var olan bir email'e güncelle
		err = svc.UpdateUserEmail(ctx, testUser.ID, "test2@example.com")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrEmailAlreadyRegistered)
	})

	t.Run("UpdateUserPassword_Success", func(t *testing.T) {
		// Önce test kullanıcısını al
		testUser, err := infrastructure.GetTestUserByEmail(ctx, pool, "test1@example.com")
		require.NoError(t, err)

		newPassword := "newservicepassword"
		err = svc.UpdateUserPassword(ctx, testUser.ID, newPassword)
		require.NoError(t, err)

		// Güncellemenin başarılı olduğunu kontrol et
		updatedUser, err := svc.GetUserByID(ctx, testUser.ID)
		require.NoError(t, err)

		// Şifrenin hash'lendiğini kontrol et
		err = bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte(newPassword))
		assert.NoError(t, err)
		assert.True(t, updatedUser.UpdatedAt.After(testUser.UpdatedAt))
	})

	t.Run("UpdateUserPassword_UserNotFound", func(t *testing.T) {
		err := svc.UpdateUserPassword(ctx, 99999, "newpassword")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("UpdateUserPassword_Hashing", func(t *testing.T) {
		// Önce test kullanıcısını al
		testUser, err := infrastructure.GetTestUserByEmail(ctx, pool, "test1@example.com")
		require.NoError(t, err)

		newPassword := "testpassword456"
		err = svc.UpdateUserPassword(ctx, testUser.ID, newPassword)
		require.NoError(t, err)

		// Şifrenin hash'lendiğini kontrol et
		updatedUser, err := svc.GetUserByID(ctx, testUser.ID)
		require.NoError(t, err)

		assert.NotEqual(t, newPassword, updatedUser.PasswordHash)
		err = bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte(newPassword))
		assert.NoError(t, err)
	})

	t.Run("UpdateUserActiveStatus_Success", func(t *testing.T) {
		// Önce test kullanıcısını al
		testUser, err := infrastructure.GetTestUserByEmail(ctx, pool, "test1@example.com")
		require.NoError(t, err)

		newStatus := false
		err = svc.UpdateUserActiveStatus(ctx, testUser.ID, newStatus)
		require.NoError(t, err)

		// Güncellemenin başarılı olduğunu kontrol et
		updatedUser, err := svc.GetUserByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, newStatus, updatedUser.IsActive)
		assert.True(t, updatedUser.UpdatedAt.After(testUser.UpdatedAt))
	})

	t.Run("UpdateUserActiveStatus_UserNotFound", func(t *testing.T) {
		err := svc.UpdateUserActiveStatus(ctx, 99999, false)
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("DeleteUserByID_Success", func(t *testing.T) {
		// Önce yeni bir kullanıcı ekle
		newUser := &model.User{
			FullName:     "Service User to Delete",
			Email:        "servicedelete@example.com",
			PasswordHash: "plainpassword",
		}

		err := svc.CreateUser(ctx, newUser)
		require.NoError(t, err)

		// Kullanıcıyı sil
		err = svc.DeleteUserByID(ctx, newUser.ID)
		require.NoError(t, err)

		// Kullanıcının gerçekten silindiğini kontrol et
		_, err = svc.GetUserByID(ctx, newUser.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("DeleteUserByID_UserNotFound", func(t *testing.T) {
		err := svc.DeleteUserByID(ctx, 99999)
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("AuthenticateUser_Success", func(t *testing.T) {
		// Önce test kullanıcısını al
		testUser, err := infrastructure.GetTestUserByEmail(ctx, pool, "test1@example.com")
		require.NoError(t, err)

		// Doğru şifre ile giriş yap
		authenticatedUser, err := svc.AuthenticateUser(ctx, testUser.Email, "password123")
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, authenticatedUser.ID)
		assert.Equal(t, testUser.Email, authenticatedUser.Email)
	})

	t.Run("AuthenticateUser_InvalidCredentials", func(t *testing.T) {
		// Yanlış şifre ile giriş yap
		user, err := svc.AuthenticateUser(ctx, "test1@example.com", "wrongpassword")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrInvalidCredentials)
		assert.Empty(t, user)
	})

	t.Run("AuthenticateUser_UserNotFound", func(t *testing.T) {
		// Var olmayan kullanıcı ile giriş yap
		user, err := svc.AuthenticateUser(ctx, "nonexistent@example.com", "password123")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrInvalidCredentials)
		assert.Empty(t, user)
	})

	t.Run("AuthenticateUser_InactiveAccount", func(t *testing.T) {
		// Inactive kullanıcı ile giriş yap
		user, err := svc.AuthenticateUser(ctx, "inactive@example.com", "password789")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrInactiveAccount)
		assert.Empty(t, user)
	})
}

// TestUserServiceIntegration entegrasyon testleri
func TestUserServiceIntegration(t *testing.T) {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://postgres:1234@localhost:5432/bankapp_test?sslmode=disable")
	if err != nil {
		t.Skipf("Test database connection failed: %v", err)
	}
	defer pool.Close()

	err = infrastructure.InitializeTestDatabase(ctx, pool)
	require.NoError(t, err)

	t.Cleanup(func() {
		infrastructure.ClearTestDatabase(ctx, pool)
	})

	repo := repository.NewUserRepository(pool)
	svc := service.NewUserService(repo)

	t.Run("FullUserServiceLifecycle", func(t *testing.T) {
		// 1. Kullanıcı oluştur
		newUser := &model.User{
			FullName:     "Service Lifecycle User",
			Email:        "servicelifecycle@example.com",
			PasswordHash: "initialpassword",
		}

		err := svc.CreateUser(ctx, newUser)
		require.NoError(t, err)
		assert.NotZero(t, newUser.ID)
		assert.Equal(t, "user", newUser.Role)
		assert.True(t, newUser.IsActive)

		// 2. Kullanıcıyı getir
		retrievedUser, err := svc.GetUserByID(ctx, newUser.ID)
		require.NoError(t, err)
		assert.Equal(t, newUser.Email, retrievedUser.Email)

		// 3. Kimlik doğrulama
		authenticatedUser, err := svc.AuthenticateUser(ctx, newUser.Email, "initialpassword")
		require.NoError(t, err)
		assert.Equal(t, newUser.ID, authenticatedUser.ID)

		// 4. Email güncelle
		err = svc.UpdateUserEmail(ctx, newUser.ID, "serviceupdated@example.com")
		require.NoError(t, err)

		// 5. Şifre güncelle
		err = svc.UpdateUserPassword(ctx, newUser.ID, "newservicepassword")
		require.NoError(t, err)

		// 6. Durum güncelle
		err = svc.UpdateUserActiveStatus(ctx, newUser.ID, false)
		require.NoError(t, err)

		// 7. Güncellenmiş kullanıcıyı kontrol et
		updatedUser, err := svc.GetUserByID(ctx, newUser.ID)
		require.NoError(t, err)
		assert.Equal(t, "serviceupdated@example.com", updatedUser.Email)
		assert.False(t, updatedUser.IsActive)

		// 8. Inactive kullanıcı ile giriş yapmaya çalış
		_, err = svc.AuthenticateUser(ctx, updatedUser.Email, "newservicepassword")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrInactiveAccount)

		// 9. Kullanıcıyı tekrar aktif et
		err = svc.UpdateUserActiveStatus(ctx, newUser.ID, true)
		require.NoError(t, err)

		// 10. Tekrar giriş yap
		_, err = svc.AuthenticateUser(ctx, updatedUser.Email, "newservicepassword")
		require.NoError(t, err)

		// 11. Kullanıcıyı sil
		err = svc.DeleteUserByID(ctx, newUser.ID)
		require.NoError(t, err)

		// 12. Kullanıcının silindiğini kontrol et
		_, err = svc.GetUserByID(ctx, newUser.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})
}

// TestUserServiceEdgeCases edge case'ler için testler
func TestUserServiceEdgeCases(t *testing.T) {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://postgres:1234@localhost:5432/bankapp_test?sslmode=disable")
	if err != nil {
		t.Skipf("Test database connection failed: %v", err)
	}
	defer pool.Close()

	err = infrastructure.InitializeTestDatabase(ctx, pool)
	require.NoError(t, err)

	t.Cleanup(func() {
		infrastructure.ClearTestDatabase(ctx, pool)
	})

	repo := repository.NewUserRepository(pool)
	svc := service.NewUserService(repo)

	t.Run("CreateUserWithEmptyRole", func(t *testing.T) {
		newUser := &model.User{
			FullName:     "Empty Role User",
			Email:        "emptyrole@example.com",
			PasswordHash: "password123",
			Role:         "", // Boş role
		}

		err := svc.CreateUser(ctx, newUser)
		require.NoError(t, err)
		assert.Equal(t, "user", newUser.Role) // Service'te "user" olarak set edilmeli
	})

	t.Run("CreateUserWithCustomRole", func(t *testing.T) {
		newUser := &model.User{
			FullName:     "Custom Role User",
			Email:        "customrole@example.com",
			PasswordHash: "password123",
			Role:         "admin", // Özel role
		}

		err := svc.CreateUser(ctx, newUser)
		require.NoError(t, err)
		assert.Equal(t, "admin", newUser.Role) // Özel role korunmalı
	})

	t.Run("CreateUserWithInactiveStatus", func(t *testing.T) {
		newUser := &model.User{
			FullName:     "Inactive Status User",
			Email:        "inactivestatus@example.com",
			PasswordHash: "password123",
			IsActive:     false, // Inactive status
		}

		err := svc.CreateUser(ctx, newUser)
		require.NoError(t, err)
		assert.True(t, newUser.IsActive) // Service'te true olarak set edilmeli
	})

	t.Run("PasswordHashingConsistency", func(t *testing.T) {
		// Aynı şifre ile iki kullanıcı oluştur
		user1 := &model.User{
			FullName:     "User 1",
			Email:        "user1@example.com",
			PasswordHash: "samepassword",
		}

		user2 := &model.User{
			FullName:     "User 2",
			Email:        "user2@example.com",
			PasswordHash: "samepassword",
		}

		err := svc.CreateUser(ctx, user1)
		require.NoError(t, err)

		err = svc.CreateUser(ctx, user2)
		require.NoError(t, err)

		// Hash'lerin farklı olduğunu kontrol et (salt nedeniyle)
		assert.NotEqual(t, user1.PasswordHash, user2.PasswordHash)

		// Her iki şifrenin de doğru hash'lendiğini kontrol et
		err = bcrypt.CompareHashAndPassword([]byte(user1.PasswordHash), []byte("samepassword"))
		assert.NoError(t, err)

		err = bcrypt.CompareHashAndPassword([]byte(user2.PasswordHash), []byte("samepassword"))
		assert.NoError(t, err)
	})
}
