package main

import (
	"errors"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

func CheckSubscriptionStatus(bot *tgbotapi.BotAPI, db *gorm.DB, channelID int64, userID int64, balance float64, userName string) (bool, error) {
	chatMember, err := bot.GetChatMember(tgbotapi.ChatConfigWithUser{
		ChatID: channelID,
		UserID: int(userID),
	})
	if err != nil {
		log.Printf("Error getting chat member: %v", err)
		return false, err
	}

	isSubscribed := chatMember.Status != "left"

	if err := UpdateUserStatus(db, channelID, userID, isSubscribed, balance, userName); err != nil {
		log.Printf("Error updating subscription status in the database: %v", err)
		return false, err
	}

	return isSubscribed, nil
}

func UpdateUserStatus(db *gorm.DB, channelID int64, userID int64, subscribed bool, balance float64, userName string) error {
	var userState UserState
	result := db.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&userState)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {

			userState = UserState{
				UserID:     userID,
				UserName:   userName,
				ChannelID:  channelID,
				Subscribed: subscribed,
				Balance:    balance,
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

	userState.Subscribed = subscribed
	userState.UserName = userName
	if err := db.Save(&userState).Error; err != nil {
		log.Printf("Error updating user subscription status: %v", err)
		return err
	}

	return nil
}
