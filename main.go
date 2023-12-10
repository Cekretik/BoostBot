package main

import (
	"log"
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

	go UpdateCategoriesInDB(db, doneCategories)
	go UpdateSubcategoriesInDB(db, doneCategories)
	go UpdateServicesInDB(db, doneCategories)

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
			}
		} else if update.Message != nil {
			userID := update.Message.From.ID
			isSubscribed, err := CheckSubscriptionStatus(bot, db, channelID, int64(userID))
			if err != nil {
				log.Printf("Error checking subscription status: %v", err)
				continue
			}

			if isSubscribed {
				WelcomeMessage(bot, update.Message.Chat.ID)
				SendPromotionMessage(bot, update.Message.Chat.ID, db)
			} else {
				SendSubscriptionMessage(bot, update.Message.Chat.ID)
			}
		}
	}
}
