package functionality

import (
	"fmt"
	"html"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type Entity struct {
	Type   string
	URL    string
	Offset int
	Length int
}

type UserPromoStatus struct {
	ChatID     int64
	PromoState string
}

var userPromoStatuses = make(map[int64]*UserPromoStatus)

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

func isAdmin(bot *tgbotapi.BotAPI, userID int64) bool {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	channelIDStr := os.Getenv("CHANNEL_ID")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Error parsing CHANNEL_ID: %v", err)
	}

	member, err := bot.GetChatMember(tgbotapi.ChatConfigWithUser{
		ChatID: channelID,
		UserID: int(userID),
	})
	if err != nil {
		log.Printf("Ошибка при получении статуса пользователя: %v", err)
		return false
	}

	return member.Status == "administrator" || member.Status == "creator"
}

func GenerateSpecialLink(linkName string) string {
	return fmt.Sprint(linkName) + "_"
}
func processSpecialLink(bot *tgbotapi.BotAPI, chatID int64, linkCode string, db *gorm.DB) {
	var promo PromoCode

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

	var usedPromo UsedPromoCode
	if err := db.Where("user_id = ? AND promo_code = ?", chatID, linkCode).First(&usedPromo).Error; err == nil {
		msg := tgbotapi.NewMessage(chatID, "Вы уже переходили по этой спец. ссылке.")
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

	UpdateUserBalance(db, chatID, bonusInRubles)
	congratulationMessage := fmt.Sprintf("🎁 Поздравляем, Вы активировали промокод!\n\n🌟 Ваш баланс пополнен на %.2fр", promo.Discount)
	bot.Send(tgbotapi.NewMessage(chatID, congratulationMessage))
	promo.Activations++
	db.Save(&promo)

	newUsedPromo := UsedPromoCode{
		UserID:    chatID,
		PromoCode: linkCode,
		Used:      true,
	}
	db.Create(&newUsedPromo)
}

func broadcastMessage(bot *tgbotapi.BotAPI, db *gorm.DB, message string) {
	var users []UserState
	db.Find(&users)

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

func formatBroadcastMessage(message string, entities []Entity) (string, error) {
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

func convertEntities(tgEntities []tgbotapi.MessageEntity) []Entity {
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
