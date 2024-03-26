package functionality

import (
	"fmt"
	"html"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Cekretik/BoostBot/api"
	"github.com/Cekretik/BoostBot/database"
	"github.com/Cekretik/BoostBot/models"
	tgbotapi "github.com/Cekretik/telegram-bot-api-master"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type UserPromoStatus struct {
	ChatID     int64
	PromoState string
}

type Entity struct {
	Type   string
	URL    string
	Offset int
	Length int
}

var UserPromoStatuses = make(map[int64]*UserPromoStatus)

func ConvertEntities(tgEntities []tgbotapi.MessageEntity) []Entity {
	var entities []Entity
	for _, e := range tgEntities {
		entities = append(entities, Entity{
			Type:   e.Type,
			URL:    e.URL,
			Offset: e.Offset,
			Length: e.Length,
		})
	}
	return entities
}

func HandlePromoCommand(bot *tgbotapi.BotAPI, chatID int64, db *gorm.DB) {
	messageText := "✍️Введите ваш промокод:"
	cancelKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	UserPromoStatuses[chatID] = &UserPromoStatus{
		ChatID:     chatID,
		PromoState: "awaitingPromoCode",
	}
	msg := tgbotapi.NewMessage(chatID, messageText)
	msg.ReplyMarkup = cancelKeyboard
	bot.Send(msg)

}

func ProcessPromoCodeInput(bot *tgbotapi.BotAPI, chatID int64, promoCode string, db *gorm.DB) {
	if promoCode == "Отмена" {
		SendStandardKeyboard(bot, chatID)
		return
	}

	var promo models.PromoCode
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

	var usedPromo models.UsedPromoCode
	if err := db.Where("user_id = ? AND promo_code = ?", chatID, promoCode).First(&usedPromo).Error; err == nil {
		msg := tgbotapi.NewMessage(chatID, "Вы уже использовали этот промокод.")
		msg.ReplyMarkup = CreateQuickReplyMarkup()
		bot.Send(msg)
		return
	}
	rate, err := api.GetCurrencyRate()
	if err != nil {
		log.Printf("Error getting currency rate: %v", err)
		return
	}
	bonusInRubles := promo.Discount / rate
	switch promo.Type {
	case "fixed":
		database.UpdateUserBalance(db, chatID, bonusInRubles)
		congratulationMessage := fmt.Sprintf("🎁 Поздравляем, Вы активировали промокод!\n\n🌟 Ваш баланс пополнен на %.2fр", promo.Discount)
		bot.Send(tgbotapi.NewMessage(chatID, congratulationMessage))
	}
	newUsedPromo := models.UsedPromoCode{
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

func HandleCreatePromoCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB) {
	if !IsAdmin(bot, int64(update.Message.From.ID)) {
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

	promo := models.PromoCode{
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
func IsAdmin(bot *tgbotapi.BotAPI, userID int64) bool {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	channelIDStr := os.Getenv("CHANNEL_ID")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Error parsing CHANNEL_ID: %v", err)
	}

	chatMemberConfig := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: channelID,
			UserID: userID,
		},
	}
	member, err := bot.GetChatMember(chatMemberConfig)
	if err != nil {
		log.Printf("Ошибка при получении статуса пользователя: %v", err)
		return false
	}

	return member.Status == "administrator" || member.Status == "creator"
}

func HandleCreateUrlCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	botLink := os.Getenv("BOT_LINK")
	if !IsAdmin(bot, int64(update.Message.From.ID)) {
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
	var existingPromo models.PromoCode
	if db.Where("code = ?", linkCode).First(&existingPromo).Error == nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ссылка с таким названием уже была создана ранее."))
		return
	}
	promo := models.PromoCode{
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

func GenerateSpecialLink(linkName string) string {
	return fmt.Sprint(linkName) + "_"
}
func ProcessSpecialLink(bot *tgbotapi.BotAPI, chatID int64, linkCode string, db *gorm.DB) {
	var promo models.PromoCode

	if err := db.Where("code = ?", linkCode).First(&promo).Error; err != nil {
		msg := tgbotapi.NewMessage(chatID, "Спец. ссылка не найдена.")
		msg.ReplyMarkup = CreateQuickReplyMarkup()
		bot.Send(msg)
		return
	}

	if promo.Activations >= promo.MaxActivations {
		msg := tgbotapi.NewMessage(chatID, "Эта спец. ссылка уже использована максимальное количество раз.")
		msg.ReplyMarkup = CreateQuickReplyMarkup()
		bot.Send(msg)
		return
	}

	var usedPromo models.UsedPromoCode
	if err := db.Where("user_id = ? AND promo_code = ?", chatID, linkCode).First(&usedPromo).Error; err == nil {
		msg := tgbotapi.NewMessage(chatID, "Вы уже переходили по этой спец. ссылке.")
		msg.ReplyMarkup = CreateQuickReplyMarkup()
		bot.Send(msg)
		return
	}

	rate, err := api.GetCurrencyRate()
	if err != nil {
		log.Printf("Error getting currency rate: %v", err)
		return
	}
	bonusInRubles := promo.Discount / rate

	database.UpdateUserBalance(db, chatID, bonusInRubles)
	congratulationMessage := fmt.Sprintf("🎁 Поздравляем, Вы активировали промокод!\n\n🌟 Ваш баланс пополнен на %.2fр", promo.Discount)
	bot.Send(tgbotapi.NewMessage(chatID, congratulationMessage))
	promo.Activations++
	db.Save(&promo)

	newUsedPromo := models.UsedPromoCode{
		UserID:    chatID,
		PromoCode: linkCode,
		Used:      true,
	}
	db.Create(&newUsedPromo)
}

func HandleBonusCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB) {
	if IsAdmin(bot, int64(update.Message.From.ID)) {
		bonusActive = !bonusActive
		message := "Бонус за подписку деактивирован."

		if bonusActive {
			bonusGiven = 0
			message = "Бонус за подписку активирован."
		}

		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, message))
	} else {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "У вас нет прав доступа к этой команде."))
	}
}

func HandleBroadcastCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update, db *gorm.DB) {
	if !IsAdmin(bot, int64(update.Message.From.ID)) {
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
	if len(update.Message.Entities) > 0 {
		shiftedEntities := make([]tgbotapi.MessageEntity, len(update.Message.Entities))
		for i, entity := range update.Message.Entities {
			if entity.Offset >= len(parts[0]) {
				shiftedEntities[i] = entity
				shiftedEntities[i].Offset -= len(parts[0]) + 1
			}
		}
		entities = ConvertEntities(shiftedEntities)
	}

	formattedMessage, err := FormatBroadcastMessage(message, entities)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка форматирования сообщения: "+err.Error()))
		return
	}

	go BroadcastMessage(bot, db, formattedMessage)
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Рассылка началась."))
}
func BroadcastMessage(bot *tgbotapi.BotAPI, db *gorm.DB, message string) {
	var users []models.UserState
	db.Find(&users).Where("previously_subscribed = ? AND subscribed = ?", true, true)

	for _, user := range users {
		msg := tgbotapi.NewMessage(user.UserID, message)
		msg.ParseMode = tgbotapi.ModeHTML
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Не удалось отправить сообщение пользователю с chat ID %d: %v", user.UserID, err)
		}
	}

	log.Println("Рассылка завершена.")
}

func FormatBroadcastMessage(message string, entities []Entity) (string, error) {
	var formattedMessage strings.Builder

	lastIdx := 0
	for _, entity := range entities {
		if entity.Offset < lastIdx || entity.Offset+entity.Length > len(message) {
			return "", fmt.Errorf("некорректные границы сущности: Offset=%d, Length=%d, lastIdx=%d, messageLength=%d",
				entity.Offset, entity.Length, lastIdx, len(message))
		}

		formattedMessage.WriteString(html.EscapeString(message[lastIdx:entity.Offset]))

		entityText := html.EscapeString(message[entity.Offset : entity.Offset+entity.Length])

		switch entity.Type {
		case "bold":
			formattedMessage.WriteString("<b>" + entityText + "</b>")
		case "italic":
			formattedMessage.WriteString("<i>" + entityText + "</i>")
		case "code":
			formattedMessage.WriteString("<code>" + entityText + "</code>")
		case "pre":
			formattedMessage.WriteString("<pre>" + entityText + "</pre>")
		case "text_link":
			formattedMessage.WriteString("<a href=\"" + entity.URL + "\">" + entityText + "</a>")
		case "underline":
			formattedMessage.WriteString("<u>" + entityText + "</u>")
		case "strikethrough":
			formattedMessage.WriteString("<s>" + entityText + "</s>")
		case "spoiler":
			formattedMessage.WriteString("<tg-spoiler>" + entityText + "</tg-spoiler>")
		case "blockquote":
			formattedMessage.WriteString("<blockquote>" + entityText + "</blockquote>")
		default:
			formattedMessage.WriteString(entityText)
		}

		lastIdx = entity.Offset + entity.Length
	}

	formattedMessage.WriteString(html.EscapeString(message[lastIdx:]))

	return formattedMessage.String(), nil
}

func NotifyAdminsAboutNewUser(bot *tgbotapi.BotAPI, user *tgbotapi.User, isPremium bool, db *gorm.DB) {
	if !database.UserIsNew(db, user.ID) {
		return
	}
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	channelIDStr := os.Getenv("CHANNEL_ID")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Error parsing CHANNEL_ID: %v", err)
	}

	chatAdministratorsConfig := tgbotapi.ChatAdministratorsConfig{
		ChatConfig: tgbotapi.ChatConfig{
			ChatID: channelID,
		},
	}

	admins, err := bot.GetChatAdministrators(chatAdministratorsConfig)
	if err != nil {
		log.Printf("Ошибка при получении списка администраторов: %v", err)
		return
	}

	messageText := fmt.Sprintf("Новый пользователь: %s\nID: %d\nРегион: %s\nPremium: %t",
		user.UserName, user.ID, user.LanguageCode, isPremium)

	for _, admin := range admins {
		msg := tgbotapi.NewMessage(admin.User.ID, messageText)
		bot.Send(msg)

	}
}
