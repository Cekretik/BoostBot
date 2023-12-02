package main

import (
	"fmt"
	"log"
	"strconv"

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

	itemsPerPage := 10

	// Используем ID категории (пустая строка, если необходимы все подкатегории)
	categoryID := ""

	totalPages, err := GetTotalPagesForCategory(db, itemsPerPage, categoryID)
	if err != nil {
		log.Println("Error getting total pages:", err)
		return
	}

	// Изначально устанавливаем текущую страницу в 1
	currentPage := "1"

	keyboard, err := CreatePromotionKeyboard(db, false, categoryID, currentPage, strconv.Itoa(totalPages))
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

func CreatePromotionKeyboard(db *gorm.DB, showSubcategories bool, categoryID, currentPage, totalPages string) (tgbotapi.InlineKeyboardMarkup, error) {
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

		// Разбиваем подкатегории на страницы

		// Разбиваем подкатегории на страницы
		itemsPerPage := 10
		startIdx, endIdx := calculatePageRange(len(subcategories), itemsPerPage, currentPage)

		for i := startIdx; i < endIdx; i++ {
			subcategory := subcategories[i]
			button := tgbotapi.NewInlineKeyboardButtonData(subcategory.Name, fmt.Sprintf("subcategory:%s", subcategory.ID))
			row := []tgbotapi.InlineKeyboardButton{button}
			rows = append(rows, row)
		}

		// Добавляем кнопки "вперед" и "назад" и информацию о текущей странице
		paginationRow := createPaginationRow(categoryID, currentPage, totalPages)
		rows = append(rows, paginationRow)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...), nil
}
