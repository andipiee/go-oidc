package handler

import (
	"net/http"

	"github.com/andipiee/go-oidc/internal/application/usecase"
	"github.com/gin-gonic/gin"
)

type TokenHandler struct {
	tokenUseCase *usecase.TokenUseCase
}

func NewTokenHandler(tokenUseCase *usecase.TokenUseCase) *TokenHandler {
	return &TokenHandler{tokenUseCase: tokenUseCase}
}

func (h *TokenHandler) HandleToken(c *gin.Context) {
	var req usecase.TokenRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": err.Error()})
		return
	}

	var resp *usecase.TokenResponse
	var err error

	switch req.GrantType {
	case "authorization_code":
		resp, err = h.tokenUseCase.ExchangeCode(c.Request.Context(), req)
	case "refresh_token":
		resp, err = h.tokenUseCase.RefreshToken(c.Request.Context(), req)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *TokenHandler) HandleRevoke(c *gin.Context) {
	token := c.PostForm("token")
	tokenTypeHint := c.PostForm("token_type_hint")

	if err := h.tokenUseCase.RevokeToken(c.Request.Context(), token, tokenTypeHint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (h *TokenHandler) HandleIntrospect(c *gin.Context) {
	token := c.PostForm("token")
	_ = c.PostForm("token_type_hint")

	if token == "" {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}

	accessToken, err := h.tokenUseCase.IntrospectToken(c.Request.Context(), token)
	if err != nil || accessToken == nil {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"active":    true,
		"client_id": accessToken.ClientID,
		"scope":     accessToken.Scope,
		"exp":       accessToken.ExpiresAt.Unix(),
	})
}
