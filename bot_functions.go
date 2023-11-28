package main

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

func WelcomeMessage(bot *tgbotapi.BotAPI, chatID int64) {
	messageText := "Добро пожаловать!"
	msg := tgbotapi.NewMessage(chatID, messageText)
	bot.Send(msg)
}

func SendSubscriptionMessage(bot *tgbotapi.BotAPI, chatID int64) {
	messageText := "Чтобы пользоваться ботом, вам нужно подписаться на каналы. После подписки заново напишите /start"
	msg := tgbotapi.NewMessage(chatID, messageText)

	// Добавляем вложение с ссылкой на канал
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Подписаться на канал", "https://t.me/botixaforcheck"),
		),
	)
	msg.ReplyMarkup = keyboard

	bot.Send(msg)
}

func SendPromotionMessage(bot *tgbotapi.BotAPI, chatID int64, db *gorm.DB) {
	messageText := "🤖Наш бот предназначен для продвижения ваших проектов и аккаунтов в социальных сетях.\n\n 🌟Здесь вы можете приобрести подписчиков, просмотры и комментарии."

	msg := tgbotapi.NewMessage(chatID, messageText)

	keyboard, err := СreatePromotionKeyboard(db)
	if err != nil {
		log.Println("Error creating promotion keyboard:", err)
		// В случае ошибки создания клавиатуры, просто отправим сообщение без нее
		if _, err := bot.Send(msg); err != nil {
			log.Println("Error sending promotion message:", err)
		}
		return
	}

	// Добавляем инлайн-кнопки к сообщению
	msg.ReplyMarkup = keyboard
	if _, err := bot.Send(msg); err != nil {
		log.Println("Error sending promotion message:", err)
	}
}

func СreatePromotionKeyboard(db *gorm.DB) (tgbotapi.InlineKeyboardMarkup, error) {
	categories, err := GetCategoriesFromDB(db)
	if err != nil {
		return tgbotapi.InlineKeyboardMarkup{}, err
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, category := range categories {
		// Добавляем кнопку для каждой категории
		button := tgbotapi.NewInlineKeyboardButtonData(category.Name, fmt.Sprintf("category:%s", category.ID))
		row := []tgbotapi.InlineKeyboardButton{button}
		rows = append(rows, row)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...), nil
}
