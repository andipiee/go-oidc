package handler

import (
	"net/http"

	"github.com/andipiee/go-oidc/internal/application/usecase"
	"github.com/gin-gonic/gin"
)

type AuthorizeHandler struct {
	authorizeUseCase *usecase.AuthorizeUseCase
}

func NewAuthorizeHandler(authorizeUseCase *usecase.AuthorizeUseCase) *AuthorizeHandler {
	return &AuthorizeHandler{authorizeUseCase: authorizeUseCase}
}

func (h *AuthorizeHandler) HandleAuthorize(c *gin.Context) {
	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	responseType := c.Query("response_type")
	scope := c.DefaultQuery("scope", "openid")
	state := c.Query("state")
	nonce := c.Query("nonce")
	codeChallenge := c.Query("code_challenge")
	codeChallengeMethod := c.DefaultQuery("code_challenge_method", "plain")

	if clientID == "" || redirectURI == "" || responseType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "missing required parameters"})
		return
	}

	if responseType != "code" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_response_type"})
		return
	}

	result, err := h.authorizeUseCase.Authorize(c.Request.Context(), usecase.AuthorizeParams{
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		ResponseType:        responseType,
		Scope:               scope,
		State:               state,
		Nonce:               nonce,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": err.Error()})
		return
	}

	c.Redirect(http.StatusFound, result.RedirectURI+"?code="+result.Code+"&state="+result.State)
}

func (h *AuthorizeHandler) HandleAuthorizePost(c *gin.Context) {
	h.HandleAuthorize(c)
}
