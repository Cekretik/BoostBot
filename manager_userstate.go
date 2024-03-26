package main

import (
	"errors"
	"log"

	"github.com/Cekretik/BoostBot/models"
	tgbotapi "github.com/Cekretik/telegram-bot-api-master"
	"gorm.io/gorm"
)

func UpdateUserStatus(bot *tgbotapi.BotAPI, db *gorm.DB, userID int64, userName string) error {
	var botOwners models.BotOwners
	result := db.Where("user_id = ?", userID).First(&botOwners)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			botOwners = models.BotOwners{
				UserID:   userID,
				UserName: userName,
				Token:    "",
				Running:  false,
				BotName:  "",
			}
			if err := db.Create(&botOwners).Error; err != nil {
				log.Printf("Error creating new user state: %v", err)
				return err
			}
			return nil
		}
		log.Printf("Error getting user state: %v", result.Error)
		return result.Error
	}
	if err := db.Model(&botOwners).Where("user_id = ?", userID).Updates(models.BotOwners{UserName: userName}).Error; err != nil {
		log.Printf("Error updating user state: %v", err)
		return err
	}

	return nil
}
