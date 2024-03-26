package functionality

import (
	"errors"
	"log"

	"github.com/Cekretik/BoostBot/models"
	tgbotapi "github.com/Cekretik/telegram-bot-api-master"
	"gorm.io/gorm"
)

var (
	bonusActive bool  = false
	bonusLimit  int64 = 1
	bonusGiven  int64 = 0
)

func CheckSubscriptionStatus(bot *tgbotapi.BotAPI, db *gorm.DB, channelID, userID int64, balance float64, userName string) (bool, error) {
	chatMemberConfig := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: channelID,
			UserID: userID,
		},
	}
	chatMember, err := bot.GetChatMember(chatMemberConfig)
	if err != nil {
		log.Printf("Error getting chat member: %v", err)
		return false, err
	}

	isSubscribed := chatMember.Status != "left"

	if err := UpdateUserStatus(bot, db, channelID, userID, isSubscribed, balance, userName); err != nil {
		log.Printf("Error updating subscription status in the database: %v", err)
		return false, err
	}

	return isSubscribed, nil
}

func UpdateUserStatus(bot *tgbotapi.BotAPI, db *gorm.DB, channelID int64, userID int64, subscribed bool, balance float64, userName string) error {
	var userState models.UserState
	result := db.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&userState)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			userState = models.UserState{
				UserID:               userID,
				UserName:             userName,
				ChannelID:            channelID,
				Subscribed:           subscribed,
				IsNewUser:            true,
				PreviouslySubscribed: subscribed,
				Balance:              balance,
				Currency:             "RUB",
			}
			if err := db.Create(&userState).Error; err != nil {
				log.Printf("Error creating new user state: %v", err)
				return err
			}
			return nil
		}
		log.Printf("Error finding user state: %v", result.Error)
		return result.Error
	}
	if userState.IsNewUser && subscribed && bonusActive && bonusGiven < bonusLimit {
		GiveSubscriptionBonus(bot, db, &userState)
	}

	//	log.Printf("Found user state: %+v", userState.IsNewUser)
	userState.Subscribed = subscribed
	if subscribed {
		userState.PreviouslySubscribed = true
	}
	userState.UserName = userName
	if !bonusActive || bonusGiven == bonusLimit {
		userState.IsNewUser = false
	}
	if err := db.Save(&userState).Error; err != nil {
		log.Printf("Error updating user subscription status: %v", err)
		return err
	}
	//log.Printf("Found user state2: %+v", userState.IsNewUser)
	return nil
}
