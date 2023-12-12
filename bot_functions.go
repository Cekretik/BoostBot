package main

import (
	"fmt"
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

var currentPage = ""

func WelcomeMessage(bot *tgbotapi.BotAPI, chatID int64) {
	messageText := "Добро пожаловать!"
	msg := tgbotapi.NewMessage(chatID, messageText)
	balanceButton := tgbotapi.NewKeyboardButton("💰Баланс")
	quickReplyMarkup := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(balanceButton),
	)

	msg.ReplyMarkup = quickReplyMarkup
	bot.Send(msg)
}

func SendSubscriptionMessage(bot *tgbotapi.BotAPI, chatID int64) {
	messageText := "Чтобы пользоваться ботом, вам нужно подписаться на каналы. После подписки заново напишите /start"
	msg := tgbotapi.NewMessage(chatID, messageText)

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

	categoryID := ""

	totalPages, err := GetTotalPagesForCategory(db, itemsPerPage, categoryID)
	if err != nil {
		log.Println("Error getting total pages:", err)
		return
	}

	categoryKeyboard, err := CreateCategoryKeyboard(db)
	if err != nil {
		log.Println("Error creating category keyboard:", err)

		if _, err := bot.Send(msg); err != nil {
			log.Println("Error sending promotion message:", err)
		}
		return
	}

	msg.ReplyMarkup = categoryKeyboard
	if _, err := bot.Send(msg); err != nil {
		log.Println("Error sending promotion message:", err)
		return
	}

	subcategories, err := GetSubcategoriesByCategoryID(db, categoryID)
	if err != nil {
		log.Println("Error getting subcategories:", err)
		return
	}

	for _, subcategory := range subcategories {
		subcategoryMsg := tgbotapi.NewMessage(chatID, subcategory.Name)

		subcategoryKeyboard, err := CreateSubcategoryKeyboard(db, subcategory.ID, currentPage, strconv.Itoa(totalPages))
		if err != nil {
			log.Println("Error creating subcategory keyboard:", err)
			continue
		}

		subcategoryMsg.ReplyMarkup = subcategoryKeyboard
		if _, err := bot.Send(subcategoryMsg); err != nil {
			log.Println("Error sending subcategory message:", err)
		}
	}

	subcategoryID := ""
	totalServicePages, err := GetTotalPagesForService(db, itemsPerPage, categoryID)
	if err != nil {
		log.Println("Error getting total pages:", err)
		return
	}

	services, err := GetServicesBySubcategoryID(db, subcategoryID)
	if err != nil {
		log.Println("Error getting services:", err)
		return
	}

	for _, service := range services {
		serviceMsg := tgbotapi.NewMessage(chatID, service.Name)

		serviceKeyboard, err := CreateServiceKeyboard(db, service.ServiceID, currentPage, strconv.Itoa(totalServicePages))
		if err != nil {
			log.Println("Error creating service keyboard:", err)
			continue
		}

		serviceMsg.ReplyMarkup = serviceKeyboard
		if _, err := bot.Send(serviceMsg); err != nil {
			log.Println("Error sending service message:", err)
		}
	}
}

func CreateCategoryKeyboard(db *gorm.DB) (tgbotapi.InlineKeyboardMarkup, error) {
	var rows [][]tgbotapi.InlineKeyboardButton

	categories, err := GetCategoriesFromDB(db)
	if err != nil {
		return tgbotapi.InlineKeyboardMarkup{}, err
	}

	for _, category := range categories {
		button := tgbotapi.NewInlineKeyboardButtonData(category.Name, fmt.Sprintf("category:%s", category.ID))
		row := []tgbotapi.InlineKeyboardButton{button}
		rows = append(rows, row)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...), nil
}

