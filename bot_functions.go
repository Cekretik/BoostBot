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

func CreateQuickReplyMarkup() tgbotapi.ReplyKeyboardMarkup {
	balanceButton := tgbotapi.NewKeyboardButton("💳 Баланс")
	settingsButton := tgbotapi.NewKeyboardButton("⚙️Настройки")
	ordersButton := tgbotapi.NewKeyboardButton("📝Мои заказы")
	makeOrderButton := tgbotapi.NewKeyboardButton("✍️Сделать заказ")
	makeTechSupButton := tgbotapi.NewKeyboardButton("⛑ Помощь")
	makeReferralpButton := tgbotapi.NewKeyboardButton("🤝 Партнерам")
	makeProfileButton := tgbotapi.NewKeyboardButton("🧩Профиль")
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(balanceButton, ordersButton),
		tgbotapi.NewKeyboardButtonRow(makeOrderButton, makeTechSupButton),
		tgbotapi.NewKeyboardButtonRow(makeProfileButton, makeReferralpButton),
		tgbotapi.NewKeyboardButtonRow(settingsButton),
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

	totalPages, err := GetTotalPagesForCategory(db, itemsPerPage, categoryID)
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
	var tempRow []tgbotapi.InlineKeyboardButton // Временный массив для хранения кнопок

	categories, err := GetCategoriesFromDB(db)
	if err != nil {
		return tgbotapi.InlineKeyboardMarkup{}, err
	}

	for i, category := range categories {
		categoryNameWithEmoji := addEmojiToCategoryName(category.Name)
		categoryButton := tgbotapi.NewInlineKeyboardButtonData(categoryNameWithEmoji, fmt.Sprintf("category:%s", category.ID))

		tempRow = append(tempRow, categoryButton)

		// Добавляем строку в rows после каждой второй кнопки или если это последняя кнопка
		if (i+1)%2 == 0 || i == len(categories)-1 {
			rows = append(rows, tempRow)
			tempRow = []tgbotapi.InlineKeyboardButton{} // Очищаем временный массив
		}
	}

	// Добавляем кнопку "Избранное" отдельно внизу
	favoriteButton := tgbotapi.NewInlineKeyboardButtonData("❤️\u200d🔥Избранное", "profile:favorites")
	rows = append(rows, []tgbotapi.InlineKeyboardButton{favoriteButton})

	return tgbotapi.NewInlineKeyboardMarkup(rows...), nil
}

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

func FormatServiceInfo(service Services, subcategory Subcategory, increasePercent float64, userCurrency string, currencyRate float64) string {
	increasedRate := service.Rate + service.Rate*(increasePercent/100)

	if userCurrency == "RUB" {
		increasedRate = convertAmount(increasedRate, currencyRate, true)
		currencySymbol := "₽"
		return fmt.Sprintf(
			"ℹ️ Информация об услуге\n\n"+
				"🔢 ID услуги: %d\n"+
				"📝 Услуга: %s\n\n"+
				"📝 Категория: %s\n\n"+
				"💸 Цена за 1000: %s%.*f\n\n"+
				"📉 Минимальное количество: %d\n"+
				"📈 Максимальное количество: %d",
			service.ID, service.Name, subcategory.Name, currencySymbol, decimalPlaces, increasedRate, service.Min, service.Max)
	} else {
		currencySymbol := "$"
		return fmt.Sprintf(
			"ℹ️ Информация об услуге\n\n"+
				"🔢 ID услуги: %d\n"+
				"📝 Услуга: %s\n\n"+
				"📝 Категория: %s\n\n"+
				"💸 Цена за 1000: %s%.*f\n\n"+
				"📉 Минимальное количество: %d\n"+
				"📈 Максимальное количество: %d",
			service.ID, service.Name, subcategory.Name, currencySymbol, decimalPlaces, increasedRate, service.Min, service.Max)
	}
}

