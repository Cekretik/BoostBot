package main

import (
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

type UserStatus struct {
	ChatID           int64
	CurrentState     string
	PendingServiceID string
	Link             string
	Quantity         int
}

var userStatuses map[int64]*UserStatus = make(map[int64]*UserStatus)

func getUserStatus(chatID int64) *UserStatus {
	if status, exists := userStatuses[chatID]; exists {
		return status
	}
	userStatuses[chatID] = &UserStatus{ChatID: chatID}
	return userStatuses[chatID]
}

func handleOrderCommand(bot *tgbotapi.BotAPI, chatID int64, service Service) {
	userStatus := getUserStatus(chatID)
	userStatus.CurrentState = "awaitingLink"
	userStatus.PendingServiceID = strconv.Itoa(service.ID)

	msgText := fmt.Sprintf("💬 Вы заказываете услугу: %s.\n\n Айди усулги %d. \n\nДля оформления заказа укажите ссылку.", service.Name, service.ID)
	cancelKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ReplyMarkup = cancelKeyboard
	bot.Send(msg)
}

func handleUserInput(db *gorm.DB, bot *tgbotapi.BotAPI, update tgbotapi.Update, service Service) {
	chatID := update.Message.Chat.ID
	userStatus := getUserStatus(chatID)

	switch userStatus.CurrentState {
	case "awaitingLink":
		userStatus.Link = update.Message.Text
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
		cost := (float64(quantity) / 1000.0) * service.Rate
		// Получение баланса пользователя
		var user UserState
		if err := db.Where("user_id = ?", chatID).First(&user).Error; err != nil {
			// Обработка ошибки получения пользователя
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
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Цена услуги: $%.5f. Ваш баланс: $%.5f.", cost, user.Balance))
			msg.ReplyMarkup = cancelKeyboard
			msg.ReplyMarkup = keyboard
			bot.Send(msg)
		} else {
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💳 Пополнить баланс", "replenish"),
				),
			)
			cancelKeyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Отмена"),
				),
			)
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("На вашем балансе недостаточно средств. Цена услуги: $%.5f. Ваш баланс: $%.5f.", cost, user.Balance))
			msg.ReplyMarkup = cancelKeyboard
			msg.ReplyMarkup = keyboard
			bot.Send(msg)
		}
	}
}

func handlePurchase(bot *tgbotapi.BotAPI, chatID int64, service Service) {

	userStatus, exists := userStatuses[chatID]
	if !exists {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при оформлении заказа. Пожалуйста, попробуйте снова."))
		return
	}
	userStatus.PendingServiceID = strconv.Itoa(service.ID)

	order := Order{
		ServiceID: userStatus.PendingServiceID,
		Link:      userStatus.Link,
		Quantity:  userStatus.Quantity,
	}

	// Отправка заказа
	createdOrder, err := createOrder(order, token)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка при создании заказа: %s", err.Error())))
		return
	}

	// Отправка подтверждения пользователю
	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Заказ успешно создан. Номер заказа: %s", createdOrder.ServiceID)))

	delete(userStatuses, chatID)
}
