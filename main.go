package main

import (
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var channelID int64 = -1002105354472

func main() {
	bot, err := tgbotapi.NewBotAPI("6497443652:AAHYg2UaQCZuOn1BF3jXllnyMldoXzAERFs")
	if err != nil {
		log.Panic(err)
	}

	db, err := InitDB()
	if err != nil {
		log.Panic(err)
	}

	doneCategories := make(chan bool)
	doneOrder := make(chan bool)
	go UpdateCategoriesInDB(db, doneCategories)
	go UpdateSubcategoriesInDB(db, doneCategories)
	go UpdateServicesInDB(db, doneCategories)
	go updateOrdersPeriodically(db, doneOrder)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}
	itemsPerPage := 10

	for update := range updates {
		if update.CallbackQuery != nil {
			callbackData := update.CallbackQuery.Data
			if strings.HasPrefix(callbackData, "subcategory:") || strings.HasPrefix(callbackData, "prevServ:") || strings.HasPrefix(callbackData, "nextServ:") {
				var subcategoryID string
				if strings.HasPrefix(callbackData, "subcategory:") {
					subcategoryID = strings.TrimPrefix(callbackData, "subcategory:")
				} else {
					parts := strings.Split(callbackData, ":")
					subcategoryID = parts[1]
				}

				totalServicePages, err := GetTotalPagesForService(db, itemsPerPage, subcategoryID)
				if err != nil {
					log.Printf("Error getting total pages for services: %v", err)
					continue
				}

				HandleServiceCallBackQuery(bot, db, update.CallbackQuery, totalServicePages)
			} else if strings.HasPrefix(callbackData, "serviceInfo:") {
				HandleServiceCallBackQuery(bot, db, update.CallbackQuery, 0)
			} else if strings.HasPrefix(callbackData, "backToServices:") {
				subcategoryID := strings.TrimPrefix(callbackData, "backToServices:")
				totalServicePages, err := GetTotalPagesForService(db, itemsPerPage, subcategoryID)
				if err != nil {
					log.Printf("Error getting total pages for services: %v", err)
					continue
				}

				HandleServiceCallBackQuery(bot, db, update.CallbackQuery, totalServicePages)
			} else if strings.HasPrefix(callbackData, "backToSubcategories:") {
				categoryID := strings.TrimPrefix(callbackData, "backToSubcategories:")
				totalPages, err := GetTotalPagesForCategory(db, itemsPerPage, categoryID)
				if err != nil {
					log.Printf("Error getting total pages for category: %v", err)
					continue
				}
				HandleServiceCallBackQuery(bot, db, update.CallbackQuery, totalPages)
			} else if strings.HasPrefix(callbackData, "category:") || strings.HasPrefix(callbackData, "prevCat:") || strings.HasPrefix(callbackData, "nextCat:") || strings.HasPrefix(callbackData, "backToCategories:") {
				var categoryID string
				if strings.HasPrefix(callbackData, "category:") {
					categoryID = strings.TrimPrefix(callbackData, "category:")
				} else {
					parts := strings.Split(callbackData, ":")
					categoryID = parts[1]
				}

				totalPages, err := GetTotalPagesForCategory(db, itemsPerPage, categoryID)
				if err != nil {
					log.Printf("Error getting total pages for category: %v", err)
					continue
				}

				HandleCallbackQuery(bot, db, update.CallbackQuery, totalPages)
			} else if strings.HasPrefix(callbackData, "order:") {
				serviceID := strings.TrimPrefix(callbackData, "order:")
				if serviceID == "" {
					log.Printf("Service ID is empty in callback data: %s", callbackData)
					bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Ошибка: ID сервиса не указан."))
					continue
				}
				serviceIDInt, err := strconv.Atoi(serviceID)
				if err != nil {
					log.Printf("Error converting service ID to integer: %v", err)
					bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Ошибка: ID сервиса не указан."))
					continue
				}
				service, err := GetService(db, serviceIDInt)
				if err != nil {
					log.Printf("Error getting service '%s': %v", serviceID, err)
					bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Ошибка при получении данных сервиса."))
					continue
				}
				handleOrderCommand(bot, update.CallbackQuery.Message.Chat.ID, service)
			}

			// Обработка кнопки "Купить"
			if strings.HasPrefix(callbackData, "buy") {
				chatID := update.CallbackQuery.Message.Chat.ID
				if userStatus, exists := userStatuses[chatID]; exists {
					serviceID, err := strconv.Atoi(userStatus.PendingServiceID)
					if err != nil {
						log.Printf("Error converting service ID to integer: %v", err)
						bot.Send(tgbotapi.NewMessage(chatID, "Ошибка: ID сервиса не указан."))
						continue
					}
					service, err := GetService(db, serviceID)
					if err != nil {
						log.Printf("Error getting service '%s': %v", userStatus.PendingServiceID, err)
						bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении данных сервиса."))
						continue
					}
					handlePurchase(db, bot, chatID, service)
				} else {
					bot.Send(tgbotapi.NewMessage(chatID, "Ваш запрос не может быть обработан. Пожалуйста, начните процесс заново."))
				}
			}
		}

		// Обработка обычных сообщений
		if update.Message != nil {
			chatID := update.Message.Chat.ID

			// Обработка команды "Отмена"
			if update.Message.Text == "Отмена" {
				if _, exists := userStatuses[chatID]; exists {
					delete(userStatuses, chatID)
					delete(userStatuses, chatID)
					sendStandardKeyboard(bot, chatID)
					continue
				}
			}

			// Проверяем, находится ли пользователь в процессе заказа
			if userStatus, exists := userStatuses[chatID]; exists && userStatus.CurrentState != "" {
				serviceID, err := strconv.Atoi(userStatus.PendingServiceID)
				if err != nil {
					log.Printf("Error converting service ID to integer: %v", err)
					bot.Send(tgbotapi.NewMessage(chatID, "Ошибка: ID сервиса не указан."))
					continue
				}
				service, err := GetService(db, serviceID)
				if err != nil {
					log.Printf("Error getting service '%s': %v", userStatus.PendingServiceID, err)
					bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении данных сервиса."))
					continue
				}
				handleUserInput(db, bot, update, service)
			} else {
				// Обработка других команд и сообщений
				userID := update.Message.From.ID
				userName := update.Message.From.UserName
				balance := 0.0

				// Проверка статуса подписки и другие команды
				isSubscribed, err := CheckSubscriptionStatus(bot, db, channelID, int64(userID), balance, userName)
				if err != nil {
					log.Printf("Error checking subscription status: %v", err)
					continue
				}

				if isSubscribed {
					if update.Message.Text == "💰Баланс" {
						handleBalanceCommand(bot, update.Message.Chat.ID, db)
					} else if update.Message.Text == "📝Мои заказы" {
						handleOrdersCommand(bot, update.Message.Chat.ID, db)
					} else {
						WelcomeMessage(bot, update.Message.Chat.ID)
						SendPromotionMessage(bot, update.Message.Chat.ID, db)
					}
				} else {
					SendSubscriptionMessage(bot, update.Message.Chat.ID)
				}
			}
		}
	}
}
