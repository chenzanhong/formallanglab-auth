package user_s

import (
	"context"
	"errors"
	"testing"

	"github.com/chenzanhong/formallanglab-auth/configs"
	"github.com/chenzanhong/formallanglab-auth/internal/domain/dto"
	"github.com/chenzanhong/formallanglab-auth/internal/domain/model"
	userErrors "github.com/chenzanhong/formallanglab-auth/internal/errors"
	"github.com/chenzanhong/formallanglab-auth/internal/middleware"
	"github.com/chenzanhong/formallanglab-auth/pkg/cryptoutil"
	"github.com/chenzanhong/goutil/jwtx"
)

func TestMain(m *testing.M) {
	jwtx.InitWithHS256("testkey", &middleware.AccessTokenClaims{}, jwtx.WithAutoInject(true))
	m.Run()
}

type mockUserRepository struct {
	users         map[string]*model.User
	refreshTokens map[string]string
}

func (m *mockUserRepository) CreateUser(ctx context.Context, user *model.User) error {
	m.users[user.Name] = user
	return nil
}

func (m *mockUserRepository) ExistsByID(ctx context.Context, id int64) (bool, error) {
	return false, nil
}

func (m *mockUserRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	_, exists := m.users[name]
	return exists, nil
}

func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	for _, user := range m.users {
		if user.Email == email {
			return true, nil
		}
	}

	return false, nil
}

func (m *mockUserRepository) GetAllUsers(ctx context.Context) ([]*model.User, error) {
	users := make([]*model.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}

	return users, nil
}

func (m *mockUserRepository) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}

	return nil, userErrors.ErrUserNotFound
}

func (m *mockUserRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, userErrors.ErrUserNotFound
}

func (m *mockUserRepository) GetUserByName(ctx context.Context, name string) (*model.User, error) {
	user, exists := m.users[name]
	if !exists {
		return nil, userErrors.ErrUserNotFound
	}

	return user, nil
}

func (m *mockUserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	m.users[user.Name] = user
	return nil
}

func (m *mockUserRepository) UpdatePasswordByUsername(ctx context.Context, username, password string) error {
	if user, exists := m.users[username]; exists {
		user.Password = password
		return nil
	}

	return userErrors.ErrUserNotFound
}

func (m *mockUserRepository) UpdatePasswordByEmail(ctx context.Context, email, password string) error {
	for _, user := range m.users {
		if user.Email == email {
			user.Password = password
			return nil
		}
	}

	return userErrors.ErrUserNotFound
}

func (m *mockUserRepository) DeleteUser(ctx context.Context, user *model.User) error {
	delete(m.users, user.Name)
	return nil
}

func (m *mockUserRepository) DeleteUserByID(ctx context.Context, id int64) error {
	for name, user := range m.users {
		if user.ID == id {
			delete(m.users, name)
			return nil
		}
	}

	return userErrors.ErrUserNotFound
}

func (m *mockUserRepository) DeleteUserByEmail(ctx context.Context, email string) error {
	for name, user := range m.users {
		if user.Email == email {
			delete(m.users, name)
			return nil
		}
	}

	return userErrors.ErrUserNotFound
}

func (m *mockUserRepository) DeleteUserByName(ctx context.Context, name string) error {
	delete(m.users, name)
	return nil
}

func (m *mockUserRepository) SaveRefreshToken(ctx context.Context, refreshToken string, username string, userID int64) error {
	m.refreshTokens[refreshToken] = username
	return nil
}

func (m *mockUserRepository) ValidateRefreshToken(ctx context.Context, refreshToken string) (bool, error) {
	_, exists := m.refreshTokens[refreshToken]
	return exists, nil
}

func (m *mockUserRepository) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	delete(m.refreshTokens, refreshToken)
	return nil
}

func (m *mockUserRepository) GetUserNameAndIDByRefreshToken(ctx context.Context, refreshToken string) (string, int64, error) {
	username, exists := m.refreshTokens[refreshToken]
	if !exists {
		return "", 0, userErrors.ErrInvalidToken
	}

	return username, 1, nil
}

type mockEmailRepository struct {
	registerTokens map[string]string
	resetTokens    map[string]string
	emailTokens    map[string]string
}

func (m *mockEmailRepository) SaveRegisterVerificationToken(ctx context.Context, email, token string) error {
	if m.registerTokens == nil {
		m.registerTokens = make(map[string]string)
	}
	m.registerTokens[email] = token

	return nil
}

func (m *mockEmailRepository) HasRegisterVerificationToken(ctx context.Context, email string) (bool, error) {
	_, exists := m.registerTokens[email]
	return exists, nil
}

func (m *mockEmailRepository) ValidateRegisterVerificationToken(ctx context.Context, email, token string) (bool, error) {
	stored, exists := m.registerTokens[email]
	return exists && stored == token, nil
}

