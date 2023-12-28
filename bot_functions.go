package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var currentPage = ""

func sendKeyboardAfterOrder(bot *tgbotapi.BotAPI, chatID int64) {
	messageText := "Заказ создан, ожидайте."
	msg := tgbotapi.NewMessage(chatID, messageText)
	balanceButton := tgbotapi.NewKeyboardButton("💰Баланс")
	ordersButton := tgbotapi.NewKeyboardButton("📝Мои заказы")
	makeOrderButton := tgbotapi.NewKeyboardButton("⭐️Сделать заказ")
	makeFavoriteButton := tgbotapi.NewKeyboardButton("❤️Избранное")
	quickReplyMarkup := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(balanceButton),
		tgbotapi.NewKeyboardButtonRow(ordersButton),
		tgbotapi.NewKeyboardButtonRow(makeOrderButton),
		tgbotapi.NewKeyboardButtonRow(makeFavoriteButton),
	)

	msg.ReplyMarkup = quickReplyMarkup
	bot.Send(msg)
}
func sendStandardKeyboard(bot *tgbotapi.BotAPI, chatID int64) {
	messageText := "Отменено"
	msg := tgbotapi.NewMessage(chatID, messageText)
	balanceButton := tgbotapi.NewKeyboardButton("💰Баланс")
	ordersButton := tgbotapi.NewKeyboardButton("📝Мои заказы")
	makeOrderButton := tgbotapi.NewKeyboardButton("⭐️Сделать заказ")
	makeFavoriteButton := tgbotapi.NewKeyboardButton("❤️Избранное")
	quickReplyMarkup := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(balanceButton),
		tgbotapi.NewKeyboardButtonRow(ordersButton),
		tgbotapi.NewKeyboardButtonRow(makeOrderButton),
		tgbotapi.NewKeyboardButtonRow(makeFavoriteButton),
	)

	msg.ReplyMarkup = quickReplyMarkup
	bot.Send(msg)
}

func sendStandardKeyboardAfterPayment(bot *tgbotapi.BotAPI, chatID int64) {
	messageText := "После оплаты проверьте баланс."
	msg := tgbotapi.NewMessage(chatID, messageText)
	balanceButton := tgbotapi.NewKeyboardButton("💰Баланс")
	ordersButton := tgbotapi.NewKeyboardButton("📝Мои заказы")
	makeOrderButton := tgbotapi.NewKeyboardButton("⭐️Сделать заказ")
	makeFavoriteButton := tgbotapi.NewKeyboardButton("❤️Избранное")
	quickReplyMarkup := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(balanceButton),
		tgbotapi.NewKeyboardButtonRow(ordersButton),
		tgbotapi.NewKeyboardButtonRow(makeOrderButton),
		tgbotapi.NewKeyboardButtonRow(makeFavoriteButton),
	)

	msg.ReplyMarkup = quickReplyMarkup
	bot.Send(msg)
}
func WelcomeMessage(bot *tgbotapi.BotAPI, chatID int64) {
	messageText := "Добро пожаловать!"
	msg := tgbotapi.NewMessage(chatID, messageText)
	balanceButton := tgbotapi.NewKeyboardButton("💰Баланс")
	ordersButton := tgbotapi.NewKeyboardButton("📝Мои заказы")
	makeOrderButton := tgbotapi.NewKeyboardButton("⭐️Сделать заказ")
	makeFavoriteButton := tgbotapi.NewKeyboardButton("❤️Избранное")
	quickReplyMarkup := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(balanceButton),
		tgbotapi.NewKeyboardButtonRow(ordersButton),
		tgbotapi.NewKeyboardButtonRow(makeOrderButton),
		tgbotapi.NewKeyboardButtonRow(makeFavoriteButton),
	)

	msg.ReplyMarkup = quickReplyMarkup
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

func FormatServiceInfo(service Services, subcategory Subcategory) string {
	return fmt.Sprintf(
		"ℹ️ Информация об услуге\n\n"+
			"🔢 ID услуги: %d\n"+
			"📝 Услга: %s\n\n"+
			"📝Категория:%s\n\n"+
			"💸 Цена за 1000: $%.*f\n\n"+
			"📉 Минимальное количество: %d\n"+
			"📈 Максимальное количество: %d",
		service.ID, service.Name, subcategory.Name, decimalPlaces, service.Rate, service.Min, service.Max)
}

// Функция для обработки нажатия кнопки "Баланс"
func handleBalanceCommand(bot *tgbotapi.BotAPI, userID int64, db *gorm.DB) {
	var userState UserState
	if err := db.Where("user_id = ?", userID).First(&userState).Error; err != nil {
		log.Printf("Error fetching user state: %v", err)
		return
	}

	balanceMsgText := fmt.Sprintf("🆔 Ваш ID: %d\n💵 Ваш баланс: $%.*f", userState.UserID, decimalPlaces, userState.Balance)
	msg := tgbotapi.NewMessage(userID, balanceMsgText)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰Пополнить баланс", "replenishBalance"),
		),
	)
	msg.ReplyMarkup = keyboard

	bot.Send(msg)
}

func handleOrdersCommand(bot *tgbotapi.BotAPI, chatID int64, db *gorm.DB) {
	var userOrders []UserOrders
	chatIDString := strconv.FormatInt(chatID, 10) // Преобразование chatID в строку
	result := db.Where("user_id = ?", chatIDString).Find(&userOrders)

	if result.Error != nil {
		log.Printf("Ошибка при получении заказов пользователя: %v", result.Error)
		bot.Send(tgbotapi.NewMessage(chatID, "Произошла ошибка при получении информации о ваших заказах."))
		return
	}

	if len(userOrders) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "Вы еще не совершали покупок."))
		return
	}

	messageText := "📝 Ваши заказы:\n\n"
	for _, order := range userOrders {
		status := translateOrderStatus(order.Status)
		messageText += fmt.Sprintf("Номер услуги: %s\nСсылка: %s\nКоличество: %d\nСтатус: %s\n\n",
			order.ServiceID, order.Link, order.Quantity, status)
	}

	msg := tgbotapi.NewMessage(chatID, messageText)
	bot.Send(msg)
}

// translateOrderStatus переводит статус заказа на русский язык
func translateOrderStatus(status string) string {
	switch status {
	case "PENDING":
		return "Ожидание"
	case "COMPLETED":
		return "Выполнен"
	case "IN_PROGRESS":
		return "В процессе"
	case "PARTIAL":
		return "Частично выполнен"
	case "CANCELED":
		return "Отменен"
	default:
		return "Неизвестный статус"
	}
}

func handleFavoritesCommand(bot *tgbotapi.BotAPI, db *gorm.DB, chatID int64) {
	favorites, err := GetUserFavorites(db, chatID)
	if err != nil || len(favorites) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "В избранном пока нет услуг."))
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, service := range favorites {
		button := tgbotapi.NewInlineKeyboardButtonData(service.Name, "serviceInfo:"+service.ServiceID)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(chatID, "Ваши избранные услуги:")
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}
