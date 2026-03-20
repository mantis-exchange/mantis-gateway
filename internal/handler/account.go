package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AccountHandler proxies account-related requests to the account service.
type AccountHandler struct {
	accountAddr string
	httpClient  *http.Client
}

// NewAccountHandler creates a new AccountHandler that proxies to the account service.
func NewAccountHandler(accountAddr string) *AccountHandler {
	return &AccountHandler{
		accountAddr: accountAddr,
		httpClient:  &http.Client{Timeout: 10 * 1e9}, // 10s
	}
}

// Register handles POST /api/v1/account/register.
func (h *AccountHandler) Register(c *gin.Context) {
	h.proxy(c, "POST", "/api/v1/account/register")
}

// Login handles POST /api/v1/account/login.
func (h *AccountHandler) Login(c *gin.Context) {
	h.proxy(c, "POST", "/api/v1/account/login")
}

// GetAccount handles GET /api/v1/account.
func (h *AccountHandler) GetAccount(c *gin.Context) {
	h.proxy(c, "GET", "/api/v1/account")
}

// GetBalances handles GET /api/v1/account/balances.
func (h *AccountHandler) GetBalances(c *gin.Context) {
	h.proxy(c, "GET", "/api/v1/account/balances")
}

// GenerateAPIKeys handles POST /api/v1/account/api-keys.
func (h *AccountHandler) GenerateAPIKeys(c *gin.Context) {
	h.proxy(c, "POST", "/api/v1/account/api-keys")
}

// Faucet handles POST /api/v1/faucet.
func (h *AccountHandler) Faucet(c *gin.Context) {
	h.proxy(c, "POST", "/api/v1/faucet")
}

func (h *AccountHandler) proxy(c *gin.Context, method, path string) {
	url := h.accountAddr + path

	var bodyReader io.Reader
	if method == "POST" || method == "PUT" {
		bodyReader = c.Request.Body
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), method, url, bodyReader)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// Forward Authorization header for authenticated endpoints
	if auth := c.GetHeader("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "account service unavailable"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", body)
}
