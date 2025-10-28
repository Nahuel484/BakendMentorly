package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type OAuthProvider string

const (
	GoogleProvider   OAuthProvider = "google"
	GitHubProvider   OAuthProvider = "github"
	LinkedInProvider OAuthProvider = "linkedin"
)

type OAuthUserInfo struct {
	ID       string
	Email    string
	Name     string
	Avatar   string
	Provider OAuthProvider
}

type GoogleTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type GoogleUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

type GitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type GitHubUserInfo struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

type LinkedInTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type LinkedInUserInfo struct {
	ID                 string `json:"id"`
	LocalizedFirstName string `json:"localizedFirstName"`
	LocalizedLastName  string `json:"localizedLastName"`
	ProfilePicture     struct {
		DisplayImage string `json:"displayImage"`
	} `json:"profilePicture"`
}

type LinkedInEmailResponse struct {
	Elements []struct {
		Handle string `json:"handle"`
	} `json:"elements"`
}

// ExchangeGoogleCode intercambia el código de Google por token y obtiene info del usuario
func ExchangeGoogleCode(code string) (*OAuthUserInfo, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	tokenURL := "https://oauth2.googleapis.com/token"
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURL)
	data.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("error al obtener token de Google: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp GoogleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("error al decodificar token de Google: %w", err)
	}

	// Obtener información del usuario
	userInfoURL := "https://www.googleapis.com/oauth2/v2/userinfo"
	req, _ := http.NewRequest("GET", userInfoURL, nil)
	req.Header.Add("Authorization", "Bearer "+tokenResp.AccessToken)

	client := &http.Client{}
	userResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al obtener info de usuario de Google: %w", err)
	}
	defer userResp.Body.Close()

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("error al decodificar info de usuario de Google: %w", err)
	}

	return &OAuthUserInfo{
		ID:       userInfo.ID,
		Email:    userInfo.Email,
		Name:     userInfo.Name,
		Avatar:   userInfo.Picture,
		Provider: GoogleProvider,
	}, nil
}

// ExchangeGitHubCode intercambia el código de GitHub por token y obtiene info del usuario
func ExchangeGitHubCode(code string) (*OAuthUserInfo, error) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	tokenURL := "https://github.com/login/oauth/access_token"
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	req, _ := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al obtener token de GitHub: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp GitHubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("error al decodificar token de GitHub: %w", err)
	}

	// Obtener información del usuario
	userInfoURL := "https://api.github.com/user"
	req, _ = http.NewRequest("GET", userInfoURL, nil)
	req.Header.Add("Authorization", "Bearer "+tokenResp.AccessToken)
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	userResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al obtener info de usuario de GitHub: %w", err)
	}
	defer userResp.Body.Close()

	var userInfo GitHubUserInfo
	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("error al decodificar info de usuario de GitHub: %w", err)
	}

	// Si no hay email en el perfil, obtenerlo del endpoint de emails
	if userInfo.Email == "" {
		userInfo.Email = getGitHubUserEmail(tokenResp.AccessToken)
	}

	name := userInfo.Name
	if name == "" {
		name = userInfo.Login
	}

	return &OAuthUserInfo{
		ID:       fmt.Sprintf("%d", userInfo.ID),
		Email:    userInfo.Email,
		Name:     name,
		Avatar:   userInfo.AvatarURL,
		Provider: GitHubProvider,
	}, nil
}

// getGitHubUserEmail obtiene el email del usuario de GitHub
func getGitHubUserEmail(accessToken string) string {
	emailURL := "https://api.github.com/user/emails"
	req, _ := http.NewRequest("GET", emailURL, nil)
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var emails []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return ""
	}

	for _, email := range emails {
		if primary, ok := email["primary"].(bool); ok && primary {
			if e, ok := email["email"].(string); ok {
				return e
			}
		}
	}

	return ""
}

// ExchangeLinkedInCode intercambia el código de LinkedIn por token y obtiene info del usuario
func ExchangeLinkedInCode(code string) (*OAuthUserInfo, error) {
	clientID := os.Getenv("LINKEDIN_CLIENT_ID")
	clientSecret := os.Getenv("LINKEDIN_CLIENT_SECRET")
	redirectURL := os.Getenv("LINKEDIN_REDIRECT_URL")

	tokenURL := "https://www.linkedin.com/oauth/v2/accessToken"
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURL)

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("error al obtener token de LinkedIn: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp LinkedInTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("error al decodificar token de LinkedIn: %w", err)
	}

	// Obtener información del usuario
	userInfoURL := "https://api.linkedin.com/v2/me?projection=(id,localizedFirstName,localizedLastName,profilePicture(displayImage))"
	req, _ := http.NewRequest("GET", userInfoURL, nil)
	req.Header.Add("Authorization", "Bearer "+tokenResp.AccessToken)

	client := &http.Client{}
	userResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al obtener info de usuario de LinkedIn: %w", err)
	}
	defer userResp.Body.Close()

	var userInfo LinkedInUserInfo
	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("error al decodificar info de usuario de LinkedIn: %w", err)
	}

	// Obtener email del usuario
	email := getLinkedInUserEmail(tokenResp.AccessToken)

	fullName := strings.TrimSpace(userInfo.LocalizedFirstName + " " + userInfo.LocalizedLastName)

	return &OAuthUserInfo{
		ID:       userInfo.ID,
		Email:    email,
		Name:     fullName,
		Avatar:   userInfo.ProfilePicture.DisplayImage,
		Provider: LinkedInProvider,
	}, nil
}

// getLinkedInUserEmail obtiene el email del usuario de LinkedIn
func getLinkedInUserEmail(accessToken string) string {
	emailURL := "https://api.linkedin.com/v2/emailAddress?q=primary&projection=(elements*(handle~))"
	req, _ := http.NewRequest("GET", emailURL, nil)
	req.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var emailResp LinkedInEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&emailResp); err != nil {
		return ""
	}

	if len(emailResp.Elements) > 0 {
		return emailResp.Elements[0].Handle
	}

	return ""
}
