package main

import (
	"log"

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
			totalPages, _ := GetTotalPagesForCategory(db, itemsPerPage, update.CallbackQuery.Data)
			HandleCallbackQuery(bot, db, update.CallbackQuery, totalPages)
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
