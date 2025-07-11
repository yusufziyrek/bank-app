package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yusufziyrek/bank-app/internal/model"
	"github.com/yusufziyrek/bank-app/internal/service"
	"golang.org/x/crypto/bcrypt"
)

// TestUserServiceWithMock UserService için mock repository ile testler
func TestUserServiceWithMock(t *testing.T) {
	ctx := context.Background()

	t.Run("GetAllUsers_Empty", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		users, err := svc.GetAllUsers(ctx)
		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("GetAllUsers_WithData", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		// Test kullanıcıları ekle
		testUsers := []*model.User{
			{
				FullName:     "Test User 1",
				Email:        "test1@example.com",
				PasswordHash: "hashedpassword1",
				Role:         "user",
				IsActive:     true,
			},
			{
				FullName:     "Test User 2",
				Email:        "test2@example.com",
				PasswordHash: "hashedpassword2",
				Role:         "admin",
				IsActive:     true,
			},
		}

		for _, user := range testUsers {
			mockRepo.AddTestUser(user)
		}

		users, err := svc.GetAllUsers(ctx)
		require.NoError(t, err)
		assert.Len(t, users, 2)
	})

	t.Run("GetUserByID_Success", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		testUser := &model.User{
			FullName:     "Test User",
			Email:        "test@example.com",
			PasswordHash: "hashedpassword",
			Role:         "user",
			IsActive:     true,
		}
		mockRepo.AddTestUser(testUser)

		user, err := svc.GetUserByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, testUser.Email, user.Email)
		assert.Equal(t, testUser.FullName, user.FullName)
	})

	t.Run("GetUserByID_NotFound", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		user, err := svc.GetUserByID(ctx, 999)
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
		assert.Empty(t, user)
	})

	t.Run("CreateUser_Success", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		newUser := &model.User{
			FullName:     "New User",
			Email:        "newuser@example.com",
			PasswordHash: "plainpassword",
			Role:         "",
			IsActive:     false,
		}

		err := svc.CreateUser(ctx, newUser)
		require.NoError(t, err)
		assert.NotZero(t, newUser.ID)
		assert.Equal(t, "user", newUser.Role)
		assert.True(t, newUser.IsActive)

		// Şifrenin hash'lendiğini kontrol et
		err = bcrypt.CompareHashAndPassword([]byte(newUser.PasswordHash), []byte("plainpassword"))
		assert.NoError(t, err)
	})

	t.Run("CreateUser_DuplicateEmail", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		// İlk kullanıcıyı ekle
		existingUser := &model.User{
			FullName:     "Existing User",
			Email:        "existing@example.com",
			PasswordHash: "hashedpassword",
		}
		mockRepo.AddTestUser(existingUser)

		// Aynı email ile ikinci kullanıcı eklemeye çalış
		duplicateUser := &model.User{
			FullName:     "Duplicate User",
			Email:        "existing@example.com",
			PasswordHash: "anotherpassword",
		}

		err := svc.CreateUser(ctx, duplicateUser)
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrEmailAlreadyRegistered)
	})

	t.Run("UpdateUserEmail_Success", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		testUser := &model.User{
			FullName:     "Test User",
			Email:        "oldemail@example.com",
			PasswordHash: "hashedpassword",
		}
		mockRepo.AddTestUser(testUser)

		newEmail := "newemail@example.com"
		err := svc.UpdateUserEmail(ctx, testUser.ID, newEmail)
		require.NoError(t, err)

		// Güncellemenin başarılı olduğunu kontrol et
		updatedUser, err := svc.GetUserByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, newEmail, updatedUser.Email)
	})

	t.Run("UpdateUserEmail_UserNotFound", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		err := svc.UpdateUserEmail(ctx, 999, "newemail@example.com")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("UpdateUserEmail_DuplicateEmail", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		// İki kullanıcı ekle
		user1 := &model.User{
			FullName:     "User 1",
			Email:        "user1@example.com",
			PasswordHash: "hashedpassword1",
		}
		user2 := &model.User{
			FullName:     "User 2",
			Email:        "user2@example.com",
			PasswordHash: "hashedpassword2",
		}
		mockRepo.AddTestUser(user1)
		mockRepo.AddTestUser(user2)

		// User1'in email'ini User2'nin email'ine güncellemeye çalış
		err := svc.UpdateUserEmail(ctx, user1.ID, user2.Email)
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrEmailAlreadyRegistered)
	})

	t.Run("UpdateUserPassword_Success", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		testUser := &model.User{
			FullName:     "Test User",
			Email:        "test@example.com",
			PasswordHash: "oldhashedpassword",
		}
		mockRepo.AddTestUser(testUser)

		newPassword := "newpassword123"
		err := svc.UpdateUserPassword(ctx, testUser.ID, newPassword)
		require.NoError(t, err)

		// Şifrenin hash'lendiğini kontrol et
		updatedUser, err := svc.GetUserByID(ctx, testUser.ID)
		require.NoError(t, err)

		err = bcrypt.CompareHashAndPassword([]byte(updatedUser.PasswordHash), []byte(newPassword))
		assert.NoError(t, err)
	})

	t.Run("UpdateUserPassword_UserNotFound", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		err := svc.UpdateUserPassword(ctx, 999, "newpassword")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("UpdateUserActiveStatus_Success", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		testUser := &model.User{
			FullName:     "Test User",
			Email:        "test@example.com",
			PasswordHash: "hashedpassword",
			IsActive:     true,
		}
		mockRepo.AddTestUser(testUser)

		err := svc.UpdateUserActiveStatus(ctx, testUser.ID, false)
		require.NoError(t, err)

		// Durumun güncellendiğini kontrol et
		updatedUser, err := svc.GetUserByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.False(t, updatedUser.IsActive)
	})

	t.Run("UpdateUserActiveStatus_UserNotFound", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		err := svc.UpdateUserActiveStatus(ctx, 999, false)
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("DeleteUserByID_Success", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		testUser := &model.User{
			FullName:     "Test User",
			Email:        "test@example.com",
			PasswordHash: "hashedpassword",
		}
		mockRepo.AddTestUser(testUser)

		err := svc.DeleteUserByID(ctx, testUser.ID)
		require.NoError(t, err)

		// Kullanıcının silindiğini kontrol et
		_, err = svc.GetUserByID(ctx, testUser.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("DeleteUserByID_UserNotFound", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		err := svc.DeleteUserByID(ctx, 999)
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("AuthenticateUser_Success", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		// Hash'lenmiş şifre ile kullanıcı oluştur
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		testUser := &model.User{
			FullName:     "Test User",
			Email:        "test@example.com",
			PasswordHash: string(hashedPassword),
			IsActive:     true,
		}
		mockRepo.AddTestUser(testUser)

		// Doğru şifre ile giriş yap
		authenticatedUser, err := svc.AuthenticateUser(ctx, testUser.Email, "password123")
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, authenticatedUser.ID)
		assert.Equal(t, testUser.Email, authenticatedUser.Email)
	})

	t.Run("AuthenticateUser_InvalidCredentials", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		testUser := &model.User{
			FullName:     "Test User",
			Email:        "test@example.com",
			PasswordHash: string(hashedPassword),
			IsActive:     true,
		}
		mockRepo.AddTestUser(testUser)

		// Yanlış şifre ile giriş yap
		user, err := svc.AuthenticateUser(ctx, testUser.Email, "wrongpassword")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrInvalidCredentials)
		assert.Empty(t, user)
	})

	t.Run("AuthenticateUser_UserNotFound", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		user, err := svc.AuthenticateUser(ctx, "nonexistent@example.com", "password123")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrInvalidCredentials)
		assert.Empty(t, user)
	})

	t.Run("AuthenticateUser_InactiveAccount", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		testUser := &model.User{
			FullName:     "Test User",
			Email:        "test@example.com",
			PasswordHash: string(hashedPassword),
			IsActive:     false, // Inactive
		}
		mockRepo.AddTestUser(testUser)

		user, err := svc.AuthenticateUser(ctx, testUser.Email, "password123")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrInactiveAccount)
		assert.Empty(t, user)
	})
}

