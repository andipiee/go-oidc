package handler

import (
	"net/http"

	"github.com/andipiee/go-oidc/internal/application/usecase"
	"github.com/andipiee/go-oidc/internal/presentation/httputil"
)

type TokenHandler struct {
	tokenUseCase *usecase.TokenUseCase
}

func NewTokenHandler(tokenUseCase *usecase.TokenUseCase) *TokenHandler {
	return &TokenHandler{tokenUseCase: tokenUseCase}
}

func (h *TokenHandler) HandleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	req := usecase.TokenRequest{
		GrantType:    r.FormValue("grant_type"),
		Code:         r.FormValue("code"),
		RedirectURI:  r.FormValue("redirect_uri"),
		ClientID:     r.FormValue("client_id"),
		ClientSecret: r.FormValue("client_secret"),
		CodeVerifier: r.FormValue("code_verifier"),
		RefreshToken: r.FormValue("refresh_token"),
		Scope:        r.FormValue("scope"),
	}

	// Client credentials from Basic auth take precedence
	if u, p, ok := r.BasicAuth(); ok {
		req.ClientID = u
		req.ClientSecret = p
	}

	var resp *usecase.TokenResponse
	var err error

	switch req.GrantType {
	case "authorization_code":
		resp, err = h.tokenUseCase.ExchangeCode(r.Context(), req)
	case "refresh_token":
		resp, err = h.tokenUseCase.RefreshToken(r.Context(), req)
	default:
		httputil.Error(w, http.StatusBadRequest, "unsupported_grant_type", "")
		return
	}

	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid_grant", err.Error())
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}

func (h *TokenHandler) HandleRevoke(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	token := r.FormValue("token")
	tokenTypeHint := r.FormValue("token_type_hint")

	if err := h.tokenUseCase.RevokeToken(r.Context(), token, tokenTypeHint); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid_request", "")
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]any{})
}

func (h *TokenHandler) HandleIntrospect(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	token := r.FormValue("token")

	if token == "" {
		httputil.JSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}

	accessToken, err := h.tokenUseCase.IntrospectToken(r.Context(), token)
	if err != nil || accessToken == nil {
		httputil.JSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"active":    true,
		"client_id": accessToken.ClientID,
		"scope":     accessToken.Scope,
		"exp":       accessToken.ExpiresAt.Unix(),
	})
}
