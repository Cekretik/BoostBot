package payment

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/Cekretik/BoostBot/database"
	"github.com/Cekretik/BoostBot/models"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var (
	AAIOPaymentURL = "https://aaio.so/merchant/pay"
	MerchantID     string
	SecretKey1     string
	SecretKey2     string
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	MerchantID = os.Getenv("AAIO_SHOPID")
	SecretKey1 = os.Getenv("AAIO_KEY1")
	SecretKey2 = os.Getenv("AAIO_KEY2")
}

type Payment struct {
	ID     uint
	UserID uint
	Amount float64
	Status string
}

func generateAAIOSign(params map[string]string) string {
	var signParts = []string{params["merchant_id"], params["amount"], params["currency"], SecretKey1, params["order_id"]}
	signString := bytes.NewBufferString("")
	for i, part := range signParts {
		if i > 0 {
			signString.WriteString(":")
		}
		signString.WriteString(part)
	}

	hasher := sha256.New()
	hasher.Write(signString.Bytes())
	signature := hex.EncodeToString(hasher.Sum(nil))
	return signature
}

func CreateAAIOPayment(amount, orderID, currency, description, email, lang string) (string, error) {
	params := map[string]string{
		"merchant_id": MerchantID,
		"amount":      amount,
		"currency":    currency,
		"order_id":    orderID,
		"desc":        description,
		"email":       email,
		"lang":        lang,
	}

	sign := generateAAIOSign(params)
	params["sign"] = sign

	values := url.Values{}
	for key, value := range params {
		values.Add(key, value)
	}

	paymentURL := fmt.Sprintf("%s?%s", AAIOPaymentURL, values.Encode())
	return paymentURL, nil
}

func handleAAIONotification(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}
	orderID := r.FormValue("payment_id")
	var payment models.Payments
	if err := db.Where("order_id = ?", orderID).First(&payment).Error; err != nil {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}

	if payment.Status == "success" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := database.UpdatePaymentStatusInDB(db, payment.OrderID, "success"); err != nil {
		log.Printf("Error updating payment status: %v", err)
		http.Error(w, "Error updating payment status", http.StatusInternalServerError)
		return
	}

	if err := database.UpdateUserBalance(db, int64(payment.ChatID), payment.Amount); err != nil {
		log.Printf("balance %v", payment.Amount)
		log.Printf("Error updating user balance: %v", err)
		http.Error(w, "Error updating user balance", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
