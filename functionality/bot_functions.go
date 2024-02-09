package functionality

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/Cekretik/BoostBot/api"
	"github.com/Cekretik/BoostBot/cmd"
	"github.com/Cekretik/BoostBot/database"
	"github.com/Cekretik/BoostBot/models"
	tgbotapi "github.com/Cekretik/telegram-bot-api-master"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func convertAmount(amount float64, rate float64, toRUB bool) float64 {
	if toRUB {
		return amount * rate
	} else {
		return amount / rate
	}
}

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

func handleBalanceCommand(bot *tgbotapi.BotAPI, userID int64, db *gorm.DB) {
	var userState models.UserState
	if err := db.Where("user_id = ?", userID).First(&userState).Error; err != nil {
		log.Printf("Error fetching user state: %v", err)
		return
	}

	rate, err := api.GetCurrencyRate()
	if err != nil {
		log.Printf("Error getting currency rate: %v", err)
		return
	}

	balance := userState.Balance
	var balanceMsgText string

	if userState.Currency == "RUB" {
		balance = convertAmount(balance, rate, true)
		balanceMsgText = fmt.Sprintf("💳 Ваш баланс: ₽%.*f", cmd.DecimalPlaces, balance)
	} else {
		balanceMsgText = fmt.Sprintf("💳 Ваш баланс: $%.*f", cmd.DecimalPlaces, balance)
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
	var userState models.UserState
	if err := db.Where("user_id = ?", chatID).First(&userState).Error; err != nil {
		log.Printf("Error fetching user state: %v", err)
		return
	}
	rate, err := api.GetCurrencyRate()
	if err != nil {
		log.Printf("Error getting currency rate: %v", err)
		return
	}
	balance := userState.Balance
	var messageText string
	if userState.Currency == "RUB" {
		balance = convertAmount(balance, rate, true)
		messageText = fmt.Sprintf("🤵‍♂️ Пользователь:%v\n 🔎 ID:%v\n 💳 Ваш баланс:₽%.*f", userState.UserName, userState.UserID, main.DecimalPlaces, balance)
	} else {
		messageText = fmt.Sprintf("🤵‍♂️ Пользователь:%v\n 🔎 ID:%v\n 💳 Ваш баланс:$%.*f", userState.UserName, userState.UserID, main.DecimalPlaces, balance)
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
	var userOrders []models.UserOrders
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

func GiveSubscriptionBonus(bot *tgbotapi.BotAPI, db *gorm.DB, userState *models.UserState) {
	rate, _ := api.GetCurrencyRate()
	bonusAmount := 25.00 / rate
	userState.Balance += bonusAmount
	bonusGiven++
	message := ("🎁 Поздравляем, Вы получили бонус за подписку!\n\n🌟 Ваш баланс пополнен на 25р")
	bot.Send(tgbotapi.NewMessage(userState.UserID, message))
	userState.IsNewUser = false
}

func handleFavoritesCommand(bot *tgbotapi.BotAPI, db *gorm.DB, chatID int64) {
	favorites, err := database.GetUserFavorites(db, chatID)
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
	var referrals []models.Referral
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

func handleChangeCurrency(bot *tgbotapi.BotAPI, userID int64, db *gorm.DB, toRUB bool) {
	var user models.UserState
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

func FormatServiceInfo(service models.Services, subcategory models.Subcategory, increasePercent float64, userCurrency string, currencyRate float64) string {
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
			service.ID, service.Name, subcategory.Name, currencySymbol, cmd.DecimalPlaces, increasedRate, service.Min, service.Max)
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
			service.ID, service.Name, subcategory.Name, currencySymbol, cmd.DecimalPlaces, increasedRate, service.Min, service.Max)
	}
}
