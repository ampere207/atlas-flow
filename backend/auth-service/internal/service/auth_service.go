package service

import (
	"errors"
	"time"

	"atlasflow/backend/auth-service/internal/repository"
	"atlasflow/backend/shared/auth"
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo   *repository.UserRepository
	jwtManager *auth.JWTManager
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo *repository.UserRepository, jwtManager *auth.JWTManager) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

// SignupRequest represents a signup request
type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"full_name" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

// UserResponse represents user information
type UserResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

// Signup creates a new user
func (as *AuthService) Signup(req SignupRequest) (*AuthResponse, error) {
	// Check if user already exists
	_, err := as.userRepo.GetByEmail(req.Email)
	if err == nil {
		return nil, errors.New("user already exists")
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create user
	user, err := as.userRepo.Create(req.Email, req.FullName, passwordHash)
	if err != nil {
		return nil, err
	}

	// Generate tokens
	accessToken, err := as.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := as.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	// Save refresh token
	expiresAt := time.Now().AddDate(0, 0, 30)
	err = as.userRepo.SaveRefreshToken(user.ID, refreshToken, expiresAt)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserResponse{
			ID:       user.ID,
			Email:    user.Email,
			FullName: user.FullName,
		},
	}, nil
}

// Login authenticates a user
func (as *AuthService) Login(req LoginRequest) (*AuthResponse, error) {
	// Get user by email
	user, err := as.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Check password
	if !auth.CheckPassword(req.Password, user.Password) {
		return nil, errors.New("invalid credentials")
	}

	// Generate tokens
	accessToken, err := as.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := as.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	// Save refresh token
	expiresAt := time.Now().AddDate(0, 0, 30)
	err = as.userRepo.SaveRefreshToken(user.ID, refreshToken, expiresAt)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserResponse{
			ID:       user.ID,
			Email:    user.Email,
			FullName: user.FullName,
		},
	}, nil
}

// RefreshAccessToken generates a new access token
func (as *AuthService) RefreshAccessToken(refreshToken string) (*AuthResponse, error) {
	// Validate refresh token
	userID, err := as.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Get user
	user, err := as.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Generate new access token
	accessToken, err := as.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserResponse{
			ID:       user.ID,
			Email:    user.Email,
			FullName: user.FullName,
		},
	}, nil
}
