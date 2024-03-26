package database

import (
	"log"
	"os"
	"time"

	"github.com/Cekretik/BoostBot/models"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	updateCategoriesInterval    = time.Hour
	updateSubcategoriesInterval = time.Hour
	updateServicesInterval      = time.Hour
)

func InitDB() (*gorm.DB, error) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	dsn := os.Getenv("DSN")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&models.UserState{}, &models.Category{}, &models.Subcategory{}, &models.Services{}, &models.UserOrders{}, &models.RefundedOrder{}, &models.Payments{}, &models.Referral{}, &models.PromoCode{}, &models.UsedPromoCode{}, &models.BotOwners{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
