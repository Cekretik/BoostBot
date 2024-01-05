package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var (
	PayOKPaymentURL = "https://payok.io/pay"
	SecretKey       string
	ShopID          string
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	SecretKey = os.Getenv("SECRET_KEY")
	ShopID = os.Getenv("PAYOK_ID")
}

type PayOKNotification struct {
	PaymentID string `json:"order_id"`
}

func generatePayOKSign(params map[string]string) string {
	var orderedKeys = []string{"amount", "payment", "shop", "currency", "desc"}
	var buffer bytes.Buffer
	for _, key := range orderedKeys {
		buffer.WriteString(params[key])
		buffer.WriteString("|")
	}
	buffer.WriteString(SecretKey)

	signatureString := buffer.String()
	log.Printf("String to sign: %v", signatureString)

	hasher := md5.New()
	hasher.Write([]byte(signatureString))
	signature := hex.EncodeToString(hasher.Sum(nil))
	log.Printf("Generated PayOK sign: %v", signature)
	return signature
}

func CreatePayOKPayment(amount, paymentID, currency, description string) (string, error) {
	params := map[string]string{
		"amount":   amount,
		"payment":  paymentID,
		"shop":     ShopID,
		"currency": currency,
		"desc":     description,
	}

	// Генерация подписи
	sign := generatePayOKSign(params)
	params["sign"] = sign

	values := url.Values{}
	for key, value := range params {
		values.Add(key, value)
	}

	paymentURL := fmt.Sprintf("%s?%s", PayOKPaymentURL, values.Encode())
	return paymentURL, nil
}
func handlePayOKNotification(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	paymentID := r.FormValue("payment_id")

	log.Printf("Payment ID: %v", paymentID)
	var payment Payments
	if err := db.Where("order_id = ?", paymentID).First(&payment).Error; err != nil {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	// Проверка, был ли платеж уже обработан
	if payment.Status == "paid" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Обновление статуса платежа
	if err := updatePaymentStatusInDB(db, payment.OrderID, "paid"); err != nil {
		log.Printf("Error updating payment status: %v", err)
		http.Error(w, "Error updating payment status", http.StatusInternalServerError)
		return
	}

	if err := UpdateUserBalance(db, int64(payment.ChatID), payment.Amount); err != nil {
		log.Printf("balance %v", payment.Amount)
		log.Printf("Error updating user balance: %v", err)
		http.Error(w, "Error updating user balance", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
