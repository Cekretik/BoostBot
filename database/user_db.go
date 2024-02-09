package database

import (
	"errors"
	"log"

	"gorm.io/gorm"
)

// Get user state
func GetUserState(db *gorm.DB, userID, channelID int64, subscribed bool, balance float64, userName string) (*UserState, error) {
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

// Update user subscription status
func UpdateUserState(db *gorm.DB, userID, channelID int64, subscribed bool, balance float64, userName string) error {
	userState, err := GetUserState(db, userID, channelID, true, balance, userName)
	if err != nil {
		return err
	}

	if userState.Subscribed != subscribed || userState.UserName != userName || userState.Balance != balance {
		userState.Balance = balance
		userState.Subscribed = subscribed
		userState.UserName = userName
		userState.Balance = balance
		if err := db.Save(userState).Error; err != nil {
			log.Printf("Error updating user subscription status: %v", err)
			return err
		}
	}

	return nil
}

func updatePaymentStatusInDB(db *gorm.DB, orderID, status string) error {
	var payment Payments
	if err := db.Model(&payment).Where("order_id = ?", orderID).Update("status", status).Error; err != nil {
		return err
	}
	return nil
}

func UserIsNew(db *gorm.DB, userID int64) bool {
	var user UserState
	result := db.Where("user_id = ?", userID).First(&user)
	return errors.Is(result.Error, gorm.ErrRecordNotFound)
}

func GetUserFavorites(db *gorm.DB, userID int64) ([]Services, error) {
	var user UserState
	if err := db.Preload("Favorites").Where("user_id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return user.Favorites, nil
}

func getUserCurrency(db *gorm.DB, userID int64) (string, error) {
	var user UserState
	if err := db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		log.Printf("Error fetching user state: %v", err)
		return "", err
	}
	return user.Currency, nil
}

func getCurrentCurrencyRate() float64 {
	return currentRate
}
