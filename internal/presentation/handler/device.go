package handler

import (
	"html/template"
	"net/http"

	"github.com/andipiee/go-oidc/internal/application/usecase"
	"github.com/andipiee/go-oidc/internal/presentation/httputil"
)

type DeviceHandler struct {
	deviceUseCase *usecase.DeviceUseCase
	tmpl          *template.Template
}

func NewDeviceHandler(deviceUseCase *usecase.DeviceUseCase, tmpl *template.Template) *DeviceHandler {
	return &DeviceHandler{deviceUseCase: deviceUseCase, tmpl: tmpl}
}

func (h *DeviceHandler) HandleDeviceAuthorize(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	clientID := r.FormValue("client_id")
	scope := r.FormValue("scope")
	if scope == "" {
		scope = "openid"
	}

	if clientID == "" {
		httputil.Error(w, http.StatusBadRequest, "invalid_request", "")
		return
	}

	deviceCode, err := h.deviceUseCase.CreateDeviceCode(r.Context(), clientID, scope)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid_client", "")
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"device_code":               deviceCode.DeviceCode,
		"user_code":                 deviceCode.UserCode,
		"verification_uri":          "/oidc/device",
		"verification_uri_complete": "/oidc/device?user_code=" + deviceCode.UserCode,
		"expires_in":                300,
		"interval":                  deviceCode.PollingInterval,
	})
}

func (h *DeviceHandler) HandleDevice(w http.ResponseWriter, r *http.Request) {
	userCode := r.URL.Query().Get("user_code")
	if userCode == "" {
		httputil.HTML(w, h.tmpl, "device", http.StatusOK, map[string]any{})
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]any{"user_code": userCode})
}