// Функция для обработки нажатия кнопки "Баланс"
func handleBalanceCommand(bot *tgbotapi.BotAPI, userID int64, db *gorm.DB) {
	var userState UserState
	if err := db.Where("user_id = ?", userID).First(&userState).Error; err != nil {
		log.Printf("Error fetching user state: %v", err)
		return
	}

	// Получаем текущий курс обмена
	rate, err := getCurrencyRate()
	if err != nil {
		log.Printf("Error getting currency rate: %v", err)
		return
	}

	balance := userState.Balance
	var balanceMsgText string

	if userState.Currency == "RUB" {
		balance = convertAmount(balance, rate, true)
		balanceMsgText = fmt.Sprintf("💳 Ваш баланс: ₽%.*f", decimalPlaces, balance)
	} else {
		balanceMsgText = fmt.Sprintf("💳 Ваш баланс: $%.*f", decimalPlaces, balance)
	}

	msg := tgbotapi.NewMessage(userID, balanceMsgText)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚡️Пополнить баланс", "replenishBalance"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎁Промокод", "promo"),
		),
	)
	msg.ReplyMarkup = keyboard

	bot.Send(msg)
}

func handlePromoCommand(bot *tgbotapi.BotAPI, chatID int64, db *gorm.DB) {
	messageText := "✍️Введите ваш промокод:"
	cancelKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, messageText)
	msg.ReplyMarkup = cancelKeyboard
	bot.Send(msg)
	sendStandardKeyboard(bot, chatID)
}

func handleProfileCommand(bot *tgbotapi.BotAPI, chatID int64, db *gorm.DB) {
	var userState UserState
	if err := db.Where("user_id = ?", chatID).First(&userState).Error; err != nil {
		log.Printf("Error fetching user state: %v", err)
		return
	}
	rate, err := getCurrencyRate()
	if err != nil {
		log.Printf("Error getting currency rate: %v", err)
		return
	}
	balance := userState.Balance
	var messageText string
	if userState.Currency == "RUB" {
		balance = convertAmount(balance, rate, true)
		messageText = fmt.Sprintf("🤵‍♂️ Пользователь:%v\n 🔎 ID:%v\n 💳 Ваш баланс:₽%.*f", userState.UserName, userState.UserID, decimalPlaces, balance)
	} else {
		messageText = fmt.Sprintf("🤵‍♂️ Пользователь:%v\n 🔎 ID:%v\n 💳 Ваш баланс:$%.*f", userState.UserName, userState.UserID, decimalPlaces, balance)
	}
	msg := tgbotapi.NewMessage(chatID, messageText)
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

func GenerateReferralLink(chatID int64) string {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	botLink := os.Getenv("BOT_LINK")
	return fmt.Sprintf(botLink+"?start=%d", chatID)
}

func ShowReferralStats(bot *tgbotapi.BotAPI, db *gorm.DB, userID int64) {
	var referrals []Referral
	db.Where("referrer_id = ?", userID).Find(&referrals)
	count := len(referrals)

	var totalEarned float64
	for _, referral := range referrals {
		totalEarned += referral.AmountEarned
	}

	msgText := fmt.Sprintf("🏂Приглашено человек: %d\n💸Заработано с ваших рефералов: $%.2f\n\n 🔘Приглашайте друзей и партнёров и получайте 10%% на баланс с каждой покупки. \n\n ✨Ваша партнёрская ссылка: %s", count, totalEarned, GenerateReferralLink(userID))

	msg := tgbotapi.NewMessage(userID, msgText)
	bot.Send(msg)
}

func convertAmount(amount float64, rate float64, toRUB bool) float64 {
	if toRUB {
		return amount * rate
	} else {
		return amount / rate
	}
}

func handleChangeCurrency(bot *tgbotapi.BotAPI, userID int64, db *gorm.DB, toRUB bool) {
	var user UserState
	err := db.Where("user_id = ?", userID).First(&user).Error
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if toRUB {
		user.Currency = "RUB"
	} else {
		user.Currency = "USD"
	}

	err = db.Save(&user).Error
	if err != nil {
		log.Printf("Error saving user: %v", err)
		return
	}

	msgText := "Валюта изменена на "
	if toRUB {
		msgText += "рубли."
	} else {
		msgText += "доллары."
	}
	msg := tgbotapi.NewMessage(userID, msgText)
	bot.Send(msg)
}
