package main

import (
	"errors"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	dsn := "host=localhost user=postgres password=gopher dbname=boostbot port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	err := db.AutoMigrate(&UserState{}, &Category{}, &Subcategory{}, &APIService{})
	if err != nil {
		return err
	}

	// Определение связей
	db.Model(&APIService{}).Association("Subcategory")

	return nil
}

// Миграция таблицы

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

func UpdateCategoriesInDB(db *gorm.DB) {

	for {
		categories, err := fetchCategoriesFromAPI()
		if err != nil {
			log.Printf("Error fetching categories from API: %v", err)
		} else {
			// Очищаем текущие категории в БД
			db.Exec("DELETE FROM categories")

			// Вставляем новые категории
			for _, category := range categories {
				db.Create(&category)
			}

			log.Println("Categories updated in the database.")
		}

		// Ждем заданный интервал перед следующим обновлением
		time.Sleep(updateCategoriesInterval)
	}
}

// UpdateSubcategoriesInDB periodically updates the subcategories in the database from the API.
func UpdateSubcategoriesInDB(db *gorm.DB) {
	for {
		var categories []Category
		db.Find(&categories)

		for _, category := range categories {
			subcategories, err := fetchSubcategoriesFromAPI(category.ID)
			if err != nil {
				log.Printf("Error fetching subcategories from API for category %s: %v", category.Name, err)
			} else {
				// Очищаем текущие подкатегории в БД для данной категории
				db.Exec("DELETE FROM subcategories WHERE category_id = ?", category.ID)

				// Вставляем новые подкатегории
				for _, subcategory := range subcategories {
					db.Create(&subcategory)
				}

				log.Printf("Subcategories updated in the database for category %s.", category.Name)
			}
		}

		// Ждем заданный интервал перед следующим обновлением
		time.Sleep(updateSubcategoriesInterval)
	}
}

// UpdateServicesInDB periodically updates the services in the database from the API.
func UpdateServicesInDB(db *gorm.DB) {
	for {
		var subcategories []Subcategory
		db.Find(&subcategories)

		for _, subcategory := range subcategories {
			services, err := fetchServicesFromAPI(subcategory.ID)
			if err != nil {
				log.Printf("Error fetching services from API for subcategory %s: %v", subcategory.Name, err)
			} else {
				// Очищаем текущие сервисы в БД для данной подкатегории
				db.Exec("DELETE FROM services WHERE subcategory_id = ?", subcategory.ID)

				// Вставляем новые сервисы
				for _, service := range services {
					db.Create(&service)
				}

				log.Printf("Services updated in the database for subcategory %s.", subcategory.Name)
			}
		}

		// Ждем заданный интервал перед следующим обновлением
		time.Sleep(updateServicesInterval)
	}
}

// GetCategoriesFromDB retrieves categories from the database.
func GetCategoriesFromDB(db *gorm.DB) ([]Category, error) {
	var categories []Category
	if err := db.Find(&categories).Error; err != nil {
		log.Printf("Error fetching categories from DB: %v", err)
		return nil, err
	}
	return categories, nil
}

func GetSubCategoriesFromDB(db *gorm.DB) ([]Subcategory, error) {
	var subcategories []Subcategory
	if err := db.Find(&subcategories).Error; err != nil {
		log.Printf("Error fetching subcategories from DB: %v", err)
		return nil, err
	}
	return subcategories, nil
}
