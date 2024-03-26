package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Cekretik/BoostBot/api"
	"github.com/Cekretik/BoostBot/database"
	"github.com/Cekretik/BoostBot/functionality"
	"github.com/Cekretik/BoostBot/models"
	"github.com/Cekretik/BoostBot/payment"
	tgbotapi "github.com/Cekretik/telegram-bot-api-master"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := database.InitDB()
	if err != nil {
		log.Panic(err)
	}
	go BotManager(db)
	go RunBots(db)

	doneCategories := make(chan bool)
	doneOrder := make(chan bool)
	go database.UpdateCategoriesInDB(db, doneCategories)
	go database.UpdateSubcategoriesInDB(db, doneCategories)
	go database.UpdateServicesInDB(db, doneCategories)
	go database.UpdateOrdersPeriodically(db, doneOrder)
	go api.UpdateCurrencyRatePeriodically()
	go payment.StartHTTPServer(db)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	select {}
}

func BotManager(db *gorm.DB) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	token := os.Getenv("TOKEN_BOT")
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	for update := range updates {
		if update.CallbackQuery != nil {
			chatID := update.CallbackQuery.Message.Chat.ID
			callbackData := update.CallbackQuery.Data
			switch callbackData {
			case "create_bot":
				InitiateTokenInput(bot, chatID)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "bots":
				HandleBotStart(bot, update.CallbackQuery, db)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "backtomenu":
				HandleBackButton(bot, update.CallbackQuery, db)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			}
		}
		if update.Message != nil {
			chatID := update.Message.Chat.ID
			if strings.HasPrefix(update.Message.Text, "/start") {
				WelcomeMessage(bot, chatID)
				SendMenuButton(bot, chatID, db)
				continue
			} else if update.Message.Text == "" {
				WelcomeMessage(bot, chatID)
				SendMenuButton(bot, chatID, db)
				continue
			}
			if update.Message.Text == "Меню" {
				SendMenuButton(bot, chatID, db)
				continue
			}
		}
		if update.Message != nil && update.Message.Text != "" {
			if status, exists := BotStatuses[update.Message.Chat.ID]; exists && status.CurrentState == "awaiting_token" {
				HandleTokenInput(bot, update, db) // Обработка ввода токена
			}
		}

	}
}

