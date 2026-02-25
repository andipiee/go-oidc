package handler

import (
	"net/http"

	"github.com/andipiee/go-oidc/internal/application/usecase"
	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	deviceUseCase *usecase.DeviceUseCase
}

func NewDeviceHandler(deviceUseCase *usecase.DeviceUseCase) *DeviceHandler {
	return &DeviceHandler{deviceUseCase: deviceUseCase}
}

func (h *DeviceHandler) HandleDeviceAuthorize(c *gin.Context) {
	clientID := c.PostForm("client_id")
	scope := c.DefaultPostForm("scope", "openid")

	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	deviceCode, err := h.deviceUseCase.CreateDeviceCode(c.Request.Context(), clientID, scope)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_code":               deviceCode.DeviceCode,
		"user_code":                 deviceCode.UserCode,
		"verification_uri":          "/oidc/device",
		"verification_uri_complete": "/oidc/device?user_code=" + deviceCode.UserCode,
		"expires_in":                300,
		"interval":                  deviceCode.PollingInterval,
	})
}

func (h *DeviceHandler) HandleDevice(c *gin.Context) {
	userCode := c.Query("user_code")
	if userCode == "" {
		c.HTML(http.StatusOK, "device", gin.H{})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_code": userCode})
}
