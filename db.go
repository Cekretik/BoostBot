package main

import (
	"crypto/rand"
	"errors"
	"fmt"
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

// Update categories, subcategories and services in DB
func UpdateCategoriesInDB(db *gorm.DB, done chan bool) {
	for {
		categories, err := fetchCategoriesFromAPI()
		if err != nil {
			log.Printf("Error fetching categories from API: %v", err)
		} else {
			tx := db.Begin()
			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
				}
			}()

			for _, category := range categories {
				if err := updateCategory(tx, category); err != nil {
					log.Printf("Error updating category with ID %s: %v", category.ID, err)
					tx.Rollback()
					break
				}
			}

			if err := tx.Commit().Error; err != nil {
				log.Printf("Error committing transaction for categories: %v", err)
			} else {
				log.Println("Categories updated in the database.")
				done <- true
			}
		}

		time.Sleep(updateCategoriesInterval)
	}
}

func UpdateSubcategoriesInDB(db *gorm.DB, done chan bool) {
	for {
		<-done
		var categories []Category
		db.Find(&categories)

		for _, category := range categories {
			subcategories, err := fetchSubcategoriesFromAPI(category.ID)
			if err != nil {
				log.Printf("Error fetching subcategories from API for category %s: %v", category.Name, err)
				continue
			}

			tx := db.Begin()
			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
				}
			}()

			for _, subcategory := range subcategories {
				if err := updateSubcategory(tx, subcategory); err != nil {
					log.Printf("Error updating subcategory with ID %s: %v", subcategory.ID, err)
					tx.Rollback()
					break
				}
			}

			if err := tx.Commit().Error; err != nil {
				log.Printf("Error committing transaction for subcategories in category %s: %v", category.Name, err)
			} else {
				log.Printf("Subcategories updated in the database for category %s.", category.Name)
			}
		}

		time.Sleep(updateSubcategoriesInterval)
	}
}

func UpdateServicesInDB(db *gorm.DB, done chan bool) {
	for {
		var subcategories []Subcategory
		db.Find(&subcategories)

		for _, subcategory := range subcategories {
			services, err := fetchServicesFromAPI(subcategory.ID)
			if err != nil {
				log.Printf("Error fetching services from API for subcategory %s: %v", subcategory.Name, err)
				continue
			}

			tx := db.Begin()
			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
				}
			}()

			for _, service := range services {
				if err := updateService(tx, service); err != nil {
					log.Printf("Error updating service with ID %s: %v", service.ServiceID, err)
					tx.Rollback()
					break
				}
			}

			if err := tx.Commit().Error; err != nil {
				log.Printf("Error committing transaction for subcategory %s: %v", subcategory.Name, err)
			} else {
				log.Printf("Services updated in the database for subcategory %s.", subcategory.Name)
			}
		}

		time.Sleep(updateServicesInterval)
	}
}

// Get categories, subcategories and services from DB
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

func GetServicesFromDB(db *gorm.DB) ([]Services, error) {
	var services []Services
	if err := db.Find(&services).Error; err != nil {
		log.Printf("Error fetching services from DB: %v", err)
		return nil, err
	}
	return services, nil
}

// Get subcategories by category ID
func GetSubcategoriesByCategoryID(db *gorm.DB, categoryID string) ([]Subcategory, error) {
	var subcategories []Subcategory
	if err := db.Where("category_id = ?", categoryID).Find(&subcategories).Error; err != nil {
		return nil, err
	}
	return subcategories, nil
}

// Get services by subcategory ID
func GetServicesBySubcategoryID(db *gorm.DB, subcategoryID string) ([]Services, error) {
	var services []Services
	if err := db.Where("category_id = ?", subcategoryID).Find(&services).Error; err != nil {
		return nil, err
	}
	return services, nil
}

// Get service by service ID
func GetServiceByID(db *gorm.DB, serviceID string) (Services, error) {
	var service Services
	result := db.First(&service, "service_id = ?", serviceID)
	return service, result.Error
}

// Get subcategory by subcategory ID
func GetSubcategoryByID(db *gorm.DB, subcategoryID string) (Subcategory, error) {
	var subcategory Subcategory
	result := db.First(&subcategory, "subcategory_id = ?", subcategoryID)
	return subcategory, result.Error
}

// Updating category, subcategory and service
func updateCategory(tx *gorm.DB, newCategory Category) error {
	var existingCategory Category
	result := tx.Where("category_id = ?", newCategory.ID).First(&existingCategory)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return tx.Create(&newCategory).Error
		}
		return result.Error
	}

	if existingCategory.Name != newCategory.Name {
		return tx.Model(&existingCategory).Updates(Category{Name: newCategory.Name}).Error
	}

	return nil
}

func updateSubcategory(tx *gorm.DB, newSubcategory Subcategory) error {
	var existingSubcategory Subcategory
	result := tx.Where("subcategory_id = ?", newSubcategory.ID).First(&existingSubcategory)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return tx.Create(&newSubcategory).Error
		}
		return result.Error
	}

	if existingSubcategory.Name != newSubcategory.Name || existingSubcategory.CategoryID != newSubcategory.CategoryID {
		return tx.Model(&existingSubcategory).Updates(Subcategory{Name: newSubcategory.Name, CategoryID: newSubcategory.CategoryID}).Error
	}

	return nil
}

func GetService(db *gorm.DB, id int) (Services, error) {
	var service Services
	result := db.First(&service, "id = ?", id)
	return service, result.Error
}

