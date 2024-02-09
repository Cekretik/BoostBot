package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/Cekretik/telegram-bot-api-master"
	"github.com/joho/godotenv"
)

var DecimalPlaces = 4

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	token := os.Getenv("TOKEN_BOT")
	channelIDStr := os.Getenv("CHANNEL_ID")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Error parsing CHANNEL_ID: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(token)
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
	go updateCurrencyRatePeriodically()
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	itemsPerPage := 10
	go startHTTPServer(db)

	for update := range updates {
		if update.CallbackQuery != nil {
			chatID := update.CallbackQuery.Message.Chat.ID
			callbackData := update.CallbackQuery.Data
			switch callbackData {
			case "replenishBalance":
				handleReplenishCommand(bot, update.CallbackQuery.Message.Chat.ID)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "cryptomus_USDT", "cryptomus_BTC", "cryptomus_MATIC", "cryptomus_OTHER":
				handleCryptomusButton(bot, update.CallbackQuery.Message.Chat.ID, db)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

			case "AAIO_SBP", "AAIO_RU":
				handleAAIOButton(bot, update.CallbackQuery.Message.Chat.ID, db)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "changeCurrencyToRUB":
				handleChangeCurrency(bot, chatID, db, true)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "changeCurrencyToUSD":
				handleChangeCurrency(bot, chatID, db, false)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "profile:favorites":
				handleFavoritesCommand(bot, db, chatID)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "promo":
				handlePromoCommand(bot, chatID, db)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "allorders":
				handleOrdersCommand(bot, update.CallbackQuery.Message.Chat.ID, db)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "settings":
				sendSettingsKeyboard(bot, chatID)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "techsup":
				techSupMessage(bot, chatID)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

			}
			if strings.HasPrefix(update.CallbackQuery.Data, "addFavorite:") || strings.HasPrefix(update.CallbackQuery.Data, "removeFavorite:") {
				handleAddToFavoritesCallback(bot, db, update.CallbackQuery)
			}
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
		if update.Message != nil && update.Message.Text != "" {
			chatID := update.Message.Chat.ID

			if status, exists := userPromoStatuses[chatID]; exists && status.PromoState == "awaitingPromoCode" {
				processPromoCodeInput(bot, chatID, update.Message.Text, db)
				delete(userPromoStatuses, chatID) // Удаление статуса после обработки
				continue
			}
			if update.Message.Text == "Отмена" {
				if _, exists := userPromoStatuses[chatID]; exists {
					delete(userPromoStatuses, chatID)
					sendStandardKeyboard(bot, chatID)
					continue
				}
			}
			if status, exists := userPromoStatuses[chatID]; exists && status.PromoState == "awaitingPromoCode" {
				processPromoCodeInput(bot, chatID, update.Message.Text, db)
				delete(userPromoStatuses, chatID)
				continue
			}
		}
		if update.Message != nil {
			chatID := update.Message.Chat.ID
			userPaymentStatus, exists := userPaymentStatuses[chatID]
			if update.Message.Text == "Отмена" {
				if _, exists := userStatuses[chatID]; exists {
					delete(userStatuses, chatID)
					sendStandardKeyboard(bot, chatID)
					continue
				} else if _, exists := userPaymentStatuses[chatID]; exists {
					delete(userPaymentStatuses, chatID)
					sendStandardKeyboard(bot, chatID)
					continue
				}
			}
			notifyAdminsAboutNewUser(bot, update.Message.From, update.Message.From.IsPremium, db)
			if exists && userPaymentStatus.CurrentState == "awaitingAmount" {
				handlePaymentInput(db, bot, chatID, update.Message.Text)
				continue
			} else if exists && userPaymentStatus.CurrentState == "awaitingAmountAAIO" {
				handlePaymentInputAAIO(db, bot, chatID, update.Message.Text)
				continue
			}
			if strings.HasPrefix(update.Message.Text, "/start") {
				args := strings.Split(update.Message.Text, " ")
				if len(args) > 1 {
					param := args[1]
					// Проверяем, является ли параметр специальной ссылкой
					if strings.Contains(param, "_") {
						processSpecialLink(bot, update.Message.Chat.ID, param, db)
					} else {
						// Обработка реферального ID
						referrerID, err := strconv.ParseInt(param, 10, 64)
						if err == nil && referrerID != 0 {
							// Проверяем, существует ли пользователь-реферер
							var referrer UserState
							if err := db.Where("user_id = ?", referrerID).First(&referrer).Error; err == nil {
								// Проверяем, что реферер и реферал - разные люди
								if referrer.UserID != int64(update.Message.From.ID) {
									// Создаем запись о реферале, если она еще не существует
									var existingReferral Referral
									if err := db.Where("referrer_id = ? AND referred_id = ?", referrerID, update.Message.From.ID).First(&existingReferral).Error; err != nil {
										// Добавляем нового реферала
										newReferral := Referral{
											ReferrerID:   referrerID,
											ReferredID:   int64(update.Message.From.ID),
											AmountEarned: 0,
										}
										db.Create(&newReferral)
									}
								}
							}
						}
					}
				}
			} else if strings.HasPrefix(update.Message.Text, "/createpromo") {
				handleCreatePromoCommand(bot, update, db)
				continue
			} else if strings.HasPrefix(update.Message.Text, "/createurl") {
				handleCreateUrlCommand(bot, update, db)
				continue
			} else if strings.HasPrefix(update.Message.Text, "/broadcast ") {
				handleBroadcastCommand(bot, update, db)
				continue
			} else if strings.HasPrefix(update.Message.Text, "/bonus") {
				handleBonusCommand(bot, update, db)
				continue
			}
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
				userID := update.Message.From.ID
				userName := update.Message.From.UserName
				balance := 0.0
				isSubscribed, err := CheckSubscriptionStatus(bot, db, channelID, int64(userID), balance, userName)
				if err != nil {
					log.Printf("Error checking subscription status: %v", err)
					continue
				}

				if isSubscribed {
					if update.Message.Text == "💳 Баланс" {
						handleBalanceCommand(bot, update.Message.Chat.ID, db)
					} else if update.Message.Text == "🤝 Партнерам" {
						ShowReferralStats(bot, db, update.Message.Chat.ID)
					} else if update.Message.Text == "✍️Сделать заказ" {
						SendPromotionMessage(bot, update.Message.Chat.ID, db)
					} else if update.Message.Text == "🧩Профиль" {
						handleProfileCommand(bot, update.Message.Chat.ID, db)
					} else if update.Message.Text == "⚡️Сайт (-55%)" {
						SendSiteMessage(bot, update.Message.Chat.ID)
					} else {
						SendPromotionMessage(bot, update.Message.Chat.ID, db)
					}
				} else {
					SendSubscriptionMessage(bot, update.Message.Chat.ID)
				}
			}
		}
	}
}
