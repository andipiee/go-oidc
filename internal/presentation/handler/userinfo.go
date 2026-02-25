package handler

import (
	"net/http"

	"github.com/andipiee/go-oidc/internal/application/usecase"
	"github.com/andipiee/go-oidc/internal/infrastructure/auth"
	"github.com/gin-gonic/gin"
)

type UserInfoHandler struct {
	userUseCase *usecase.UserUseCase
	jwtService  *auth.JWTService
}

func NewUserInfoHandler(userUseCase *usecase.UserUseCase, jwtService *auth.JWTService) *UserInfoHandler {
	return &UserInfoHandler{
		userUseCase: userUseCase,
		jwtService:  jwtService,
	}
}

func (h *UserInfoHandler) HandleUserInfo(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	token := authHeader[len("Bearer "):]
	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	user, err := h.userUseCase.GetUserByID(c.Request.Context(), claims.Subject)
	if err != nil || user == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user_not_found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sub":            user.ID.String(),
		"name":           user.Name,
		"email":          user.Email,
		"email_verified": user.EmailVerified,
		"picture":        user.Picture,
	})
}