func updateService(tx *gorm.DB, newService Services) error {
	var existingService Services
	result := tx.Where("service_id = ?", newService.ServiceID).First(&existingService)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return tx.Create(&newService).Error
		}
		return result.Error
	}

	if existingService.ID != newService.ID ||
		existingService.Name != newService.Name ||
		existingService.CategoryID != newService.CategoryID ||
		existingService.Min != newService.Min ||
		existingService.Max != newService.Max ||
		existingService.Dripfeed != newService.Dripfeed ||
		existingService.Refill != newService.Refill ||
		existingService.Cancel != newService.Cancel ||
		existingService.ServiceID != newService.ServiceID ||
		existingService.Rate != newService.Rate ||
		existingService.Type != newService.Type {
		return tx.Model(&existingService).Updates(Services{ID: newService.ID, Name: newService.Name, CategoryID: newService.CategoryID, Min: newService.Min,
			Max: newService.Max, Dripfeed: newService.Dripfeed, Refill: newService.Refill, Cancel: newService.Cancel, ServiceID: newService.ServiceID, Rate: newService.Rate,
			Type: newService.Type}).Error
	}

	return nil
}

func updateOrdersPeriodically(db *gorm.DB, done chan bool) {
	for {
		serviceDetails, err := fetchOrders()
		if err != nil {
			log.Printf("Error fetching orders from API: %v", err)
			continue
		}

		tx := db.Begin()

		for _, detail := range serviceDetails {
			var order UserOrders
			if err := tx.Where("order_id = ?", detail.ID).First(&order).Error; err != nil {
				log.Printf("Error finding order with ID %d: %v", detail.ID, err)
				continue
			}

			// Обновляем поля заказа, если они изменились
			if order.Status != detail.Status || order.Remains != detail.Remains ||
				order.Charge != detail.Charge || order.StartCount != detail.StartCount {
				order.Status = detail.Status
				order.Remains = detail.Remains
				order.Charge = detail.Charge
				order.StartCount = detail.StartCount
				tx.Save(&order)
			}

			if order.Status != "PARTIAL" && order.Status != "CANCELED" && order.Status != "COMPLETED" && order.Status != "IN_PROGRESS" {
				order.Status = "PENDING"
				tx.Save(&order)
			}

			// Возврат средств
			if order.Status == "CANCELED" || order.Status == "PARTIAL" {
				// Проверяем, был ли этот заказ уже возвращен
				var refundedOrder RefundedOrder
				if err := tx.Where("order_id = ?", order.ID).First(&refundedOrder).Error; err == nil {
					// Заказ уже возвращен, пропускаем его
					continue
				}

				var user UserState
				if err := tx.Where("user_id = ?", order.ChatID).First(&user).Error; err != nil {
					log.Printf("Error finding user with ChatID %s: %v", order.ChatID, err)
					continue
				}

				var refundAmount float64
				if order.Status == "CANCELED" {
					refundAmount = order.Cost
				} else if order.Status == "PARTIAL" {
					refundAmount = (float64(detail.Remains) / 1000.0) * detail.Charge
				}

				user.Balance += refundAmount
				tx.Save(&user)

				// Добавляем запись о возврате заказа в базу данных
				tx.Create(&RefundedOrder{OrderID: order.ID})
			}
		}

		if err := tx.Commit().Error; err != nil {
			log.Printf("Error committing transaction for updating orders: %v", err)
			tx.Rollback()
		} else {
			log.Println("User orders updated in the database.")
		}

		select {
		case <-done:
			return
		default:
			time.Sleep(30 * time.Minute)
		}
	}
}

func updatePaymentStatusInDB(db *gorm.DB, orderID, status string) error {
	var payment Payments
	if err := db.Model(&payment).Where("order_id = ?", orderID).Update("status", status).Error; err != nil {
		return err
	}
	return nil
}

func AddServiceToFavorites(db *gorm.DB, userID int64, serviceID int) error {
	var user UserState
	if err := db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		log.Printf("User not found with userID %d: %v", userID, err)
		return err
	}
	log.Printf("skvazizambza %v", serviceID)
	var service Services
	if err := db.Where("id = ?", serviceID).First(&service).Error; err != nil {
		log.Printf("Service not found with serviceID %d: %v", service.ID, err)
		return err
	}

	return db.Model(&user).Association("Favorites").Append(&service)
}

func GetUserFavorites(db *gorm.DB, userID int64) ([]Services, error) {
	var user UserState
	if err := db.Preload("Favorites").Where("user_id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return user.Favorites, nil
}

func RemoveServiceFromFavorites(db *gorm.DB, userID int64, serviceID int) error {
	var user UserState
	var service Services
	if err := db.Where("user_id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	if err := db.Where("id = ?", serviceID).First(&service).Error; err != nil {
		return err
	}

	return db.Model(&user).Association("Favorites").Delete(&service)
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

func generateUniquePromoCode(db *gorm.DB) (string, error) {
	for {
		promoCode := generateRandomCode(8)
		var count int64
		db.Model(&PromoCode{}).Where("code = ?", promoCode).Count(&count)
		if count == 0 {
			return promoCode, nil
		}

	}
}

func generateRandomCode(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		log.Printf("Error generating random code: %v", err)
	}
	return fmt.Sprintf("%X", b)
}

func savePromoCode(db *gorm.DB, discount float64, maxActivations int64) (*PromoCode, error) {
	promoCode, err := generateUniquePromoCode(db)
	if err != nil {
		return nil, err
	}

	newPromo := &PromoCode{
		Code:           promoCode,
		Discount:       discount,
		MaxActivations: maxActivations,
		Activations:    0,
	}

	result := db.Create(newPromo)
	if result.Error != nil {
		return nil, result.Error
	}

	return newPromo, nil
}
