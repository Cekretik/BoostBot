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
	UUID          string `json:"uuid"`
	OrderID       string `json:"order_id"`
	PaymentStatus string `json:"payment_status"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	UrlCallback   string `json:"url_callback"`
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
	paymentResponse, err := CreatePayment(fmt.Sprintf("%.4f", amount), "USD", fmt.Sprintf("order_%d_%d", chatID, timestamp))
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
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Для пополнения на сумму $%.4f перейдите по ссылке: %s", amount, paymentURL)))
		delete(userPaymentStatuses, chatID)
		sendStandardKeyboardAfterPayment(bot, chatID)
	}
}

func handleWebhook(bot *tgbotapi.BotAPI, db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	var webhookData CryptomusWebhookData
	err := json.NewDecoder(r.Body).Decode(&webhookData)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("Error decoding webhook request: %v", err)
		return
	}
	chatID, err := extractChatIDFromOrderID(webhookData.OrderID)
	if err != nil {
		log.Printf("Error extracting chatID from orderID: %v", err)
		return
	}

	if userStatus, exists := userPaymentStatuses[chatID]; exists {
		userStatus.PaymentStatus = webhookData.PaymentStatus
		userPaymentStatuses[chatID] = userStatus
	}
	switch webhookData.PaymentStatus {
	case "paid":
		orderID := webhookData.OrderID
		chatID, err := extractChatIDFromOrderID(orderID)
		if err != nil {
			log.Printf("Error extracting chatID from orderID: %v", err)
			return
		}
		amount, err := strconv.ParseFloat(webhookData.Amount, 64)
		if err != nil {
			log.Printf("Error parsing amount: %v", err)
			return
		}
		updatePaymentStatusInDB(db, orderID, "paid")

		err = UpdateUserBalance(db, chatID, amount)
		if err != nil {
			log.Printf("Error updating user balance: %v", err)
		}
	case "check":
		orderID := webhookData.OrderID
		chatID, err := extractChatIDFromOrderID(orderID)
		if err != nil {
			log.Printf("Error extracting chatID from orderID: %v", err)
			return
		}
		updatePaymentStatusInDB(db, orderID, "check")
		err = UpdateUserBalance(db, chatID, 0)
		if err != nil {
			log.Printf("Error updating user balance: %v", err)
		}
	case "cancel":
		orderID := webhookData.OrderID
		chatID, err := extractChatIDFromOrderID(orderID)
		if err != nil {
			log.Printf("Error extracting chatID from orderID: %v", err)
			return
		}
		updatePaymentStatusInDB(db, orderID, "cancel")
		err = UpdateUserBalance(db, chatID, 0)
		if err != nil {
			log.Printf("Error updating user balance: %v", err)
		}
		delete(userPaymentStatuses, chatID)
	case "fail":
		orderID := webhookData.OrderID
		chatID, err := extractChatIDFromOrderID(orderID)
		if err != nil {
			log.Printf("Error extracting chatID from orderID: %v", err)
			return
		}
		updatePaymentStatusInDB(db, orderID, "fail")
		err = UpdateUserBalance(db, chatID, 0)
		if err != nil {
			log.Printf("Error updating user balance: %v", err)
		}
		delete(userPaymentStatuses, chatID)
	default:
		log.Printf("Unhandled payment status %s for orderID %s", webhookData.PaymentStatus, webhookData.OrderID)
	}

	w.WriteHeader(http.StatusOK)
}

func extractChatIDFromOrderID(orderID string) (int64, error) {
	var chatID int64
	timestamp := time.Now().Unix()
	_, err := fmt.Sscanf(orderID, "order%d_%d", &chatID, timestamp)
	if err != nil {
		return 0, err
	}
	return chatID, nil
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

func startHTTPServer(bot *tgbotapi.BotAPI, db *gorm.DB) {
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		handleWebhook(bot, db, w, r)
	})

	http.HandleFunc("/create_payment", func(w http.ResponseWriter, r *http.Request) {
		handleCreatePayment(w, r, db)
	})

	log.Println("HTTP server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