func (m *mockEmailRepository) DeleteRegisterVerificationToken(ctx context.Context, email string) error {
	delete(m.registerTokens, email)
	return nil
}

func (m *mockEmailRepository) SaveResetPwdToken(ctx context.Context, token, email string) error {
	if m.resetTokens == nil {
		m.resetTokens = make(map[string]string)
	}
	m.resetTokens[token] = email

	return nil
}

func (m *mockEmailRepository) HasResetPwdToken(ctx context.Context, token string) (bool, error) {
	_, exists := m.resetTokens[token]
	return exists, nil
}

func (m *mockEmailRepository) GetEmailByResetPwdToken(ctx context.Context, token string) (string, error) {
	email, exists := m.resetTokens[token]
	if !exists {
		return "", userErrors.ErrInvalidToken
	}

	return email, nil
}

func (m *mockEmailRepository) DeleteResetPwdToken(ctx context.Context, token string) error {
	delete(m.resetTokens, token)
	return nil
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() (*mockUserRepository, *mockEmailRepository)
		input     struct {
			name     string
			email    string
			password string
			token    string
		}
		wantUser *model.User
		wantErr  error
	}{
		{
			name: "valid registration",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users:         make(map[string]*model.User),
					refreshTokens: make(map[string]string),
				}
				emailRepo := &mockEmailRepository{
					registerTokens: map[string]string{"test@example.com": "validtoken"},
				}

				return userRepo, emailRepo
			},
			input: struct {
				name     string
				email    string
				password string
				token    string
			}{
				name:     "testuser",
				email:    "test@example.com",
				password: "password123",
				token:    "validtoken",
			},
			wantUser: &model.User{Name: "testuser", Email: "test@example.com"},
			wantErr:  nil,
		},
		{
			name: "user already exists",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users: map[string]*model.User{
						"existinguser": {ID: 1, Name: "existinguser", Email: "existing@example.com"},
					},
					refreshTokens: make(map[string]string),
				}
				emailRepo := &mockEmailRepository{
					registerTokens: map[string]string{"test@example.com": "validtoken"},
				}

				return userRepo, emailRepo
			},
			input: struct {
				name     string
				email    string
				password string
				token    string
			}{
				name:     "existinguser",
				email:    "test@example.com",
				password: "password123",
				token:    "validtoken",
			},
			wantErr: userErrors.ErrUserAlreadyExists,
		},
		{
			name: "email already exists",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users: map[string]*model.User{
						"someuser": {ID: 1, Name: "someuser", Email: "existing@example.com"},
					},
					refreshTokens: make(map[string]string),
				}
				emailRepo := &mockEmailRepository{
					registerTokens: map[string]string{"existing@example.com": "validtoken"},
				}

				return userRepo, emailRepo
			},
			input: struct {
				name     string
				email    string
				password string
				token    string
			}{
				name:     "newuser",
				email:    "existing@example.com",
				password: "password123",
				token:    "validtoken",
			},
			wantErr: userErrors.ErrEmailAlreadyExists,
		},
		{
			name: "invalid token",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users:         make(map[string]*model.User),
					refreshTokens: make(map[string]string),
				}
				emailRepo := &mockEmailRepository{
					registerTokens: map[string]string{"test@example.com": "validtoken"},
				}

				return userRepo, emailRepo
			},
			input: struct {
				name     string
				email    string
				password string
				token    string
			}{
				name:     "testuser",
				email:    "test@example.com",
				password: "password123",
				token:    "invalidtoken",
			},
			wantErr: userErrors.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo, emailRepo := tt.setupMock()
			jwtCfg := configs.JWTConfig{
				Key:                    "testkey",
				AccessTokenExpireTime:  3600,
				RefreshTokenExpireTime: 86400,
			}
			svc := NewUserService(userRepo, emailRepo, jwtCfg)

			gotUser, gotErr := svc.Register(context.Background(), tt.input.name, tt.input.email, tt.input.password, tt.input.token)

			if (gotErr == nil) != (tt.wantErr == nil) {
				t.Errorf("Register() error = %v, want %v", gotErr, tt.wantErr)
				return
			}

			if gotErr != nil && tt.wantErr != nil && gotErr.Error() != tt.wantErr.Error() {
				t.Errorf("Register() error = %v, want %v", gotErr, tt.wantErr)
			}

			if tt.wantUser != nil && gotUser != nil {
				if gotUser.Name != tt.wantUser.Name {
					t.Errorf("Register() gotUser.Name = %v, want %v", gotUser.Name, tt.wantUser.Name)
				}
				if gotUser.Email != tt.wantUser.Email {
					t.Errorf("Register() gotUser.Email = %v, want %v", gotUser.Email, tt.wantUser.Email)
				}
			}
		})
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() (*mockUserRepository, *mockEmailRepository)
		input     *dto.LoginRequest
		wantErr   error
		checkResp func(*dto.LoginResponse) bool
	}{
		{
			name: "valid login",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				hashedPassword, _ := cryptoutil.HashPassword("password123")
				userRepo := &mockUserRepository{
					users: map[string]*model.User{
						"testuser": {ID: 1, Name: "testuser", Password: hashedPassword},
					},
					refreshTokens: make(map[string]string),
				}
				emailRepo := &mockEmailRepository{}

				return userRepo, emailRepo
			},
			input: &dto.LoginRequest{
				Name:     "testuser",
				Password: "password123",
			},
			wantErr: nil,
			checkResp: func(resp *dto.LoginResponse) bool {
				return resp.Result && resp.AccessToken != "" && resp.RefreshToken != ""
			},
		},
		{
			name: "user not found",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users:         make(map[string]*model.User),
					refreshTokens: make(map[string]string),
				}
				emailRepo := &mockEmailRepository{}

				return userRepo, emailRepo
			},
			input: &dto.LoginRequest{
				Name:     "nonexistent",
				Password: "password123",
			},
			wantErr: userErrors.ErrUserNotFound,
		},
		{
			name: "invalid credentials",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users: map[string]*model.User{
						"testuser": {ID: 1, Name: "testuser", Password: "hashedpassword"},
					},
					refreshTokens: make(map[string]string),
				}
				emailRepo := &mockEmailRepository{}

				return userRepo, emailRepo
			},
			input: &dto.LoginRequest{
				Name:     "testuser",
				Password: "wrongpassword",
			},
			wantErr: userErrors.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo, emailRepo := tt.setupMock()
			jwtCfg := configs.JWTConfig{
				Key:                    "testkey",
				AccessTokenExpireTime:  3600,
				RefreshTokenExpireTime: 86400,
			}
			svc := NewUserService(userRepo, emailRepo, jwtCfg)

			gotResp, gotErr := svc.Login(context.Background(), tt.input)

			if (gotErr == nil) != (tt.wantErr == nil) {
				t.Errorf("Login() error = %v, want %v", gotErr, tt.wantErr)
				return
			}

			if gotErr != nil && tt.wantErr != nil && gotErr.Error() != tt.wantErr.Error() {
				t.Errorf("Login() error = %v, want %v", gotErr, tt.wantErr)
			}

			if tt.checkResp != nil && gotErr == nil {
				if !tt.checkResp(gotResp) {
					t.Error("Login() response check failed")
				}
			}
		})
	}
}

