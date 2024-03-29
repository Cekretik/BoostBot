package payment

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Cekretik/BoostBot/api"
	"github.com/Cekretik/BoostBot/database"
	"github.com/Cekretik/BoostBot/functionality"
	"github.com/Cekretik/BoostBot/models"
	tgbotapi "github.com/Cekretik/telegram-bot-api-master"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var port string

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	port = os.Getenv("PORT")
}

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

var UserPaymentStatuses map[int64]*UserPaymentStatus = make(map[int64]*UserPaymentStatus)

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

func HandleReplenishCommand(bot *tgbotapi.BotAPI, chatID int64) {
	userPaymentStatus := updateUserStatus(chatID)
	userPaymentStatus.CurrentState = "awaitingPaymentSystem"

	msgText := ("Выберите платежную систему")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("СБП|RUB", "AAIO_SBP"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("RU Карта|RUB", "AAIO_RU"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("USDT", "cryptomus_USDT"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("BTC", "cryptomus_BTC"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("MATIC", "cryptomus_MATIC"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Другая Крипта", "cryptomus_OTHER"),
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

func HandleCryptomusButton(bot *tgbotapi.BotAPI, chatID int64, db *gorm.DB) {
	userPaymentStatus := updateUserStatus(chatID)
	userPaymentStatus.CurrentState = "awaitingAmount"
	log.Printf("chuba %v", userPaymentStatus)
	userPaymentStatus.OrderID = createOrderID(chatID, time.Now().Unix())

	var user models.UserState
	if err := db.Where("user_id = ?", chatID).First(&user).Error; err != nil {
		log.Printf("Error fetching user state: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "Произошла ошибка."))
		return
	}

	msgText := "Введите желаемую сумму в долларах."
	if user.Currency == "RUB" {
		msgText = "Введите желаемую сумму в рублях."
	}

	cancelKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = cancelKeyboard
	bot.Send(msg)
}

func HandleAAIOButton(bot *tgbotapi.BotAPI, chatID int64, db *gorm.DB) {
	userPaymentStatus := updateUserStatus(chatID)
	userPaymentStatus.CurrentState = "awaitingAmountAAIO"
	orderID := createOrderID(chatID, time.Now().Unix())
	userPaymentStatus.OrderID = orderID

	var user models.UserState
	if err := db.Where("user_id = ?", chatID).First(&user).Error; err != nil {
		log.Printf("Error fetching user state: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "Произошла ошибка."))
		return
	}

	msgText := "Введите желаемую сумму в долларах."
	if user.Currency == "RUB" {
		msgText = "Введите желаемую сумму в рублях."
	}

	cancelKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = cancelKeyboard
	bot.Send(msg)
	UserPaymentStatuses[chatID] = userPaymentStatus
}
func HandlePaymentInput(db *gorm.DB, bot *tgbotapi.BotAPI, chatID int64, amountText string) {
	userPaymentStatus := updateUserStatus(chatID)

	if userPaymentStatus.CurrentState == "awaitingAmount" {
		var user models.UserState
		if err := db.Where("user_id = ?", chatID).First(&user).Error; err != nil {
			log.Printf("Error fetching user state: %v", err)
			return
		}

		amount, err := strconv.ParseFloat(amountText, 64)
		if err != nil || amount <= 0 {
			msg := tgbotapi.NewMessage(chatID, "Введите корректную сумму.")
			UserPaymentStatuses[chatID] = userPaymentStatus
			cancelKeyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Отмена"),
				),
			)
			msg.ReplyMarkup = cancelKeyboard
			bot.Send(msg)
			return
		}

		if user.Currency == "RUB" {
			rate := api.GetCurrentCurrencyRate()
			amount = functionality.ConvertAmount(amount, rate, false)
		}

		UserPaymentStatuses[chatID] = userPaymentStatus
		CreateAndSendPaymentLink(db, bot, chatID, amount, userPaymentStatus.OrderID, time.Now().Unix())
	}
}

func HandlePaymentInputAAIO(db *gorm.DB, bot *tgbotapi.BotAPI, chatID int64, amountText string) {
	userPaymentStatus := updateUserStatus(chatID)
	if userPaymentStatus.CurrentState == "awaitingAmountAAIO" {
		var user models.UserState
		if err := db.Where("user_id = ?", chatID).First(&user).Error; err != nil {
			log.Printf("Error fetching user state: %v", err)
			return
		}

		amount, err := strconv.ParseFloat(amountText, 64)
		if err != nil || amount <= 0 {
			msg := tgbotapi.NewMessage(chatID, "Введите корректную сумму.")
			UserPaymentStatuses[chatID] = userPaymentStatus
			cancelKeyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Отмена"),
				),
			)
			msg.ReplyMarkup = cancelKeyboard
			bot.Send(msg)
			return
		}
		originalAmount := amount
		currency := "USD"
		if user.Currency == "RUB" {
			currency = "RUB"
		} else {
			amount = originalAmount
		}
		createAndSendPaymentLinkAAIO(db, bot, chatID, amount, userPaymentStatus.OrderID, time.Now().Unix(), currency)
		userPaymentStatus.CurrentState = ""
		UserPaymentStatuses[chatID] = userPaymentStatus
	}
}
func CreateAndSendPaymentLink(db *gorm.DB, bot *tgbotapi.BotAPI, chatID int64, amount float64, orderID string, timestamp int64) {
	paymentResponse, err := CreatePayment(fmt.Sprintf("%.4f", amount), "USD", orderID)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при создании платежа."))
		return
	}

	newPayment := models.Payments{
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
				tgbotapi.NewInlineKeyboardButtonData("🎁Промокод", "promo"),
			),
		)
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Для пополнения на сумму $%.4f нажмите на кнопку оплатить:", amount))
		msg.ReplyMarkup = inlineKeyboard
		bot.Send(msg)
		delete(UserPaymentStatuses, chatID)
		functionality.SendStandardKeyboardAfterPayment(bot, chatID)
	}
}
func createAndSendPaymentLinkAAIO(db *gorm.DB, bot *tgbotapi.BotAPI, chatID int64, amount float64, orderID string, timestamp int64, currency string) {
	originalAmount := amount
	if currency == "RUB" {
		rate := api.GetCurrentCurrencyRate()
		amount = functionality.ConvertAmount(originalAmount, rate, false)
	}

	paymentURL, err := CreateAAIOPayment(fmt.Sprintf("%.2f", originalAmount), orderID, currency, "Пополнение баланса", "", "ru")
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при создании платежа."))
		return
	}

	newPayment := models.Payments{
		ChatID:  int(chatID),
		OrderID: orderID,
		Amount:  amount,
		Url:     paymentURL,
		Status:  "check",
		Type:    "aaio",
	}
	db.Create(&newPayment)

	paymentMessage := fmt.Sprintf("Для пополнения на сумму $%.4f нажмите на кнопку оплатить:", amount)
	if currency == "RUB" {
		paymentMessage = fmt.Sprintf("Для пополнения на сумму %.4f₽ нажмите на кнопку оплатить:", originalAmount)
	}

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Оплатить", paymentURL),
			tgbotapi.NewInlineKeyboardButtonData("🎁Промокод", "promo"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, paymentMessage)
	msg.ReplyMarkup = inlineKeyboard
	bot.Send(msg)
	delete(UserPaymentStatuses, chatID)
	functionality.SendStandardKeyboardAfterPayment(bot, chatID)
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
	var payment models.Payments
	if err := db.Where("order_id = ?", orderID).First(&payment).Error; err != nil {
		log.Printf("Error retrieving payment: %v", err)
		return
	}

	switch webhookData.PaymentStatus {
	case "paid":
		if payment.Status != "paid" {
			database.UpdatePaymentStatusInDB(db, orderID, "paid")
			err = database.UpdateUserBalance(db, int64(payment.ChatID), payment.Amount)
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
	if status, exists := UserPaymentStatuses[chatID]; exists {
		if IsOrderExpired(status) {
			status.OrderID = createOrderID(chatID, time.Now().Unix())
			status.PaymentStatus = "cancel"
			UserPaymentStatuses[chatID] = status
		}
		return status
	}
	newUserStatus := &UserPaymentStatus{
		ChatID:        chatID,
		OrderID:       createOrderID(chatID, time.Now().Unix()),
		PaymentStatus: "",
	}
	UserPaymentStatuses[chatID] = newUserStatus
	return newUserStatus
}

func IsOrderExpired(userStatus *UserPaymentStatus) bool {
	return userStatus.PaymentStatus == "cancel" || userStatus.PaymentStatus == "fail"
}

func HandleCreatePayment(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	orderID := createOrderID(req.ChatID, time.Now().Unix()) // функция для генерации OrderID
	paymentResponse, err := CreatePayment(strconv.FormatFloat(req.Amount, 'f', 2, 64), req.Currency, orderID)
	if err != nil {
		http.Error(w, "Failed to create payment", http.StatusInternalServerError)
		return
	}

	newPayment := models.Payments{
		ChatID:  int(req.ChatID),
		OrderID: orderID,
		Amount:  req.Amount,
		Url:     paymentResponse.Result.PaymentURL,
		Status:  "pending",
		Type:    "cryptomus",
	}
	db.Create(&newPayment)

	response := map[string]string{
		"url":    paymentResponse.Result.PaymentURL,
		"status": "success",
	}
	json.NewEncoder(w).Encode(response)
}
func HandleCreatePaymentAAIO(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	orderID := createOrderID(req.ChatID, time.Now().Unix())

	amountFormatted := fmt.Sprintf("%.2f", req.Amount)
	paymentURL, err := CreateAAIOPayment(amountFormatted, orderID, req.Currency, "Пополнение баланса", "", "ru")
	if err != nil {
		http.Error(w, "Failed to create payment", http.StatusInternalServerError)
		return
	}

	// Сохранение информации о платеже в БД
	newPayment := models.Payments{
		ChatID:  int(req.ChatID),
		OrderID: orderID,
		Amount:  req.Amount,
		Url:     paymentURL,
		Status:  "check",
		Type:    "aaio",
	}
	db.Create(&newPayment)
	log.Printf("Payment created: %+v", newPayment)
	response := map[string]string{
		"url":    paymentURL,
		"status": "success",
	}
	json.NewEncoder(w).Encode(response)
}

func createOrderID(chatID int64, timestamp int64) string {
	return fmt.Sprintf("order_%d_%d", chatID, timestamp)
}

func StartHTTPServer(db *gorm.DB) {
	// Существующие обработчики
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		handleWebhook(db, w, r)
	})
	http.HandleFunc("/create_payment", func(w http.ResponseWriter, r *http.Request) {
		HandleCreatePayment(w, r, db)
	})

	http.HandleFunc("/create_payment_aaio", func(w http.ResponseWriter, r *http.Request) {
		HandleCreatePaymentAAIO(w, r, db)
	})

	http.HandleFunc("/aaio_notification", func(w http.ResponseWriter, r *http.Request) {
		handleAAIONotification(db, w, r)
	})

	log.Printf("HTTP server started on %v", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
