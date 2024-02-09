package database

import (
	"errors"
	"log"

	"github.com/Cekretik/BoostBot/models"
	"gorm.io/gorm"
)

// Get user state
func GetUserState(db *gorm.DB, userID, channelID int64, subscribed bool, balance float64, userName string) (*models.UserState, error) {
	var userState models.UserState
	result := db.Where("user_id = ? AND channel_id = ?", userID, channelID).First(&userState)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			userState = models.UserState{
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

func UpdateUserBalance(db *gorm.DB, userID int64, amount float64) error {
	var user models.UserState
	if err := db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		return err
	}

	var activePromoCode models.UsedPromoCode
	if err := db.Where("user_id = ? AND used = ?", userID, false).First(&activePromoCode).Error; err == nil {
		var promo models.PromoCode
		if err := db.Where("code = ?", activePromoCode.PromoCode).First(&promo).Error; err == nil {
			bonus := amount * promo.Discount / 100
			amount += bonus

			db.Model(&models.UsedPromoCode{}).Where("user_id = ? AND promo_code = ?", userID, activePromoCode.PromoCode).Update("used", true)
		}
	}

	user.Balance += amount
	if err := db.Save(&user).Error; err != nil {
		return err
	}

	var referral models.Referral
	if err := db.Where("referred_id = ?", userID).First(&referral).Error; err == nil {
		commission := amount * 0.10
		db.Model(&models.UserState{}).Where("user_id = ?", referral.ReferrerID).Update("balance", gorm.Expr("balance + ?", commission))
		db.Model(&models.Referral{}).Where("id = ?", referral.ID).Update("amount_earned", gorm.Expr("amount_earned + ?", commission))
	}
	return nil
}

func UpdatePaymentStatusInDB(db *gorm.DB, orderID, status string) error {
	var payment models.Payments
	if err := db.Model(&payment).Where("order_id = ?", orderID).Update("status", status).Error; err != nil {
		return err
	}
	return nil
}

func UserIsNew(db *gorm.DB, userID int64) bool {
	var user models.UserState
	result := db.Where("user_id = ?", userID).First(&user)
	return errors.Is(result.Error, gorm.ErrRecordNotFound)
}

func GetUserFavorites(db *gorm.DB, userID int64) ([]models.Services, error) {
	var user models.UserState
	if err := db.Preload("Favorites").Where("user_id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return user.Favorites, nil
}

func GetUserCurrency(db *gorm.DB, userID int64) (string, error) {
	var user models.UserState
	if err := db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		log.Printf("Error fetching user state: %v", err)
		return "", err
	}
	return user.Currency, nil
}
