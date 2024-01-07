package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

type UserStatus struct {
	ChatID           int64
	CurrentState     string
	PendingServiceID string
	Link             string
	Quantity         int
	ReplenishAmount  float64
	OrderID          string
}

var userStatuses map[int64]*UserStatus = make(map[int64]*UserStatus)

func getUserStatus(chatID int64) *UserStatus {
	if status, exists := userStatuses[chatID]; exists {
		return status
	}
	userStatuses[chatID] = &UserStatus{ChatID: chatID}
	return userStatuses[chatID]
}

func handleOrderCommand(bot *tgbotapi.BotAPI, chatID int64, service Services) {
	userStatus := getUserStatus(chatID)
	userStatus.CurrentState = "awaitingLink"
	userStatus.PendingServiceID = strconv.Itoa(service.ID)

	msgText := fmt.Sprintf("💬 Вы заказываете услугу: %s.\n\n ID усулги %d. \n\nДля оформления заказа укажите ссылку.", service.Name, service.ID)
	cancelKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = cancelKeyboard
	bot.Send(msg)
}

func isValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func handleUserInput(db *gorm.DB, bot *tgbotapi.BotAPI, update tgbotapi.Update, service Services) {
	chatID := update.Message.Chat.ID
	userStatus := getUserStatus(chatID)

	var user UserState
	if err := db.Where("user_id = ?", chatID).First(&user).Error; err != nil {
		log.Printf("Error fetching user state: %v", err)
		return
	}
	userCurrency := user.Currency
	currencyRate := getCurrentCurrencyRate()

	switch userStatus.CurrentState {
	case "awaitingLink":
		link := update.Message.Text
		if !isValidURL(link) {
			bot.Send(tgbotapi.NewMessage(chatID, "Введите ссылку корректно."))
			return
		}
		userStatus.Link = link
		userStatus.CurrentState = "awaitingQuantity"
		msgText := fmt.Sprintf("Введите количество. Минимальное: %d, максимальное: %d.", service.Min, service.Max)
		msg := tgbotapi.NewMessage(chatID, msgText)
		bot.Send(msg)

	case "awaitingQuantity":
		quantity, err := strconv.Atoi(update.Message.Text)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Пожалуйста, введите действительное число."))
			return
		} else if quantity < service.Min || quantity > service.Max {
			msgText := fmt.Sprintf("Количество должно быть в диапазоне от %d до %d.", service.Min, service.Max)
			bot.Send(tgbotapi.NewMessage(chatID, msgText))
			return
		}
		userStatus.Quantity = quantity
		increasePercent, err := strconv.ParseFloat(os.Getenv("PRICE_PERCENT"), 64)
		if err != nil {
			increasePercent = 0
		}
		cost := (float64(quantity) / 1000.0) * service.Rate
		cost += cost * (increasePercent / 100.0)
		// Получение баланса пользователя
		var user UserState
		if err := db.Where("user_id = ?", chatID).First(&user).Error; err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Произошла ошибка при получении информации о вашем балансе."))
			return
		}

		if user.Balance >= cost {
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💰Купить", "buy"),
				),
			)
			cancelKeyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Отмена"),
				),
			)
			var infoMsg string
			if userCurrency == "RUB" {
				cost = convertAmount(cost, currencyRate, true)
				infoMsg = fmt.Sprintf("Цена услуги: ₽%.*f. Ваш баланс: ₽%.*f.", decimalPlaces, cost, decimalPlaces, convertAmount(user.Balance, currencyRate, true))
			} else {
				infoMsg = fmt.Sprintf("Цена услуги: $%.*f. Ваш баланс: $%.*f.", decimalPlaces, cost, decimalPlaces, user.Balance)
			}
			msg := tgbotapi.NewMessage(chatID, infoMsg)
			msg.ReplyMarkup = cancelKeyboard
			msg.ReplyMarkup = keyboard
			bot.Send(msg)
		} else {
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💰Пополнить баланс", "replenishBalance"),
				),
			)
			cancelKeyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Отмена"),
				),
			)
			var infoMsg string
			if userCurrency == "RUB" {
				cost = convertAmount(cost, currencyRate, true)
				infoMsg = fmt.Sprintf("На вашем балансе недостаточно средств. Цена услуги: ₽%.*f. Ваш баланс: ₽%.*f.", decimalPlaces, cost, decimalPlaces, convertAmount(user.Balance, currencyRate, true))
			} else {
				infoMsg = fmt.Sprintf("Цена услуги: $%.*f. Ваш баланс: $%.*f.", decimalPlaces, cost, decimalPlaces, user.Balance)
			}
			msg := tgbotapi.NewMessage(chatID, infoMsg)
			msg.ReplyMarkup = cancelKeyboard
			msg.ReplyMarkup = keyboard
			bot.Send(msg)
		}
	}
}

func handlePurchase(db *gorm.DB, bot *tgbotapi.BotAPI, chatID int64, service Services) {
	userStatus, exists := userStatuses[chatID]
	if !exists {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при оформлении заказа. Пожалуйста, попробуйте снова."))
		return
	}
	userStatus.PendingServiceID = strconv.Itoa(service.ID)

	var user UserState
	if err := db.Where("user_id = ?", chatID).First(&user).Error; err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Произошла ошибка при доступе к вашему балансу."))
		return
	}

	cost := (float64(userStatus.Quantity) / 1000.0) * service.Rate
	if user.Balance < cost {
		bot.Send(tgbotapi.NewMessage(chatID, "На вашем балансе недостаточно средств для оформления заказа."))
		return
	}
	user.Balance -= cost
	db.Save(&user)

	order := Order{
		ServiceID: userStatus.PendingServiceID,
		Link:      userStatus.Link,
		Quantity:  userStatus.Quantity,
	}

	// Отправка заказа
	createdOrder, err := createOrder(order, token)
	if err != nil {
		user.Balance += cost
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка при создании заказа: %s", err.Error())))
		return
	}

	db.Model(&UserOrders{}).Create(map[string]interface{}{
		"ChatID":     strconv.FormatInt(chatID, 10),
		"ServiceID":  createdOrder.ServiceID,
		"Cost":       createdOrder.Cost,
		"OrderID":    createdOrder.OrderID,
		"CreatedAt":  createdOrder.CreatedAt,
		"UpdatedAt":  createdOrder.UpdatedAt,
		"DeletedAt":  createdOrder.DeletedAt,
		"Link":       createdOrder.Link,
		"Quantity":   createdOrder.Quantity,
		"Status":     createdOrder.Status,
		"Charge":     createdOrder.Charge,
		"StartCount": createdOrder.StartCount,
		"Remains":    createdOrder.Remains,
	})

	// Отправка подтверждения пользователю
	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Заказ успешно создан. ID услуги: %s", createdOrder.ServiceID)))
	delete(userStatuses, chatID)
	sendKeyboardAfterOrder(bot, chatID)
}
