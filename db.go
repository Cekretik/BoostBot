package main

import (
	"errors"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	dsn := "host=localhost user=postgres password=gopher dbname=boostbot port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Миграция таблицы
	err = db.AutoMigrate(&UserState{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func GetUserState(db *gorm.DB, userID, channelID int64, subscribed bool) (*UserState, error) {
	var userState UserState
	result := db.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&userState)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			userState = UserState{
				UserID:     userID,
				ChannelID:  channelID,
				Subscribed: subscribed,
			}
			if err := db.Create(&userState).Error; err != nil {
				log.Printf("Error creating new user state: %v", err)
				return nil, err
			}
			log.Printf("Created new user state for chat ID %v and channel ID %v", userID, channelID)
			return &userState, nil
		}
		log.Printf("Error finding user state: %v", result.Error)
		return nil, result.Error
	}

	return &userState, nil
}

func UpdateUserSubscriptionStatus(db *gorm.DB, userID, channelID int64, subscribed bool) error {
	userState, err := GetUserState(db, userID, channelID, true)
	if err != nil {
		return err
	}

	// Проверяем, изменился ли статус подписки
	if userState.Subscribed != subscribed {
		userState.Subscribed = subscribed
		if err := db.Save(userState).Error; err != nil {
			log.Printf("Error updating user subscription status: %v", err)
			return err
		}
	}

	return nil
}
