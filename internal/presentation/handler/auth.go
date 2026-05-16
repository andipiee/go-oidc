package handler

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andipiee/go-oidc/internal/application/usecase"
	"github.com/andipiee/go-oidc/internal/domain/entity"
	"github.com/andipiee/go-oidc/internal/domain/repository"
	"github.com/andipiee/go-oidc/internal/infrastructure/auth"
	"github.com/andipiee/go-oidc/internal/presentation/httputil"
	"github.com/google/uuid"
)

type AuthHandler struct {
	userRepo      repository.UserRepository
	cryptoService *auth.CryptoService
	jwtService    *auth.JWTService
	userUseCase   *usecase.UserUseCase
	tmpl          *template.Template
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token,omitempty"`
}

type UserResponse struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	EmailVerified bool      `json:"email_verified"`
	Picture       string    `json:"picture,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewAuthHandler(userRepo repository.UserRepository, cryptoService *auth.CryptoService, jwtService *auth.JWTService, userUseCase *usecase.UserUseCase, tmpl *template.Template) *AuthHandler {
	return &AuthHandler{
		userRepo:      userRepo,
		cryptoService: cryptoService,
		jwtService:    jwtService,
		userUseCase:   userUseCase,
		tmpl:          tmpl,
	}
}

func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	if req.Email == "" || req.Name == "" || req.Password == "" {
		httputil.Error(w, http.StatusBadRequest, "email, name, and password are required", "")
		return
	}

	if len(req.Password) < 6 {
		httputil.Error(w, http.StatusBadRequest, "password must be at least 6 characters", "")
		return
	}

	ctx := context.Background()

	existingUser, _ := h.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		httputil.Error(w, http.StatusConflict, "user already exists", "")
		return
	}

	hash, err := h.cryptoService.HashPassword(req.Password)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "failed to hash password", "")
		return
	}

	user := &entity.User{
		ID:            uuid.Must(uuid.NewV7()),
		Email:         req.Email,
		Name:          req.Name,
		PasswordHash:  hash,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err = h.userRepo.Create(ctx, user)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "failed to create user: "+err.Error(), "")
		return
	}

	response := UserResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		Name:          user.Name,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt,
	}

	httputil.JSON(w, http.StatusCreated, map[string]any{
		"message": "user created successfully",
		"user":    response,
	})
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	if req.Email == "" || req.Password == "" {
		httputil.Error(w, http.StatusBadRequest, "email and password are required", "")
		return
	}

	ctx := context.Background()

	existingUser, err := h.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || existingUser == nil {
		httputil.Error(w, http.StatusUnauthorized, "invalid credentials", "")
		return
	}

	user := existingUser

	if user.PasswordHash == "" {
		httputil.Error(w, http.StatusUnauthorized, "invalid credentials", "")
		return
	}

	if !h.cryptoService.CheckPassword(req.Password, user.PasswordHash) {
		httputil.Error(w, http.StatusUnauthorized, "invalid credentials", "")
		return
	}

	scope := "openid profile email"
	accessToken, err := h.jwtService.GenerateAccessToken(
		user.ID.String(),
		"",
		scope,
		user.Email,
		user.Name,
		user.Picture,
	)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "failed to generate access token", "")
		return
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken(user.ID.String(), "", scope)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "failed to generate refresh token", "")
		return
	}

	secure := r.TLS != nil

	http.SetCookie(w, &http.Cookie{Name: "access_token", Value: accessToken, MaxAge: 900, Path: "/", Secure: secure, HttpOnly: true})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: refreshToken, MaxAge: 86400, Path: "/", Secure: secure, HttpOnly: true})

	response := AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
	}

	httputil.JSON(w, http.StatusOK, response)
}

func (h *AuthHandler) ShowLoginPage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	httputil.HTML(w, h.tmpl, "login.html", http.StatusOK, map[string]any{
		"error":              "",
		"email":              "",
		"clientID":           q.Get("client_id"),
		"redirectURI":        q.Get("redirect_uri"),
		"state":              q.Get("state"),
		"scope":              q.Get("scope"),
		"nonce":              q.Get("nonce"),
		"codeChallenge":      q.Get("code_challenge"),
		"codeChallengeMethod": q.Get("code_challenge_method"),
	})
}

func (h *AuthHandler) HandleLoginForm(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	email := r.FormValue("email")
	password := r.FormValue("password")
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	state := r.FormValue("state")
	scope := r.FormValue("scope")
	nonce := r.FormValue("nonce")
	codeChallenge := r.FormValue("code_challenge")
	codeChallengeMethod := r.FormValue("code_challenge_method")

	ctx := context.Background()

	renderError := func(errMsg string) {
		httputil.HTML(w, h.tmpl, "login.html", http.StatusOK, map[string]any{
			"error":              errMsg,
			"email":              email,
			"clientID":           clientID,
			"redirectURI":        redirectURI,
			"state":              state,
			"scope":              scope,
			"nonce":              nonce,
			"codeChallenge":      codeChallenge,
			"codeChallengeMethod": codeChallengeMethod,
		})
	}

	existingUser, err := h.userRepo.GetByEmail(ctx, email)
	if err != nil || existingUser == nil {
		renderError("Invalid credentials")
		return
	}

	user := existingUser

	if user.PasswordHash == "" || !h.cryptoService.CheckPassword(password, user.PasswordHash) {
		renderError("Invalid credentials")
		return
	}

	// Create a session for the user
	session, err := h.userUseCase.CreateSession(ctx, user.ID)
	if err != nil {
		renderError("Failed to create session")
		return
	}

	secure := r.TLS != nil
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: session.SessionID, MaxAge: 86400, Path: "/", Secure: secure, HttpOnly: true})

	if redirectURI == "" && clientID == "" {
		httputil.JSON(w, http.StatusOK, map[string]any{
			"message": "Login successful",
			"user": UserResponse{
				ID:            user.ID.String(),
				Email:         user.Email,
				Name:          user.Name,
				EmailVerified: user.EmailVerified,
				CreatedAt:     user.CreatedAt,
			},
		})
		return
	}

	// Redirect back to authorize endpoint with all original params
	params := url.Values{
		"client_id":    {clientID},
		"redirect_uri": {redirectURI},
		"response_type": {"code"},
		"state":        {state},
		"scope":        {scope},
	}
	if nonce != "" {
		params.Set("nonce", nonce)
	}
	if codeChallenge != "" {
		params.Set("code_challenge", codeChallenge)
	}
	if codeChallengeMethod != "" {
		params.Set("code_challenge_method", codeChallengeMethod)
	}
	http.Redirect(w, r, "/oauth2/authorize?"+params.Encode(), http.StatusFound)
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("session_id"); err == nil {
		h.userUseCase.DeleteSession(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "", MaxAge: -1, Path: "/", HttpOnly: true})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "", MaxAge: -1, Path: "/", HttpOnly: true})
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", MaxAge: -1, Path: "/", HttpOnly: true})
	httputil.JSON(w, http.StatusOK, map[string]any{"message": "logged out"})
}

func (h *AuthHandler) ShowRegisterPage(w http.ResponseWriter, r *http.Request) {
	httputil.HTML(w, h.tmpl, "register.html", http.StatusOK, map[string]any{
		"error":   "",
		"success": "",
		"name":    "",
		"email":   "",
	})
}

func (h *AuthHandler) HandleRegisterForm(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if name == "" || email == "" || password == "" {
		httputil.HTML(w, h.tmpl, "register.html", http.StatusOK, map[string]any{
			"error": "All fields are required",
			"name":  name,
			"email": email,
		})
		return
	}

	ctx := context.Background()

	existingUser, _ := h.userRepo.GetByEmail(ctx, email)
	if existingUser != nil {
		httputil.HTML(w, h.tmpl, "register.html", http.StatusOK, map[string]any{
			"error": "User already exists",
			"name":  name,
			"email": email,
		})
		return
	}

	hash, err := h.cryptoService.HashPassword(password)
	if err != nil {
		httputil.HTML(w, h.tmpl, "register.html", http.StatusOK, map[string]any{
			"error": "Failed to process registration",
			"name":  name,
			"email": email,
		})
		return
	}

	user := &entity.User{
		ID:            uuid.Must(uuid.NewV7()),
		Email:         email,
		Name:          name,
		PasswordHash:  hash,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err = h.userRepo.Create(ctx, user)
	if err != nil {
		httputil.HTML(w, h.tmpl, "register.html", http.StatusOK, map[string]any{
			"error": "Failed to create user: " + err.Error(),
			"name":  name,
			"email": email,
		})
		return
	}

	httputil.HTML(w, h.tmpl, "register.html", http.StatusOK, map[string]any{
		"success": "Registration successful! Please login.",
		"name":    "",
		"email":   "",
	})
}

// isValidEmail does basic email validation.
func isValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}