func TestRevokeRefreshToken(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func() (*mockUserRepository, *mockEmailRepository)
		refreshToken string
		wantErr      error
	}{
		{
			name: "revoke valid token",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users:         make(map[string]*model.User),
					refreshTokens: map[string]string{"validtoken": "testuser"},
				}
				emailRepo := &mockEmailRepository{}

				return userRepo, emailRepo
			},
			refreshToken: "validtoken",
			wantErr:      nil,
		},
		{
			name: "revoke invalid token",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users:         make(map[string]*model.User),
					refreshTokens: make(map[string]string),
				}
				emailRepo := &mockEmailRepository{}

				return userRepo, emailRepo
			},
			refreshToken: "invalidtoken",
			wantErr:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo, emailRepo := tt.setupMock()
			jwtCfg := configs.JWTConfig{
				Key:                    "testkey",
				AccessTokenExpireTime:  3600,
				RefreshTokenExpireTime: 86400,
			}
			svc := NewUserService(userRepo, emailRepo, jwtCfg)

			gotErr := svc.RevokeRefreshToken(context.Background(), tt.refreshToken)

			if (gotErr == nil) != (tt.wantErr == nil) {
				t.Errorf("RevokeRefreshToken() error = %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestGetUserNameAndIDByRefreshToken(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func() (*mockUserRepository, *mockEmailRepository)
		refreshToken string
		wantName     string
		wantID       int64
		wantErr      error
	}{
		{
			name: "valid token",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users:         make(map[string]*model.User),
					refreshTokens: map[string]string{"validtoken": "testuser"},
				}
				emailRepo := &mockEmailRepository{}

				return userRepo, emailRepo
			},
			refreshToken: "validtoken",
			wantName:     "testuser",
			wantID:       1,
			wantErr:      nil,
		},
		{
			name: "invalid token",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users:         make(map[string]*model.User),
					refreshTokens: make(map[string]string),
				}
				emailRepo := &mockEmailRepository{}

				return userRepo, emailRepo
			},
			refreshToken: "invalidtoken",
			wantName:     "",
			wantID:       0,
			wantErr:      userErrors.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo, emailRepo := tt.setupMock()
			jwtCfg := configs.JWTConfig{
				Key:                    "testkey",
				AccessTokenExpireTime:  3600,
				RefreshTokenExpireTime: 86400,
			}
			svc := NewUserService(userRepo, emailRepo, jwtCfg)

			gotName, gotID, gotErr := svc.GetUserNameAndIDByRefreshToken(context.Background(), tt.refreshToken)

			if (gotErr == nil) != (tt.wantErr == nil) {
				t.Errorf("GetUserNameAndIDByRefreshToken() error = %v, want %v", gotErr, tt.wantErr)
				return
			}

			if gotErr != nil && tt.wantErr != nil && gotErr.Error() != tt.wantErr.Error() {
				t.Errorf("GetUserNameAndIDByRefreshToken() error = %v, want %v", gotErr, tt.wantErr)
			}

			if gotName != tt.wantName {
				t.Errorf("GetUserNameAndIDByRefreshToken() gotName = %v, want %v", gotName, tt.wantName)
			}

			if gotID != tt.wantID {
				t.Errorf("GetUserNameAndIDByRefreshToken() gotID = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestResetPassword(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func() (*mockUserRepository, *mockEmailRepository)
		token       string
		newPassword string
		wantErr     error
	}{
		{
			name: "valid password reset",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users: map[string]*model.User{
						"testuser": {ID: 1, Name: "testuser", Email: "test@example.com", Password: "oldpassword"},
					},
					refreshTokens: make(map[string]string),
				}
				emailRepo := &mockEmailRepository{
					resetTokens: map[string]string{"validtoken": "test@example.com"},
				}

				return userRepo, emailRepo
			},
			token:       "validtoken",
			newPassword: "newpassword123",
			wantErr:     nil,
		},
		{
			name: "invalid token",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users:         make(map[string]*model.User),
					refreshTokens: make(map[string]string),
				}
				emailRepo := &mockEmailRepository{}

				return userRepo, emailRepo
			},
			token:       "invalidtoken",
			newPassword: "newpassword123",
			wantErr:     errors.New("无效或过期的重置链接"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo, emailRepo := tt.setupMock()
			jwtCfg := configs.JWTConfig{
				Key:                    "testkey",
				AccessTokenExpireTime:  3600,
				RefreshTokenExpireTime: 86400,
			}
			svc := NewUserService(userRepo, emailRepo, jwtCfg)

			gotErr := svc.ResetPassword(context.Background(), tt.token, tt.newPassword)

			if (gotErr == nil) != (tt.wantErr == nil) {
				t.Errorf("ResetPassword() error = %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() (*mockUserRepository, *mockEmailRepository)
		userID    int64
		wantUser  *model.User
		wantErr   error
	}{
		{
			name: "valid user",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users: map[string]*model.User{
						"testuser": {ID: 1, Name: "testuser", Email: "test@example.com"},
					},
					refreshTokens: make(map[string]string),
				}
				emailRepo := &mockEmailRepository{}

				return userRepo, emailRepo
			},
			userID:   1,
			wantUser: &model.User{ID: 1, Name: "testuser", Email: "test@example.com"},
			wantErr:  nil,
		},
		{
			name: "user not found",
			setupMock: func() (*mockUserRepository, *mockEmailRepository) {
				userRepo := &mockUserRepository{
					users:         make(map[string]*model.User),
					refreshTokens: make(map[string]string),
				}
				emailRepo := &mockEmailRepository{}

				return userRepo, emailRepo
			},
			userID:   999,
			wantUser: nil,
			wantErr:  userErrors.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo, emailRepo := tt.setupMock()
			jwtCfg := configs.JWTConfig{
				Key:                    "testkey",
				AccessTokenExpireTime:  3600,
				RefreshTokenExpireTime: 86400,
			}
			svc := NewUserService(userRepo, emailRepo, jwtCfg)

			gotUser, gotErr := svc.GetUser(context.Background(), tt.userID)

			if (gotErr == nil) != (tt.wantErr == nil) {
				t.Errorf("GetUser() error = %v, want %v", gotErr, tt.wantErr)
				return
			}

			if gotErr != nil && tt.wantErr != nil && gotErr.Error() != tt.wantErr.Error() {
				t.Errorf("GetUser() error = %v, want %v", gotErr, tt.wantErr)
			}

			if tt.wantUser != nil && gotUser != nil {
				if gotUser.ID != tt.wantUser.ID {
					t.Errorf("GetUser() gotUser.ID = %v, want %v", gotUser.ID, tt.wantUser.ID)
				}
				if gotUser.Name != tt.wantUser.Name {
					t.Errorf("GetUser() gotUser.Name = %v, want %v", gotUser.Name, tt.wantUser.Name)
				}
				if gotUser.Email != tt.wantUser.Email {
					t.Errorf("GetUser() gotUser.Email = %v, want %v", gotUser.Email, tt.wantUser.Email)
				}
			}
		})
	}
}
