package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type MercadoPagoService struct {
	accessToken string
	frontendURL string
	client      *http.Client
}

func NewMercadoPagoService() *MercadoPagoService {
	return &MercadoPagoService{
		accessToken: os.Getenv("MP_ACCESS_TOKEN"),
		frontendURL: os.Getenv("FRONTEND_URL"),
		client:      &http.Client{},
	}
}

type mpItem struct {
	Title      string  `json:"title"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	CurrencyID string  `json:"currency_id"`
}

type mpPreferenceRequest struct {
	Items      []mpItem          `json:"items"`
	BackURLs   map[string]string `json:"back_urls"`
	AutoReturn string            `json:"auto_return"`
	Metadata   map[string]any    `json:"metadata"`
}

type mpPreferenceResponse struct {
	InitPoint        string `json:"init_point"`
	SandboxInitPoint string `json:"sandbox_init_point"`
}

func (m *MercadoPagoService) CreatePreference(ctx context.Context, title string, price float64, userID, planID int) (string, error) {
	if m.accessToken == "" {
		return "", fmt.Errorf("MP_ACCESS_TOKEN no configurado")
	}

	body := mpPreferenceRequest{
		Items: []mpItem{
			{Title: title, Quantity: 1, UnitPrice: price, CurrencyID: "ARS"},
		},
		BackURLs: map[string]string{
			"success": m.frontendURL + "/pago/success",
			"failure": m.frontendURL + "/pago/failure",
			"pending": m.frontendURL + "/pago/pending",
		},
		AutoReturn: "approved",
		Metadata: map[string]any{
			"user_id": userID,
			"plan_id": planID,
		},
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://api.mercadopago.com/checkout/preferences",
		bytes.NewReader(jsonBody),
	)

	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+m.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Mercado Pago devolvió status %d", resp.StatusCode)
	}

	var mpResp mpPreferenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&mpResp); err != nil {
		return "", err
	}

	return mpResp.InitPoint, nil
}
