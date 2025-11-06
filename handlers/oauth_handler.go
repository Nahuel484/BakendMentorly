package handlers

import (
	"fmt"
	"mentorly-backend/services"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OAuthHandler struct {
	db          *pgxpool.Pool
	authService *services.AuthService
}

func NewOAuthHandler(db *pgxpool.Pool) *OAuthHandler {
	return &OAuthHandler{
		db:          db,
		authService: services.NewAuthService(db),
	}
}

// GoogleCallbackHandler maneja el callback de Google OAuth
func (h *OAuthHandler) GoogleCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "authorization code not provided",
		})
		return
	}

	userInfo, err := services.ExchangeGoogleCode(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Failed to exchange code: %v", err),
		})
		return
	}

	h.handleOAuthLogin(c, userInfo)
}

// GitHubCallbackHandler maneja el callback de GitHub OAuth
func (h *OAuthHandler) GitHubCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "authorization code not provided",
		})
		return
	}

	userInfo, err := services.ExchangeGitHubCode(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Failed to exchange code: %v", err),
		})
		return
	}

	h.handleOAuthLogin(c, userInfo)
}

// LinkedInCallbackHandler maneja el callback de LinkedIn OAuth
func (h *OAuthHandler) LinkedInCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	errorParam := c.Query("error")
	errorDesc := c.Query("error_description")
	
	if errorParam != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("LinkedIn error: %s - %s", errorParam, errorDesc),
		})
		return
	}
	
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "authorization code not provided",
		})
		return
	}

	userInfo, err := services.ExchangeLinkedInCode(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Failed to exchange code: %v", err),
		})
		return
	}

	h.handleOAuthLogin(c, userInfo)
}

// handleOAuthLogin gestiona el login/registro con OAuth
func (h *OAuthHandler) handleOAuthLogin(c *gin.Context, oauthUser *services.OAuthUserInfo) {
	// Buscar o crear usuario con el proveedor OAuth
	user, err := h.authService.FindOrCreateOAuthUser(c.Request.Context(), oauthUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Error processing user: %v", err),
		})
		return
	}

	// Generar token JWT (usando int como ID)
	idPersona := int(user.ID)
	token, err := GenerateToken(idPersona, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Error generating token: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Login successful",
		"data": gin.H{
			"token":      token,
			"id_persona": idPersona,
			"email":      user.Email,
			"nombre":     user.Name,
		},
	})
}

// GetGoogleAuthURL retorna la URL de autenticación de Google
func (h *OAuthHandler) GetGoogleAuthURL(c *gin.Context) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid+profile+email",
		clientID, redirectURL,
	)

	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

// GetGitHubAuthURL retorna la URL de autenticación de GitHub
func (h *OAuthHandler) GetGitHubAuthURL(c *gin.Context) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	redirectURL := os.Getenv("GITHUB_REDIRECT_URL")

	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=user:email",
		clientID, redirectURL,
	)

	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}

// GetLinkedInAuthURL retorna la URL de autenticación de LinkedIn
func (h *OAuthHandler) GetLinkedInAuthURL(c *gin.Context) {
	clientID := os.Getenv("LINKEDIN_CLIENT_ID")
	redirectURL := os.Getenv("LINKEDIN_REDIRECT_URL")

	authURL := fmt.Sprintf(
		"https://www.linkedin.com/oauth/v2/authorization?response_type=code&client_id=%s&redirect_uri=%s&scope=openid%%20profile%%20email",
		clientID, redirectURL,
	)

	c.JSON(http.StatusOK, gin.H{"auth_url": authURL})
}