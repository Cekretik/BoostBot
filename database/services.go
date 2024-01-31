package database

import (
	"errors"
	"log"
	"time"

	"gorm.io/gorm"
)

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
				//log.Println("Categories updated in the database.")
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
			apiSubcategories, err := fetchSubcategoriesFromAPI(category.ID)
			if err != nil {
				log.Printf("Error fetching subcategories from API for category %s: %v", category.Name, err)
				continue
			}

			apiSubcategoryIDs := make(map[string]bool)
			for _, subcategory := range apiSubcategories {
				apiSubcategoryIDs[subcategory.ID] = true
				if err := updateSubcategory(db, subcategory); err != nil {
					log.Printf("Error updating subcategory with ID %s: %v", subcategory.ID, err)
				}
			}

			// Удаление подкатегорий, которые есть в БД, но отсутствуют в API
			var existingSubcategories []Subcategory
			db.Where("category_id = ?", category.ID).Find(&existingSubcategories)
			for _, existingSubcategory := range existingSubcategories {
				if _, found := apiSubcategoryIDs[existingSubcategory.ID]; !found {
					db.Delete(&existingSubcategory)
				}
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
			apiServices, err := fetchServicesFromAPI(subcategory.ID)
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

			// Список ID услуг из API для сравнения
			apiServiceIDs := make(map[string]bool)
			for _, service := range apiServices {
				apiServiceIDs[service.ServiceID] = true
				if err := updateService(tx, service); err != nil {
					log.Printf("Error updating service with ID %s: %v", service.ServiceID, err)
					tx.Rollback()
					break
				}
			}

			// Удаление услуг, которые есть в БД, но отсутствуют в API
			var existingServices []Services
			tx.Where("category_id = ?", subcategory.ID).Find(&existingServices)
			for _, existingService := range existingServices {
				if _, found := apiServiceIDs[existingService.ServiceID]; !found {
					tx.Delete(&existingService)
				}
			}

			if err := tx.Commit().Error; err != nil {
				log.Printf("Error committing transaction for subcategory %s: %v", subcategory.Name, err)
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
		}
		select {
		case <-done:
			return
		default:
			time.Sleep(30 * time.Minute)
		}
	}
}
