package functionality

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

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
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝Мои заказы", "allorders"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️Настройки", "settings"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⛑ Помощь", "techsup"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, messageText)
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

func handleOrdersCommand(bot *tgbotapi.BotAPI, chatID int64, db *gorm.DB) {
	var userOrders []UserOrders
	chatIDString := strconv.FormatInt(chatID, 10)
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

func handlePromoCommand(bot *tgbotapi.BotAPI, chatID int64, db *gorm.DB) {
	messageText := "✍️Введите ваш промокод:"
	cancelKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	userPromoStatuses[chatID] = &UserPromoStatus{
		ChatID:     chatID,
		PromoState: "awaitingPromoCode",
	}
	msg := tgbotapi.NewMessage(chatID, messageText)
	msg.ReplyMarkup = cancelKeyboard
	bot.Send(msg)

}

func processPromoCodeInput(bot *tgbotapi.BotAPI, chatID int64, promoCode string, db *gorm.DB) {
	if promoCode == "Отмена" {
		sendStandardKeyboard(bot, chatID)
		return
	}

	var promo PromoCode
	if err := db.Where("code = ?", promoCode).First(&promo).Error; err != nil {
		msg := tgbotapi.NewMessage(chatID, "Промокод не найден.")
		msg.ReplyMarkup = CreateQuickReplyMarkup()
		bot.Send(msg)
		return
	}
	if promo.Activations >= promo.MaxActivations {
		msg := tgbotapi.NewMessage(chatID, "Этот промокод уже использован максимальное количество раз.")
		msg.ReplyMarkup = CreateQuickReplyMarkup()
		bot.Send(msg)
		return
	}

	var usedPromo UsedPromoCode
	if err := db.Where("user_id = ? AND promo_code = ?", chatID, promoCode).First(&usedPromo).Error; err == nil {
		msg := tgbotapi.NewMessage(chatID, "Вы уже использовали этот промокод.")
		msg.ReplyMarkup = CreateQuickReplyMarkup()
		bot.Send(msg)
		return
	}
	rate, err := getCurrencyRate()
	if err != nil {
		log.Printf("Error getting currency rate: %v", err)
		return
	}
	bonusInRubles := promo.Discount / rate
	switch promo.Type {
	case "fixed":
		UpdateUserBalance(db, chatID, bonusInRubles)
		congratulationMessage := fmt.Sprintf("🎁 Поздравляем, Вы активировали промокод!\n\n🌟 Ваш баланс пополнен на %.2fр", promo.Discount)
		bot.Send(tgbotapi.NewMessage(chatID, congratulationMessage))
	}
	newUsedPromo := UsedPromoCode{
		UserID:    chatID,
		PromoCode: promoCode,
		Used:      true,
	}
	db.Create(&newUsedPromo)

	promo.Activations++
	db.Save(&promo)

	msg := tgbotapi.NewMessage(chatID, "Промокод успешно применен.")
	msg.ReplyMarkup = CreateQuickReplyMarkup()
	bot.Send(msg)
}

func handleCreatePromoCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB) {
	if !isAdmin(bot, int64(update.Message.From.ID)) {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "У вас нет прав доступа к этой команде."))
		return
	}

	args := strings.Split(update.Message.Text, " ")

	if len(args) != 4 {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный формат. Используйте: /createpromo [название] [скидка] [максимальное количество использований]"))
		return
	}

	promoName := args[1]
	discount, err := strconv.ParseFloat(args[2], 64)
	if err != nil || discount <= 0 {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный формат скидки."))
		return
	}

	maxActivations, err := strconv.ParseInt(args[3], 10, 64)
	if err != nil || maxActivations <= 0 {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный формат количества использований."))
		return
	}

	promo := PromoCode{
		Code:           promoName,
		Discount:       discount,
		MaxActivations: maxActivations,
		Activations:    0,
		Type:           "discount",
	}

	if err := db.Create(&promo).Error; err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Промокод с таким названием уже существует."))
		return
	}

	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Промокод создан: %s", promo.Code)))
}

func handleCreateUrlCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	botLink := os.Getenv("BOT_LINK")
	if !isAdmin(bot, int64(update.Message.From.ID)) {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "У вас не достаточно прав"))
		return
	}

	args := strings.Split(update.Message.Text, " ")
	if len(args) != 4 {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный формат. Используйте: /createurl [название] [сумма] [кол-во переходов]"))
		return
	}

	linkName, amountStr, maxClicksStr := args[1], args[2], args[3]
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка в формате суммы."))
		return
	}
	maxClicks, err := strconv.ParseInt(maxClicksStr, 10, 64)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка в формате количества переходов."))
		return
	}

	linkCode := GenerateSpecialLink(linkName)
	var existingPromo PromoCode
	if db.Where("code = ?", linkCode).First(&existingPromo).Error == nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ссылка с таким названием уже была создана ранее."))
		return
	}
	promo := PromoCode{
		Code:           linkCode,
		Discount:       amount,
		MaxActivations: maxClicks,
		Activations:    0,
		Type:           "fixed",
	}
	db.Create(&promo)
	specialLink := fmt.Sprintf(botLink+"?start=%s", linkCode)
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ссылка создана: %s", specialLink)))
}

func handleBroadcastCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB) {
	if !isAdmin(bot, int64(update.Message.From.ID)) {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "У вас нет прав для выполнения этой команды."))
		return
	}

	parts := strings.SplitN(update.Message.Text, " ", 2)
	if len(parts) < 2 || len(parts[1]) == 0 {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Пожалуйста, укажите сообщение для рассылки."))
		return
	}

	// Используйте только текст после команды /broadcast
	message := parts[1]

	var entities []Entity
	if update.Message.Entities != nil && len(*update.Message.Entities) > 0 {
		shiftedEntities := make([]tgbotapi.MessageEntity, len(*update.Message.Entities))
		for i, entity := range *update.Message.Entities {
			if entity.Offset >= len(parts[0]) {
				shiftedEntities[i] = entity
				shiftedEntities[i].Offset -= len(parts[0]) + 1
			}
		}
		entities = convertEntities(shiftedEntities)
	}

	formattedMessage, err := formatBroadcastMessage(message, entities)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка форматирования сообщения: "+err.Error()))
		return
	}

	go broadcastMessage(bot, db, formattedMessage)
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Рассылка началась."))
}