func ProcessMessages(bot *tgbotapi.BotAPI, db *gorm.DB) {
	itemsPerPage := 10
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	channelIDStr := os.Getenv("CHANNEL_ID")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		log.Panic("Ошибка при преобразовании CHANNEL_ID:", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.CallbackQuery != nil {
			chatID := update.CallbackQuery.Message.Chat.ID
			callbackData := update.CallbackQuery.Data
			switch callbackData {
			case "replenishBalance":
				payment.HandleReplenishCommand(bot, update.CallbackQuery.Message.Chat.ID)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "cryptomus_USDT", "cryptomus_BTC", "cryptomus_MATIC", "cryptomus_OTHER":
				payment.HandleCryptomusButton(bot, update.CallbackQuery.Message.Chat.ID, db)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

			case "AAIO_SBP", "AAIO_RU":
				payment.HandleAAIOButton(bot, update.CallbackQuery.Message.Chat.ID, db)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "changeCurrencyToRUB":
				functionality.HandleChangeCurrency(bot, chatID, db, true)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "changeCurrencyToUSD":
				functionality.HandleChangeCurrency(bot, chatID, db, false)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "profile:favorites":
				functionality.HandleFavoritesCommand(bot, db, chatID)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "promo":
				functionality.HandlePromoCommand(bot, chatID, db)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "allorders":
				functionality.HandleOrdersCommand(bot, update.CallbackQuery.Message.Chat.ID, db)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "settings":
				functionality.SendSettingsKeyboard(bot, chatID)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			case "techsup":
				functionality.TechSupMessage(bot, chatID)
				bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

			}
			if strings.HasPrefix(update.CallbackQuery.Data, "addFavorite:") || strings.HasPrefix(update.CallbackQuery.Data, "removeFavorite:") {
				functionality.HandleAddToFavoritesCallback(bot, db, update.CallbackQuery)
			}
			if strings.HasPrefix(callbackData, "subcategory:") || strings.HasPrefix(callbackData, "prevServ:") || strings.HasPrefix(callbackData, "nextServ:") {
				var subcategoryID string
				if strings.HasPrefix(callbackData, "subcategory:") {
					subcategoryID = strings.TrimPrefix(callbackData, "subcategory:")
				} else {
					parts := strings.Split(callbackData, ":")
					subcategoryID = parts[1]
				}

				totalServicePages, err := functionality.GetTotalPagesForService(db, itemsPerPage, subcategoryID)
				if err != nil {
					log.Printf("Error getting total pages for services: %v", err)
					continue
				}
				functionality.HandleServiceCallBackQuery(bot, db, update.CallbackQuery, totalServicePages)
			} else if strings.HasPrefix(callbackData, "serviceInfo:") {
				functionality.HandleServiceCallBackQuery(bot, db, update.CallbackQuery, 0)
			} else if strings.HasPrefix(callbackData, "backToServices:") {
				subcategoryID := strings.TrimPrefix(callbackData, "backToServices:")
				totalServicePages, err := functionality.GetTotalPagesForService(db, itemsPerPage, subcategoryID)
				if err != nil {
					log.Printf("Error getting total pages for services: %v", err)
					continue
				}

				functionality.HandleServiceCallBackQuery(bot, db, update.CallbackQuery, totalServicePages)
			} else if strings.HasPrefix(callbackData, "backToSubcategories:") {
				categoryID := strings.TrimPrefix(callbackData, "backToSubcategories:")
				totalPages, err := functionality.GetTotalPagesForCategory(db, itemsPerPage, categoryID)
				if err != nil {
					log.Printf("Error getting total pages for category: %v", err)
					continue
				}
				functionality.HandleServiceCallBackQuery(bot, db, update.CallbackQuery, totalPages)
			} else if strings.HasPrefix(callbackData, "category:") || strings.HasPrefix(callbackData, "prevCat:") || strings.HasPrefix(callbackData, "nextCat:") || strings.HasPrefix(callbackData, "backToCategories:") {
				var categoryID string
				if strings.HasPrefix(callbackData, "category:") {
					categoryID = strings.TrimPrefix(callbackData, "category:")
				} else {
					parts := strings.Split(callbackData, ":")
					categoryID = parts[1]
				}

				totalPages, err := functionality.GetTotalPagesForCategory(db, itemsPerPage, categoryID)
				if err != nil {
					log.Printf("Error getting total pages for category: %v", err)
					continue
				}

				functionality.HandleCallbackQuery(bot, db, update.CallbackQuery, totalPages)
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
				service, err := database.GetService(db, serviceIDInt)
				if err != nil {
					log.Printf("Error getting service '%s': %v", serviceID, err)
					bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Ошибка при получении данных сервиса."))
					continue
				}
				functionality.HandleOrderCommand(bot, update.CallbackQuery.Message.Chat.ID, service)
			}

			// Обработка кнопки "Купить"
			if strings.HasPrefix(callbackData, "buy") {
				chatID := update.CallbackQuery.Message.Chat.ID
				if userStatus, exists := functionality.UserStatuses[chatID]; exists {
					serviceID, err := strconv.Atoi(userStatus.PendingServiceID)
					if err != nil {
						log.Printf("Error converting service ID to integer: %v", err)
						bot.Send(tgbotapi.NewMessage(chatID, "Ошибка: ID сервиса не указан."))
						continue
					}
					service, err := database.GetService(db, serviceID)
					if err != nil {
						log.Printf("Error getting service '%s': %v", userStatus.PendingServiceID, err)
						bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении данных сервиса."))
						continue
					}
					functionality.HandlePurchase(db, bot, chatID, service)
				} else {
					bot.Send(tgbotapi.NewMessage(chatID, "Ваш запрос не может быть обработан. Пожалуйста, начните процесс заново."))
				}
			}
		}
		if update.Message != nil && update.Message.Text != "" {
			chatID := update.Message.Chat.ID

			if status, exists := functionality.UserPromoStatuses[chatID]; exists && status.PromoState == "awaitingPromoCode" {
				functionality.ProcessPromoCodeInput(bot, chatID, update.Message.Text, db)
				delete(functionality.UserPromoStatuses, chatID) // Удаление статуса после обработки
				continue
			}
			if update.Message.Text == "Отмена" {
				if _, exists := functionality.UserPromoStatuses[chatID]; exists {
					delete(functionality.UserPromoStatuses, chatID)
					functionality.SendStandardKeyboard(bot, chatID)
					continue
				}
			}
			if status, exists := functionality.UserPromoStatuses[chatID]; exists && status.PromoState == "awaitingPromoCode" {
				functionality.ProcessPromoCodeInput(bot, chatID, update.Message.Text, db)
				delete(functionality.UserPromoStatuses, chatID)
				continue
			}
		}
		if update.Message != nil {
			chatID := update.Message.Chat.ID
			userPaymentStatus, exists := payment.UserPaymentStatuses[chatID]
			if update.Message.Text == "Отмена" {
				if _, exists := functionality.UserStatuses[chatID]; exists {
					delete(functionality.UserStatuses, chatID)
					functionality.SendStandardKeyboard(bot, chatID)
					continue
				} else if _, exists := payment.UserPaymentStatuses[chatID]; exists {
					delete(payment.UserPaymentStatuses, chatID)
					functionality.SendStandardKeyboard(bot, chatID)
					continue
				}
			}
			functionality.NotifyAdminsAboutNewUser(bot, update.Message.From, update.Message.From.IsPremium, db)
			if exists && userPaymentStatus.CurrentState == "awaitingAmount" {
				payment.HandlePaymentInput(db, bot, chatID, update.Message.Text)
				continue
			} else if exists && userPaymentStatus.CurrentState == "awaitingAmountAAIO" {
				payment.HandlePaymentInputAAIO(db, bot, chatID, update.Message.Text)
				continue
			}
			if strings.HasPrefix(update.Message.Text, "/start") {
				args := strings.Split(update.Message.Text, " ")
				if len(args) > 1 {
					param := args[1]
					// Проверяем, является ли параметр специальной ссылкой
					if strings.Contains(param, "_") {
						functionality.ProcessSpecialLink(bot, update.Message.Chat.ID, param, db)
					} else {
						// Обработка реферального ID
						referrerID, err := strconv.ParseInt(param, 10, 64)
						if err == nil && referrerID != 0 {
							// Проверяем, существует ли пользователь-реферер
							var referrer models.UserState
							if err := db.Where("user_id = ?", referrerID).First(&referrer).Error; err == nil {
								// Проверяем, что реферер и реферал - разные люди
								if referrer.UserID != int64(update.Message.From.ID) {
									// Создаем запись о реферале, если она еще не существует
									var existingReferral models.Referral
									if err := db.Where("referrer_id = ? AND referred_id = ?", referrerID, update.Message.From.ID).First(&existingReferral).Error; err != nil {
										// Добавляем нового реферала
										newReferral := models.Referral{
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
				functionality.HandleCreatePromoCommand(bot, update, db)
				continue
			} else if strings.HasPrefix(update.Message.Text, "/createurl") {
				functionality.HandleCreateUrlCommand(bot, update, db)
				continue
			} else if strings.HasPrefix(update.Message.Text, "/broadcast ") {
				functionality.HandleBroadcastCommand(bot, update, db)
				continue
			} else if strings.HasPrefix(update.Message.Text, "/bonus") {
				functionality.HandleBonusCommand(bot, update, db)
				continue
			}
			if userStatus, exists := functionality.UserStatuses[chatID]; exists && userStatus.CurrentState != "" {
				serviceID, err := strconv.Atoi(userStatus.PendingServiceID)
				if err != nil {
					log.Printf("Error converting service ID to integer: %v", err)
					bot.Send(tgbotapi.NewMessage(chatID, "Ошибка: ID сервиса не указан."))
					continue
				}
				service, err := database.GetService(db, serviceID)
				if err != nil {
					log.Printf("Error getting service '%s': %v", userStatus.PendingServiceID, err)
					bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении данных сервиса."))
					continue
				}
				functionality.HandleUserInput(db, bot, update, service)
			} else {
				userID := update.Message.From.ID
				userName := update.Message.From.UserName
				balance := 0.0
				isSubscribed, err := functionality.CheckSubscriptionStatus(bot, db, channelID, int64(userID), balance, userName)
				if err != nil {
					log.Printf("Error checking subscription status: %v", err)
					continue
				}

				if isSubscribed {
					if update.Message.Text == "💳 Баланс" {
						functionality.HandleBalanceCommand(bot, update.Message.Chat.ID, db)
					} else if update.Message.Text == "🤝 Партнерам" {
						functionality.ShowReferralStats(bot, db, update.Message.Chat.ID)
					} else if update.Message.Text == "✍️Сделать заказ" {
						functionality.SendPromotionMessage(bot, update.Message.Chat.ID, db)
					} else if update.Message.Text == "🧩Профиль" {
						functionality.HandleProfileCommand(bot, update.Message.Chat.ID, db)
					} else if update.Message.Text == "⚡️Сайт (-55%)" {
						functionality.SendSiteMessage(bot, update.Message.Chat.ID)
					} else {
						functionality.SendPromotionMessage(bot, update.Message.Chat.ID, db)
					}
				} else {
					functionality.SendSubscriptionMessage(bot, update.Message.Chat.ID)
				}
			}
		}
	}
}
