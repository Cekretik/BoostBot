package main

import (
	"fmt"
	"log"
	"strings"

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

	keyboard, err := CreatePromotionKeyboard(db, false, "")
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

func CreatePromotionKeyboard(db *gorm.DB, showSubcategories bool, categoryID string) (tgbotapi.InlineKeyboardMarkup, error) {
	var rows [][]tgbotapi.InlineKeyboardButton

	if !showSubcategories {
		// Выводим категории
		categories, err := GetCategoriesFromDB(db)
		if err != nil {
			return tgbotapi.InlineKeyboardMarkup{}, err
		}

		for _, category := range categories {
			button := tgbotapi.NewInlineKeyboardButtonData(category.Name, fmt.Sprintf("category:%s", category.ID))
			row := []tgbotapi.InlineKeyboardButton{button}
			rows = append(rows, row)
		}
	} else {
		// Выводим подкатегории
		subcategories, err := GetSubcategoriesByCategoryID(db, categoryID)
		if err != nil {
			return tgbotapi.InlineKeyboardMarkup{}, err
		}

		for _, subcategory := range subcategories {
			button := tgbotapi.NewInlineKeyboardButtonData(subcategory.Name, fmt.Sprintf("subcategory:%s", subcategory.ID))
			row := []tgbotapi.InlineKeyboardButton{button}
			rows = append(rows, row)
		}
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...), nil
}

func HandleCallbackQuery(bot *tgbotapi.BotAPI, db *gorm.DB, callbackQuery *tgbotapi.CallbackQuery) {
	// Проверка и обработка данных callbackQuery
	if strings.HasPrefix(callbackQuery.Data, "category:") {
		categoryID := strings.TrimPrefix(callbackQuery.Data, "category:")

		// Создание новой клавиатуры для подкатегорий
		keyboard, err := CreatePromotionKeyboard(db, true, categoryID)
		if err != nil {
			log.Println("Error creating promotion keyboard:", err)
			return
		}

		// Обновление сообщения с новой клавиатурой
		editMsg := tgbotapi.NewEditMessageReplyMarkup(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, keyboard)
		bot.Send(editMsg)
	}
}

func GetSubcategoriesByCategoryID(db *gorm.DB, categoryID string) ([]Subcategory, error) {
	var subcategories []Subcategory
	if err := db.Where("category_id = ?", categoryID).Find(&subcategories).Error; err != nil {
		return nil, err
	}
	return subcategories, nil
}
