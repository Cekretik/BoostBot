package payment

import (
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

func handleReplenishCommand(bot *tgbotapi.BotAPI, chatID int64) {
	userPaymentStatus := updateUserStatus(chatID)
	userPaymentStatus.CurrentState = "awaitingPaymentSystem"

	msgText := ("Выберите платежную систему")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("СБП|RUB", "payok_SBP"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("RU Карта|RUB", "payok_RU"),
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

func handleCryptomusButton(bot *tgbotapi.BotAPI, chatID int64, db *gorm.DB) {
	userPaymentStatus := updateUserStatus(chatID)
	userPaymentStatus.CurrentState = "awaitingAmount"
	log.Printf("chuba %v", userPaymentStatus)
	userPaymentStatus.OrderID = createOrderID(chatID, time.Now().Unix())

	var user UserState
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

func handlePayOKButton(bot *tgbotapi.BotAPI, chatID int64, db *gorm.DB) {
	userPaymentStatus := updateUserStatus(chatID)
	userPaymentStatus.CurrentState = "awaitingAmountPayOK"
	paymentID := createPaymentID(chatID, time.Now().Unix())
	userPaymentStatus.OrderID = paymentID

	var user UserState
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
	userPaymentStatuses[chatID] = userPaymentStatus
}