// TestUserServiceMockIntegration mock ile entegrasyon testleri
func TestUserServiceMockIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("FullUserLifecycleWithMock", func(t *testing.T) {
		mockRepo := NewMockUserRepository()
		svc := service.NewUserService(mockRepo)

		// 1. Kullanıcı oluştur
		newUser := &model.User{
			FullName:     "Lifecycle User",
			Email:        "lifecycle@example.com",
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
		err = svc.UpdateUserEmail(ctx, newUser.ID, "updated@example.com")
		require.NoError(t, err)

		// 5. Şifre güncelle
		err = svc.UpdateUserPassword(ctx, newUser.ID, "newpassword")
		require.NoError(t, err)

		// 6. Durum güncelle
		err = svc.UpdateUserActiveStatus(ctx, newUser.ID, false)
		require.NoError(t, err)

		// 7. Güncellenmiş kullanıcıyı kontrol et
		updatedUser, err := svc.GetUserByID(ctx, newUser.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated@example.com", updatedUser.Email)
		assert.False(t, updatedUser.IsActive)

		// 8. Inactive kullanıcı ile giriş yapmaya çalış
		_, err = svc.AuthenticateUser(ctx, updatedUser.Email, "newpassword")
		assert.Error(t, err)
		assert.ErrorIs(t, err, service.ErrInactiveAccount)

		// 9. Kullanıcıyı tekrar aktif et
		err = svc.UpdateUserActiveStatus(ctx, newUser.ID, true)
		require.NoError(t, err)

		// 10. Tekrar giriş yap
		_, err = svc.AuthenticateUser(ctx, updatedUser.Email, "newpassword")
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
