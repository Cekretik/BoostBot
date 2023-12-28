package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

type UserPaymentStatus struct {
	ChatID          int64
	CurrentState    string
	ReplenishAmount float64
	OrderID         string
	PaymentStatus   string
}

type CreatePaymentRequest struct {
	ChatID   int64   `json:"chat_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

var userPaymentStatuses map[int64]*UserPaymentStatus = make(map[int64]*UserPaymentStatus)

type CryptomusWebhookData struct {
	Type              string `json:"type"`
	UUID              string `json:"uuid"`
	OrderID           string `json:"order_id"`
	Amount            string `json:"amount"`
	PaymentAmount     string `json:"payment_amount"`
	PaymentAmountUSD  string `json:"payment_amount_usd"`
	MerchantAmount    string `json:"merchant_amount"`
	Commission        string `json:"commission"`
	IsFinal           bool   `json:"is_final"`
	PaymentStatus     string `json:"status"`
	From              string `json:"from"`
	WalletAddressUUID string `json:"wallet_address_uuid"`
	Network           string `json:"network"`
	Currency          string `json:"currency"`
	PayerCurrency     string `json:"payer_currency"`
	AdditionalData    string `json:"additional_data"`
	Convert           struct {
		ToCurrency string `json:"to_currency"`
		Commission string `json:"commission"`
		Rate       string `json:"rate"`
		Amount     string `json:"amount"`
	} `json:"convert"`
	TxID string `json:"txid"`
	Sign string `json:"sign"`
}

func handleReplenishCommand(bot *tgbotapi.BotAPI, chatID int64) {
	userPaymentStatus := updateUserStatus(chatID)
	userPaymentStatus.CurrentState = "awaitingPaymentSystem"

	msgText := ("Выберите платежную систему")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Cryptomus", "cryptomus"),
		),
	)
	cancelKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = cancelKeyboard
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

func handleCryptomusButton(bot *tgbotapi.BotAPI, chatID int64) {
	userPaymentStatus := updateUserStatus(chatID)
	userPaymentStatus.CurrentState = "awaitingAmount"
	userPaymentStatus.OrderID = createOrderID(chatID+44984985, time.Now().Unix())
	msgText := "Введите желаемую сумму в долларах."
	cancelKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = cancelKeyboard
	bot.Send(msg)
}

func handlePaymentInput(db *gorm.DB, bot *tgbotapi.BotAPI, chatID int64, amountText string) {
	userPaymentStatus := updateUserStatus(chatID)
	if userPaymentStatus.CurrentState == "awaitingAmount" {
		amount, err := strconv.ParseFloat(amountText, 64)
		if err != nil || amount <= 0 {
			msg := tgbotapi.NewMessage(chatID, "Введите корректную сумму в долларах.")
			userPaymentStatuses[chatID] = userPaymentStatus
			cancelKeyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Отмена"),
				),
			)
			msg.ReplyMarkup = cancelKeyboard
			bot.Send(msg)
			return
		}

		if isOrderExpired(userPaymentStatus) {
			userPaymentStatus.OrderID = createOrderID(chatID, time.Now().Unix())
		}

		userPaymentStatuses[chatID] = userPaymentStatus
		createAndSendPaymentLink(db, bot, chatID, amount, userPaymentStatus.OrderID, time.Now().Unix())
	}
}

func createAndSendPaymentLink(db *gorm.DB, bot *tgbotapi.BotAPI, chatID int64, amount float64, orderID string, timestamp int64) {
	paymentResponse, err := CreatePayment(fmt.Sprintf("%.4f", amount), "USD", orderID)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при создании платежа."))
		return
	}

	newPayment := Payments{
		ChatID:  int(chatID),
		OrderID: orderID,
		Amount:  amount,
		Url:     paymentResponse.Result.PaymentURL,
		Status:  paymentResponse.Result.PaymentStatus,
		Type:    "cryptomus",
	}
	db.Create(&newPayment)
	paymentURL := paymentResponse.Result.PaymentURL
	if paymentURL == "" {
		bot.Send(tgbotapi.NewMessage(chatID, "Не удалось получить ссылку на платеж,попробуйте снова."))
	} else {
		inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("Оплатить", paymentURL),
			),
		)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Для пополнения на сумму $%.4f нажмите на кнопку оплатить:", amount))
		msg.ReplyMarkup = inlineKeyboard
		bot.Send(msg)
		delete(userPaymentStatuses, chatID)
		sendStandardKeyboardAfterPayment(bot, chatID)
	}
}
func handleWebhook(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	var webhookData CryptomusWebhookData
	log.Printf("MISHKA GUMI BEAR: %v", r.Body)
	err := json.NewDecoder(r.Body).Decode(&webhookData)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("Error decoding webhook request: %v", err)
		return
	}

	orderID := webhookData.OrderID
	var payment Payments
	if err := db.Where("order_id = ?", orderID).First(&payment).Error; err != nil {
		log.Printf("Error retrieving payment: %v", err)
		return
	}

	switch webhookData.PaymentStatus {
	case "paid":
		if payment.Status != "paid" {
			updatePaymentStatusInDB(db, orderID, "paid")
			err = UpdateUserBalance(db, int64(payment.ChatID), payment.Amount)
			if err != nil {
				log.Printf("Error updating user balance: %v", err)
			}
		}
	default:
		log.Printf("Unhandled payment status %s for orderID %s", webhookData.PaymentStatus, webhookData.OrderID)
	}

	w.WriteHeader(http.StatusOK)
}

func updateUserStatus(chatID int64) *UserPaymentStatus {
	if status, exists := userPaymentStatuses[chatID]; exists {
		if isOrderExpired(status) {
			// Если текущий заказ истек, обновляем информацию
			status.OrderID = createOrderID(chatID, time.Now().Unix())
			status.PaymentStatus = "cancel"
			userPaymentStatuses[chatID] = status
		}
		return status
	}
	// Если статуса нет, создаем новый
	newUserStatus := &UserPaymentStatus{
		ChatID:        chatID,
		OrderID:       createOrderID(chatID, time.Now().Unix()),
		PaymentStatus: "",
	}
	userPaymentStatuses[chatID] = newUserStatus
	return newUserStatus
}

func UpdateUserBalance(db *gorm.DB, userID int64, amount float64) error {
	var user UserState
	if err := db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		return err
	}

	user.Balance += amount
	return db.Save(&user).Error
}

func isOrderExpired(userStatus *UserPaymentStatus) bool {
	return userStatus.PaymentStatus == "cancel" || userStatus.PaymentStatus == "fail"
}

func handleCreatePayment(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Создание платежа
	orderID := createOrderID(req.ChatID, time.Now().Unix()) // функция для генерации OrderID
	paymentResponse, err := CreatePayment(strconv.FormatFloat(req.Amount, 'f', 2, 64), req.Currency, orderID)
	if err != nil {
		http.Error(w, "Failed to create payment", http.StatusInternalServerError)
		return
	}

	// Сохранение информации о платеже в БД
	newPayment := Payments{
		ChatID:  int(req.ChatID),
		OrderID: orderID,
		Amount:  req.Amount,
		Url:     paymentResponse.Result.PaymentURL,
		Status:  "pending",
		Type:    "cryptomus",
	}
	db.Create(&newPayment)

	// Отправка ответа
	response := map[string]string{
		"url":    paymentResponse.Result.PaymentURL,
		"status": "success",
	}
	json.NewEncoder(w).Encode(response)
}

func createOrderID(chatID int64, timestamp int64) string {
	return fmt.Sprintf("order_%d_%d", chatID, timestamp)
}

func startHTTPServer(db *gorm.DB) {
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		handleWebhook(db, w, r)
	})

	http.HandleFunc("/create_payment", func(w http.ResponseWriter, r *http.Request) {
		handleCreatePayment(w, r, db)
	})

	log.Println("HTTP server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
