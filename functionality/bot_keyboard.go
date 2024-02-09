package functionality

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	tgbotapi "github.com/Cekretik/telegram-bot-api-master"
)

var currentPage = ""

func addEmojiToCategoryName(categoryName string) string {
	switch categoryName {
	case "Telegram":
		return "💎 Telegram"
	case "YouTube":
		return "🎯 YouTube"
	case "Instagram":
		return "📸 Instagram"
	case "TikTok":
		return "🎭 TikTok"
	case "Twitter":
		return "🐦 Twitter"
	default:
		return categoryName
	}
}

func CreateQuickReplyMarkup() tgbotapi.ReplyKeyboardMarkup {
	balanceButton := tgbotapi.NewKeyboardButton("💳 Баланс")
	makeOrderButton := tgbotapi.NewKeyboardButton("✍️Сделать заказ")
	makeReferralpButton := tgbotapi.NewKeyboardButton("🤝 Партнерам")
	makeProfileButton := tgbotapi.NewKeyboardButton("🧩Профиль")
	makeSiteButton := tgbotapi.NewKeyboardButton("⚡️Сайт (-55%)")
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(balanceButton, makeOrderButton),
		tgbotapi.NewKeyboardButtonRow(makeReferralpButton, makeProfileButton),
		tgbotapi.NewKeyboardButtonRow(makeSiteButton),
	)
}

func sendKeyboardAfterOrder(bot *tgbotapi.BotAPI, chatID int64) {
	messageText := "Заказ создан, ожидайте."
	msg := tgbotapi.NewMessage(chatID, messageText)
	quickReplyMarkup := CreateQuickReplyMarkup()
	msg.ReplyMarkup = quickReplyMarkup
	bot.Send(msg)
}
func sendStandardKeyboard(bot *tgbotapi.BotAPI, chatID int64) {
	messageText := "Отменено"
	msg := tgbotapi.NewMessage(chatID, messageText)
	quickReplyMarkup := CreateQuickReplyMarkup()
	msg.ReplyMarkup = quickReplyMarkup
	bot.Send(msg)
}

func sendStandardKeyboardAfterPayment(bot *tgbotapi.BotAPI, chatID int64) {
	messageText := "После оплаты проверьте баланс."
	msg := tgbotapi.NewMessage(chatID, messageText)
	quickReplyMarkup := CreateQuickReplyMarkup()
	msg.ReplyMarkup = quickReplyMarkup
	bot.Send(msg)
}
func techSupMessage(bot *tgbotapi.BotAPI, chatID int64) {
	channelLink := "https://t.me/DARRINAN00"
	messageText := "Техническая поддержка: "
	msg := tgbotapi.NewMessage(chatID, messageText)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Написать", channelLink),
		),
	)
	msg.ReplyMarkup = keyboard

	bot.Send(msg)
}

func sendSettingsKeyboard(bot *tgbotapi.BotAPI, chatID int64) {
	messageText := "⚙️Сменить валюту на:"
	msg := tgbotapi.NewMessage(chatID, messageText)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("RUB", "changeCurrencyToRUB"),
			tgbotapi.NewInlineKeyboardButtonData("USD", "changeCurrencyToUSD"),
		),
	)

	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

func SendSubscriptionMessage(bot *tgbotapi.BotAPI, chatID int64) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	channelLink := os.Getenv("CHANNEL_LINK")
	messageText := "Чтобы пользоваться ботом, вам нужно подписаться на каналы. После подписки заново напишите /start"
	msg := tgbotapi.NewMessage(chatID, messageText)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Подписаться на канал", channelLink),
		),
	)
	msg.ReplyMarkup = keyboard

	bot.Send(msg)
}

func SendSiteMessage(bot *tgbotapi.BotAPI, chatID int64) {
	messageText := "⚡️На нашем сайте [StageSMM](https://stagesmm.com/) вы можете накрутить все, что есть в боте, в более удобном формате.\n\n☝️Главными плюсами сайта являются:\n\n🔸 Цены ПО ВСЕМ категориям на 55% дешевле цен бота \n🔸 ОГРОМНОЕ количество услуг\n🔸 Интуитивно понятный интерфейс\n🔸 Легкие пополнения с кучей способов оплат\n\n♦️И наконец промокод на пополнение `" + "STAGE10" + "` .Используя его вы сможете пополнять баланс на 10% больше оплаченного♦️"
	msg := tgbotapi.NewMessage(chatID, messageText)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("⚡️StageSMM", "https://stagesmm.com/"),
		),
	)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true
	bot.Send(msg)
}

func SendPromotionMessage(bot *tgbotapi.BotAPI, chatID int64, db *gorm.DB) {
	var userState UserState
	err := db.Where("user_id = ?", chatID).First(&userState).Error
	if err != nil {
		log.Println("Error getting user state:", err)
		return
	}
	greetingText := fmt.Sprintf("👋 Привет, %v! Я - StageSMM_Bot, ваш верный помощник для продвижения проектов и аккаунтов в социальных сетях. 🚀 Продвигай свои проекты с нашей помощью!", userState.UserName)
	greetingMsg := tgbotapi.NewMessage(chatID, greetingText)
	quickReplyMarkup := CreateQuickReplyMarkup()
	greetingMsg.ReplyMarkup = quickReplyMarkup
	if _, err := bot.Send(greetingMsg); err != nil {
		log.Println("Error sending greeting message:", err)
		return
	}
	categoryID := ""

	totalPages, err := database.GetTotalPagesForCategory(db, callbacks.itemsPerPage, categoryID)
	if err != nil {
		log.Println("Error getting total pages:", err)
		return
	}

	categoryKeyboard, err := CreateCategoryKeyboard(db)
	if err != nil {
		log.Println("Error creating category keyboard:", err)
		return
	}

	categoryMsg := tgbotapi.NewMessage(chatID, "✨ Выберите социальную сеть для продвижения:")
	categoryMsg.ReplyMarkup = categoryKeyboard
	if _, err := bot.Send(categoryMsg); err != nil {
		log.Println("Error sending category message:", err)
		return
	}

	subcategories, err := database.GetSubcategoriesByCategoryID(db, categoryID)
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

	categoryNames := []string{"Telegram", "YouTube", "Instagram", "TikTok", "Twitter"}

	categories, err := GetCategoriesFromDB(db)
	if err != nil {
		return tgbotapi.InlineKeyboardMarkup{}, err
	}

	categoryMap := make(map[string]Category)
	for _, category := range categories {
		categoryMap[category.Name] = category
	}

	for i, name := range categoryNames {
		if category, ok := categoryMap[name]; ok {
			categoryNameWithEmoji := addEmojiToCategoryName(category.Name)
			categoryButton := tgbotapi.NewInlineKeyboardButtonData(categoryNameWithEmoji, fmt.Sprintf("category:%s", category.ID))

			if i == 0 || i%2 == 1 {
				rows = append(rows, []tgbotapi.InlineKeyboardButton{categoryButton})
			} else {
				lastRowIndex := len(rows) - 1
				rows[lastRowIndex] = append(rows[lastRowIndex], categoryButton)
			}
		}
	}

	// Добавляем кнопку "Избранное" отдельно внизу
	favoriteButton := tgbotapi.NewInlineKeyboardButtonData("❤️\u200d🔥Избранное", "profile:favorites")
	rows = append(rows, []tgbotapi.InlineKeyboardButton{favoriteButton})

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
