package main

import (
	"fmt"
	"log"

	"github.com/Cekretik/BoostBot/models"
	tgbotapi "github.com/Cekretik/telegram-bot-api-master"
	"gorm.io/gorm"
)

type BotStatus struct {
	ChatID       int64
	CurrentState string
}

var BotStatuses map[int64]*BotStatus = make(map[int64]*BotStatus)

var NewTokensChannel chan string = make(chan string, 100)

func CreateQuickReplyMarkup() tgbotapi.ReplyKeyboardMarkup {
	MenuButton := tgbotapi.NewKeyboardButton("Меню")
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(MenuButton),
	)
}

func WelcomeMessage(bot *tgbotapi.BotAPI, chatID int64) {
	replyKeyboard := CreateQuickReplyMarkup()
	replyMsg := tgbotapi.NewMessage(chatID, "👋Добро пожаловать!")
	replyMsg.ReplyMarkup = replyKeyboard
	bot.Send(replyMsg)
}

func SendMenuButton(bot *tgbotapi.BotAPI, chatID int64, db *gorm.DB) {
	var botOwners models.BotOwners
	err := db.Where("user_id = ?", chatID).First(&botOwners).Error
	if err != nil {
		log.Println("Error getting user state:", err)
		return
	}

	greetingMessage := fmt.Sprintf("👋Привет %v! Я бот-менеджер, который поможет тебе создать своего бота по накрутке", botOwners.UserName)
	messageText := greetingMessage
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💼Боты", "bots"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, messageText)
	msg.ReplyMarkup = inlineKeyboard
	bot.Send(msg)

}

func HandleBackButton(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB) {
	chatID := callbackQuery.Message.Chat.ID
	messageID := callbackQuery.Message.MessageID

	var botOwners models.BotOwners
	err := db.Where("user_id = ?", chatID).First(&botOwners).Error
	if err != nil {
		log.Println("Error getting user state:", err)
		return
	}

	greetingMessage := fmt.Sprintf("👋Привет %v! Я бот-менеджер, который поможет тебе создать своего бота по накрутке", botOwners.UserName)
	messageText := greetingMessage
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💼Боты", "bots"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, messageText)
	editMsg.ReplyMarkup = &inlineKeyboard
	bot.Send(editMsg)
}

func HandleBotStart(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery, db *gorm.DB) {
	chatID := callbackQuery.Message.Chat.ID
	messageID := callbackQuery.Message.MessageID

	var botCount int64
	err := db.Model(&models.BotOwners{}).Where("user_id = ? AND token != ''", chatID).Count(&botCount).Error
	if err != nil {
		log.Printf("Ошибка при получении количества ботов пользователя: %v", err)
		return
	}

	messageText := fmt.Sprintf("👋 Тут ты можешь добавлять, настраивать и удалять своих ботов! \n\nТекущее количество твоих ботов: %d из 10.", botCount)

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🆕Создать бота", "create_bot"),
			tgbotapi.NewInlineKeyboardButtonData("⬅️Назад", "backtomenu"),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, messageText)
	editMsg.ReplyMarkup = &inlineKeyboard
	bot.Send(editMsg)
}
func InitiateTokenInput(bot *tgbotapi.BotAPI, chatID int64) {
	BotStatuses[chatID] = &BotStatus{
		ChatID:       chatID,
		CurrentState: "awaiting_token",
	}
	msg := tgbotapi.NewMessage(chatID, "❗️Ответьте на это сообщение токеном бота")
	bot.Send(msg)
}

func HandleTokenInput(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB) {
	chatID := update.Message.Chat.ID
	token := update.Message.Text
	userName := update.Message.From.UserName

	if status, exists := BotStatuses[chatID]; exists && status.CurrentState == "awaiting_token" {
		var botCount int64
		db.Model(&models.BotOwners{}).Where("user_id = ?", chatID).Count(&botCount)

		if botCount >= 10 {
			msg := tgbotapi.NewMessage(chatID, "Превышен лимит количества ботов. Максимум можно иметь 10 ботов.")
			bot.Send(msg)
			return
		}

		tempBot, err := tgbotapi.NewBotAPI(token)
		if err != nil {
			log.Printf("Ошибка при инициализации бота с токеном %s: %v", token, err)
			msg := tgbotapi.NewMessage(chatID, "Неверный токен бота. Пожалуйста, проверьте и попробуйте снова.")
			bot.Send(msg)
			return
		}

		botInfo, err := tempBot.GetMe()
		if err != nil {
			log.Printf("Ошибка при получении информации о боте: %v", err)
			msg := tgbotapi.NewMessage(chatID, "Не удалось получить информацию о боте. Пожалуйста, проверьте токен.")
			bot.Send(msg)
			return
		}

		userBotStatus := models.BotOwners{
			UserID:   chatID,
			UserName: userName,
			Token:    token,
			Running:  true,
			BotName:  botInfo.UserName,
			Balance:  0,
		}

		if err := db.Create(&userBotStatus).Error; err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка при сохранении токена: %v", err))
			bot.Send(msg)
			return
		}

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Бот @%s был успешно добавлен и включен.", botInfo.UserName))
		bot.Send(msg)
		go func() {
			NewTokensChannel <- token
		}()
		delete(BotStatuses, chatID)
	}
}
