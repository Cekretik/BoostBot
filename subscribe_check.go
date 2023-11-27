package main

import (
	"errors"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

func CheckSubscriptionStatus(bot *tgbotapi.BotAPI, db *gorm.DB, channelID int64, userID int64) (bool, error) {
	chatMember, err := bot.GetChatMember(tgbotapi.ChatConfigWithUser{
		ChatID: channelID,
		UserID: int(userID),
	})
	if err != nil {
		log.Printf("Error getting chat member: %v", err)
		return false, err
	}

	// Проверяем статус подписки
	isSubscribed := chatMember.Status != "left"

	// Обновляем статус подписки в базе данных
	if err := UpdateSubscriptionStatus(db, channelID, userID, isSubscribed); err != nil {
		log.Printf("Error updating subscription status in the database: %v", err)
		return false, err
	}

	return isSubscribed, nil
}

func UpdateSubscriptionStatus(db *gorm.DB, channelID int64, userID int64, subscribed bool) error {
	var userState UserState
	result := db.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&userState)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Если запись не найдена, создаем новую
			userState = UserState{
				UserID:     userID,
				ChannelID:  channelID,
				Subscribed: subscribed,
			}
			if err := db.Create(&userState).Error; err != nil {
				log.Printf("Error creating new user state: %v", err)
				return err
			}
			log.Printf("Created new user state for user ID %v and channel ID %v", userID, channelID)
			return nil
		}
		log.Printf("Error finding user state: %v", result.Error)
		return result.Error
	}

	// Если запись найдена, обновляем статус подписки
	userState.Subscribed = subscribed
	if err := db.Save(&userState).Error; err != nil {
		log.Printf("Error updating user subscription status: %v", err)
		return err
	}

	return nil
}
