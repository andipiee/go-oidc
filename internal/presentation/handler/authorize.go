package handler

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/andipiee/go-oidc/internal/application/usecase"
	"github.com/andipiee/go-oidc/internal/presentation/httputil"
	"github.com/google/uuid"
)

type AuthorizeHandler struct {
	authorizeUseCase *usecase.AuthorizeUseCase
	userUseCase      *usecase.UserUseCase
}

func NewAuthorizeHandler(authorizeUseCase *usecase.AuthorizeUseCase, userUseCase *usecase.UserUseCase) *AuthorizeHandler {
	return &AuthorizeHandler{
		authorizeUseCase: authorizeUseCase,
		userUseCase:      userUseCase,
	}
}

func (h *AuthorizeHandler) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	responseType := q.Get("response_type")
	scope := q.Get("scope")
	if scope == "" {
		scope = "openid"
	}
	state := q.Get("state")
	nonce := q.Get("nonce")
	codeChallenge := q.Get("code_challenge")
	codeChallengeMethod := q.Get("code_challenge_method")
	if codeChallengeMethod == "" {
		codeChallengeMethod = "S256"
	}

	if clientID == "" || redirectURI == "" || responseType == "" {
		httputil.Error(w, http.StatusBadRequest, "invalid_request", "missing required parameters")
		return
	}

	if responseType != "code" {
		httputil.Error(w, http.StatusBadRequest, "unsupported_response_type", "")
		return
	}

	if codeChallenge == "" {
		httputil.Error(w, http.StatusBadRequest, "invalid_request", "code_challenge is required")
		return
	}

	if codeChallengeMethod != "S256" {
		httputil.Error(w, http.StatusBadRequest, "invalid_request", "code_challenge_method must be S256")
		return
	}

	// OIDC `prompt=login` (Core §3.1.2.1) forces re-authentication. If the
	// caller requests it we ignore any existing session_id cookie *and*
	// delete the server-side session so the user has to enter credentials
	// again. RPs use this on their logout to prevent silent re-login.
	prompt := q.Get("prompt")
	forceLogin := false
	for _, p := range strings.Fields(prompt) {
		if p == "login" {
			forceLogin = true
			break
		}
	}

	loginRedirect := func() {
		http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", MaxAge: -1, Path: "/", HttpOnly: true})
		loginURL := "/auth/login?" + url.Values{
			"client_id":             {clientID},
			"redirect_uri":          {redirectURI},
			"state":                 {state},
			"scope":                 {scope},
			"nonce":                 {nonce},
			"code_challenge":        {codeChallenge},
			"code_challenge_method": {codeChallengeMethod},
		}.Encode()
		http.Redirect(w, r, loginURL, http.StatusFound)
	}

	sessionCookie, err := r.Cookie("session_id")
	if err != nil || sessionCookie.Value == "" {
		loginRedirect()
		return
	}

	if forceLogin {
		// Server-side delete so the old session ID can't be replayed.
		h.userUseCase.DeleteSession(r.Context(), sessionCookie.Value)
		loginRedirect()
		return
	}

	session, err := h.userUseCase.ValidateSession(r.Context(), sessionCookie.Value)
	if err != nil || session == nil {
		loginRedirect()
		return
	}

	result, err := h.authorizeUseCase.Authorize(r.Context(), usecase.AuthorizeParams{
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		ResponseType:        responseType,
		Scope:               scope,
		State:               state,
		Nonce:               nonce,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		UserID:              session.UserID,
	})

	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	http.Redirect(w, r, result.RedirectURI+"?code="+result.Code+"&state="+result.State, http.StatusFound)
}

func (h *AuthorizeHandler) HandleAuthorizePost(w http.ResponseWriter, r *http.Request) {
	// Parse form data and merge into query params for uniform handling
	r.ParseForm()
	q := r.URL.Query()
	for key, vals := range r.PostForm {
		if q.Get(key) == "" {
			q[key] = vals
		}
	}
	r.URL.RawQuery = q.Encode()
	h.HandleAuthorize(w, r)
}

// resolveUserID extracts the user ID from a session cookie.
func (h *AuthorizeHandler) resolveUserID(r *http.Request) (uuid.UUID, bool) {
	sessionCookie, err := r.Cookie("session_id")
	if err != nil || sessionCookie.Value == "" {
		return uuid.UUID{}, false
	}

	session, err := h.userUseCase.ValidateSession(r.Context(), sessionCookie.Value)
	if err != nil || session == nil {
		return uuid.UUID{}, false
	}

	return session.UserID, true
}