func CreateSubcategoryKeyboard(db *gorm.DB, categoryID, currentPage, totalPages string) (tgbotapi.InlineKeyboardMarkup, error) {
	var rows [][]tgbotapi.InlineKeyboardButton

	subcategories, err := GetSubcategoriesByCategoryID(db, categoryID)
	if err != nil {
		return tgbotapi.InlineKeyboardMarkup{}, err
	}

	startIdx, endIdx := calculatePageRange(len(subcategories), itemsPerPage, currentPage)

	for i := startIdx; i < endIdx; i++ {
		subcategory := subcategories[i]
		button := tgbotapi.NewInlineKeyboardButtonData(subcategory.Name, fmt.Sprintf("subcategory:%s", subcategory.ID))
		row := []tgbotapi.InlineKeyboardButton{button}
		rows = append(rows, row)
	}

	totalPagesInt, err := strconv.Atoi(totalPages)
	if err != nil {
		return tgbotapi.InlineKeyboardMarkup{}, err
	}

	currentPageInt, err := strconv.Atoi(currentPage)
	if err != nil {
		return tgbotapi.InlineKeyboardMarkup{}, err
	}

	paginationRow := createPaginationRow(categoryID, currentPageInt, totalPagesInt)
	rows = append(rows, paginationRow)

	return tgbotapi.NewInlineKeyboardMarkup(rows...), nil
}

func CreateServiceKeyboard(db *gorm.DB, subcategoryID, currentPage, totalServicePages string) (tgbotapi.InlineKeyboardMarkup, error) {
	var rows [][]tgbotapi.InlineKeyboardButton

	services, err := GetServicesBySubcategoryID(db, subcategoryID)
	if err != nil {
		return tgbotapi.InlineKeyboardMarkup{}, err
	}

	startIdx, endIdx := calculatePageRange(len(services), itemsPerPage, currentPage)

	for i := startIdx; i < endIdx; i++ {
		service := services[i]

		button := tgbotapi.NewInlineKeyboardButtonData(service.Name, fmt.Sprintf("serviceInfo:%s", service.ServiceID))
		row := []tgbotapi.InlineKeyboardButton{button}
		rows = append(rows, row)
	}

	totalServicePagesInt, err := strconv.Atoi(totalServicePages)
	if err != nil {
		return tgbotapi.InlineKeyboardMarkup{}, err
	}

	currentPageInt, err := strconv.Atoi(currentPage)
	if err != nil {
		return tgbotapi.InlineKeyboardMarkup{}, err
	}
	backToSubcategoriesButton := tgbotapi.NewInlineKeyboardButtonData("🔙 Вернуться к категориям", fmt.Sprintf("backToSubcategories:%s", subcategoryID))
	rows = append(rows, []tgbotapi.InlineKeyboardButton{backToSubcategoriesButton})
	paginationRow := createServicePaginationRow(subcategoryID, currentPageInt, totalServicePagesInt)
	rows = append(rows, paginationRow)

	return tgbotapi.NewInlineKeyboardMarkup(rows...), nil
}

func FormatServiceInfo(service Service, subcategory Subcategory) string {
	return fmt.Sprintf(
		"ℹ️ Информация об услуге\n\n"+
			"🔢 ID услуги: %s\n"+
			"📝 Услга: %s\n\n"+
			"📝Категория:%s\n\n"+
			"💸 Цена за 1000: $%.2f\n\n"+
			"📉 Минимальное количество: %d\n"+
			"📈 Максимальное количество: %d",
		service.ServiceID, service.Name, subcategory.Name, service.Rate, service.Min, service.Max)
}

// Функция для обработки нажатия кнопки "Баланс"
func handleBalanceCommand(bot *tgbotapi.BotAPI, userID int64, db *gorm.DB) {
	var userState UserState
	if err := db.Where("user_id = ?", userID).First(&userState).Error; err != nil {
		log.Printf("Error fetching user state: %v", err)
		return
	}

	balanceMsgText := fmt.Sprintf("🆔 Ваш ID: %d\n💵 Ваш баланс: $%.2f", userState.UserID, userState.Balance)
	msg := tgbotapi.NewMessage(userID, balanceMsgText)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰Пополнить баланс", "replenishBalance"),
		),
	)
	msg.ReplyMarkup = keyboard

	bot.Send(msg)
}
