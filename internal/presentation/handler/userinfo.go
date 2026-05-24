package handler

import (
	"net/http"
	"strings"

	"github.com/andipiee/go-oidc/internal/application/usecase"
	"github.com/andipiee/go-oidc/internal/infrastructure/auth"
	"github.com/andipiee/go-oidc/internal/presentation/httputil"
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

func (h *UserInfoHandler) HandleUserInfo(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		httputil.Error(w, http.StatusUnauthorized, "invalid_token", "")
		return
	}

	token := authHeader[len("Bearer "):]
	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		httputil.Error(w, http.StatusUnauthorized, "invalid_token", "")
		return
	}

	user, err := h.userUseCase.GetUserByID(r.Context(), claims.Subject)
	if err != nil || user == nil {
		httputil.Error(w, http.StatusInternalServerError, "user_not_found", "")
		return
	}

	resp := map[string]any{
		"sub":  user.ID.String(),
		"role": user.Role,
	}

	scopes := strings.Fields(claims.Scope)
	scopeSet := make(map[string]bool, len(scopes))
	for _, s := range scopes {
		scopeSet[s] = true
	}

	if scopeSet["profile"] {
		resp["name"] = user.Name
		resp["picture"] = user.Picture
	}

	if scopeSet["email"] {
		resp["email"] = user.Email
		resp["email_verified"] = user.EmailVerified
	}

	httputil.JSON(w, http.StatusOK, resp)
}
