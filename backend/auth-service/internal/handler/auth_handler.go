package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"atlasflow/backend/auth-service/internal/service"
	"atlasflow/backend/shared/middleware"
)

// AuthHandler handles authentication routes
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// SignUp handles user signup
func (ah *AuthHandler) SignUp(c *gin.Context) {
	var req service.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c.Writer, http.StatusBadRequest, "invalid request")
		return
	}

	result, err := ah.authService.Signup(req)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusBadRequest, err.Error())
		return
	}

	middleware.RespondJSON(c.Writer, http.StatusCreated, result)
}

// Login handles user login
func (ah *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c.Writer, http.StatusBadRequest, "invalid request")
		return
	}

	result, err := ah.authService.Login(req)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, err.Error())
		return
	}

	middleware.RespondJSON(c.Writer, http.StatusOK, result)
}

// RefreshToken handles token refresh
func (ah *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c.Writer, http.StatusBadRequest, "invalid request")
		return
	}

	result, err := ah.authService.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		middleware.RespondError(c.Writer, http.StatusUnauthorized, err.Error())
		return
	}

	middleware.RespondJSON(c.Writer, http.StatusOK, result)
}
