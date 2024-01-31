package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&UserState{}, &Category{}, &Subcategory{}, &Services{}, &UserOrders{}, &RefundedOrder{}, &Payments{}, &Referral{}, &PromoCode{}, &UsedPromoCode{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
