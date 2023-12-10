package main

import (
	"errors"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	updateCategoriesInterval    = time.Hour
	updateSubcategoriesInterval = time.Hour
	updateServicesInterval      = time.Hour
)

func InitDB() (*gorm.DB, error) {
	dsn := "host=localhost user=postgres password=gopher dbname=boostbot port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&UserState{}, &Category{}, &Subcategory{}, &Service{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Get user state
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

// Update user subscription status
func UpdateUserSubscriptionStatus(db *gorm.DB, userID, channelID int64, subscribed bool) error {
	userState, err := GetUserState(db, userID, channelID, true)
	if err != nil {
		return err
	}

	if userState.Subscribed != subscribed {
		userState.Subscribed = subscribed
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
		// <-done
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

func GetServicesFromDB(db *gorm.DB) ([]Service, error) {
	var services []Service
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
func GetServicesBySubcategoryID(db *gorm.DB, subcategoryID string) ([]Service, error) {
	var services []Service
	if err := db.Where("category_id = ?", subcategoryID).Find(&services).Error; err != nil {
		return nil, err
	}
	return services, nil
}

// Get service by service ID
func GetServiceByID(db *gorm.DB, serviceID string) (Service, error) {
	var service Service
	result := db.First(&service, "service_id = ?", serviceID)
	return service, result.Error
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

func updateService(tx *gorm.DB, newService Service) error {
	var existingService Service
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
		existingService.Type != newService.Type ||
		existingService.ServiceType != newService.ServiceType ||
		existingService.AverageTimestamp != newService.AverageTimestamp ||
		existingService.CreatedAt != newService.CreatedAt ||
		existingService.UpdatedAt != newService.UpdatedAt {
		return tx.Model(&existingService).Updates(Service{ID: newService.ID, Name: newService.Name, CategoryID: newService.CategoryID, Min: newService.Min,
			Max: newService.Max, Dripfeed: newService.Dripfeed, Refill: newService.Refill, Cancel: newService.Cancel, ServiceID: newService.ServiceID, Rate: newService.Rate,
			Type: newService.Type, ServiceType: newService.ServiceType, AverageTimestamp: newService.AverageTimestamp,
			CreatedAt: newService.CreatedAt, UpdatedAt: newService.UpdatedAt}).Error
	}

	return nil
}
