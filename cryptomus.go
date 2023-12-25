package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type CryptomusResult struct {
	UUID          string `json:"uuid"`
	OrderID       string `json:"order_id"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	PaymentURL    string `json:"url"`
	PaymentStatus string `json:"payment_status"`
}

type CryptomusPaymentResponse struct {
	State  int             `json:"state"`
	Result CryptomusResult `json:"result"`
}

type PaymentInfoRequest struct {
	OrderID string `json:"order_id"`
}

type PaymentInfoResponse struct {
	State  int `json:"state"`
	Result struct {
		UUID          string `json:"uuid"`
		OrderID       string `json:"order_id"`
		PaymentStatus string `json:"payment_status"`
	} `json:"result"`
}

var merchant string
var apiKey string
var urlCallback string

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	urlCallback = os.Getenv("URL_CALLBACK")
	merchant = os.Getenv("CRYPTOMUS_MERCHANT")
	apiKey = os.Getenv("CRYPTOMUS_APIKEY")
}
func generateSign(data map[string]string, apiKey string) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return ""
	}

	base64Data := base64.StdEncoding.EncodeToString(jsonData)
	toHash := base64Data + apiKey

	hasher := md5.New()
	hasher.Write([]byte(toHash))
	return hex.EncodeToString(hasher.Sum(nil))
}

func CreatePayment(amount, currency, orderID string) (*CryptomusPaymentResponse, error) {
	data := map[string]string{
		"amount":       amount,
		"currency":     "USD",
		"order_id":     orderID,
		"url_callback": urlCallback + "/webhook",
	}

	log.Printf("Request data: %+v", data)

	sign := generateSign(data, apiKey)

	requestBody, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling request body: %v", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.cryptomus.com/v1/payment", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return nil, err
	}
	req.Header.Set("merchant", merchant)
	req.Header.Set("sign", sign)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	log.Printf("Sending request to CryptoMus API")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("Response status: %s", resp.Status)
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil, err
	}

	log.Printf("Response body: %s", string(responseBody))

	var paymentResponse CryptomusPaymentResponse
	if err := json.Unmarshal(responseBody, &paymentResponse); err != nil {
		log.Printf("Error decoding response: %v", err)
		return nil, err
	}

	log.Printf("Payment response: %+v", paymentResponse)
	return &paymentResponse, nil
}

//Check payment

func FetchPaymentInfo(orderID string) (*PaymentInfoResponse, error) {
	signData := map[string]string{
		"order_id": orderID,
	}
	sign := generateSign(signData, apiKey)

	data := PaymentInfoRequest{
		OrderID: orderID,
	}

	requestBody, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.cryptomus.com/v1/payment/info", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("merchant", merchant)
	req.Header.Set("sign", sign)
	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Читаем ответ
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Разбираем ответ
	var paymentInfo PaymentInfoResponse
	if err := json.Unmarshal(responseBody, &paymentInfo); err != nil {
		return nil, err
	}

	return &paymentInfo, nil
}
