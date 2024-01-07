package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

var itemsPerPage = 10

func HandleCallbackQuery(bot *tgbotapi.BotAPI, db *gorm.DB, callbackQuery *tgbotapi.CallbackQuery, totalPages int) {
	parts := strings.Split(callbackQuery.Data, ":")
	action := parts[0]

	if action == "category" {
		categoryID := parts[1]
		totalPages, err := GetTotalPagesForCategory(db, itemsPerPage, categoryID)
		if err != nil {
			log.Println("Error calculating total pages:", err)
			return
		}

		keyboard, err := CreateSubcategoryKeyboard(db, categoryID, "1", strconv.Itoa(totalPages))
		if err != nil {
			log.Println("Error creating subcategory keyboard:", err)
			return
		}

		deleteMsg := tgbotapi.NewDeleteMessage(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID)
		bot.Send(deleteMsg)

		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "Выберите категорию:")
		msg.ReplyMarkup = keyboard
		bot.Send(msg)

	} else if action == "prevCat" || action == "nextCat" {
		currentPage, _ := strconv.Atoi(parts[2])
		if action == "prevCat" && currentPage > 1 {
			currentPage--
		} else if action == "nextCat" && currentPage < totalPages {
			currentPage++
		}
		totalPages, err := GetTotalPagesForCategory(db, itemsPerPage, parts[1])
		if err != nil {
			log.Println("Error recalculating total pages:", err)
			return
		}
		keyboard, err := CreateSubcategoryKeyboard(db, parts[1], strconv.Itoa(currentPage), strconv.Itoa(totalPages))
		if err != nil {
			log.Println("Error updating subcategory keyboard:", err)
			return
		}
		editMsg := tgbotapi.NewEditMessageReplyMarkup(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, keyboard)
		bot.Send(editMsg)
	}
}

func HandleServiceCallBackQuery(bot *tgbotapi.BotAPI, db *gorm.DB, callbackQuery *tgbotapi.CallbackQuery, totalServicePages int) {
	if strings.HasPrefix(callbackQuery.Data, "subcategory:") {
		subcategoryID := strings.TrimPrefix(callbackQuery.Data, "subcategory:")

		totalServicePages, err := GetTotalPagesForService(db, itemsPerPage, subcategoryID)
		if err != nil {
			log.Printf("Error calculating total pages for subcategory '%s': %v", subcategoryID, err)
			return
		}

		keyboard, err := CreateServiceKeyboard(db, subcategoryID, "1", strconv.Itoa(totalServicePages))
		if err != nil {
			log.Printf("Error creating service keyboard for subcategory '%s': %v", subcategoryID, err)
			return
		}

		deleteMsg := tgbotapi.NewDeleteMessage(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID)
		bot.Send(deleteMsg)

		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "Выберите услугу:")
		msg.ReplyMarkup = keyboard
		bot.Send(msg)

	} else if strings.HasPrefix(callbackQuery.Data, "prevServ:") || strings.HasPrefix(callbackQuery.Data, "nextServ:") {
		parts := strings.Split(callbackQuery.Data, ":")
		action, subcategoryID, currentPageStr := parts[0], parts[1], parts[2]
		currentPage, err := strconv.Atoi(currentPageStr)
		if err != nil {
			log.Printf("Error converting currentPage to integer: %v", err)
			return
		}

		if action == "prevServ" && currentPage > 1 {
			currentPage--
		} else if action == "nextServ" && currentPage < totalServicePages {
			currentPage++
		}

		totalServicePages, err := GetTotalPagesForService(db, itemsPerPage, subcategoryID)
		if err != nil {
			log.Printf("Error recalculating total pages for subcategory '%s': %v", subcategoryID, err)
			return
		}

		keyboard, err := CreateServiceKeyboard(db, subcategoryID, strconv.Itoa(currentPage), strconv.Itoa(totalServicePages))
		if err != nil {
			log.Printf("Error updating service keyboard for subcategory '%s', page %d: %v", subcategoryID, currentPage, err)
			return
		}

		editMsg := tgbotapi.NewEditMessageReplyMarkup(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, keyboard)
		bot.Send(editMsg)

	} else if strings.HasPrefix(callbackQuery.Data, "serviceInfo:") {
		serviceIDStr := strings.TrimPrefix(callbackQuery.Data, "serviceInfo:")
		var service Services
		userID := callbackQuery.Message.Chat.ID
		deleteMsg := tgbotapi.NewDeleteMessage(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID)
		bot.Send(deleteMsg)

		service, err := GetServiceByID(db, serviceIDStr)
		if err != nil {
			log.Printf("Error getting service '%s': %v", service.Name, err)
			return
		}

		subcategory, err := GetSubcategoryByID(db, service.CategoryID)
		if err != nil {
			log.Printf("Error getting subcategory '%s': %v", subcategory.Name, err)
			return
		}

		favoriteButtonText := "✅Добавить в избранное"
		favoriteCallbackData := fmt.Sprintf("addFavorite:%d", service.ID)

		removeFavoriteButtonText := "❌Удалить из избранного"
		removeFavoriteCallbackData := fmt.Sprintf("removeFavorite:%d", service.ID)

		increasePercent, err := strconv.ParseFloat(os.Getenv("PRICE_PERCENT"), 64)
		if err != nil {
			increasePercent = 0 // или установите значение по умолчанию
		}
		userCurrency, err := getUserCurrency(db, userID)
		if err != nil {
			log.Printf("Error getting user currency: %v", err)
			return
		}
		currencyRate := getCurrentCurrencyRate()
		msgText := FormatServiceInfo(service, subcategory, increasePercent, userCurrency, currencyRate)
		backData := "backToServices:" + service.CategoryID
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙Вернуться к услугам", backData),
				tgbotapi.NewInlineKeyboardButtonData("➕Заказать", "order:"+strconv.Itoa(service.ID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(removeFavoriteButtonText, removeFavoriteCallbackData),
				tgbotapi.NewInlineKeyboardButtonData(favoriteButtonText, favoriteCallbackData),
			),
		)

		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, msgText)
		msg.ReplyMarkup = keyboard
		bot.Send(msg)

	} else if strings.HasPrefix(callbackQuery.Data, "order:") {
		serviceID := strings.TrimPrefix(callbackQuery.Data, "order:")
		service, err := GetServiceByID(db, serviceID)
		if err != nil {
			log.Printf("Error getting service '%s': %v", serviceID, err)
			bot.Send(tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "Ошибка при получении данных сервиса."))
			return
		}
		handleOrderCommand(bot, callbackQuery.Message.Chat.ID, service)
	} else if strings.HasPrefix(callbackQuery.Data, "backToServices:") {
		subcategoryID := strings.TrimPrefix(callbackQuery.Data, "backToServices:")
		deleteMsg := tgbotapi.NewDeleteMessage(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID)
		bot.Send(deleteMsg)

		totalServicePages, err := GetTotalPagesForService(db, itemsPerPage, subcategoryID)
		if err != nil {
			log.Printf("Error getting total pages for services: %v", err)
			return
		}

		keyboard, err := CreateServiceKeyboard(db, subcategoryID, "1", strconv.Itoa(totalServicePages))
		if err != nil {
			log.Printf("Error creating service keyboard for subcategory '%s': %v", subcategoryID, err)
			return
		}

		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "Выберите услугу:")
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
	} else if strings.HasPrefix(callbackQuery.Data, "backToSubcategories:") {
		// Получение ID подкатегории из данных callback
		subcategoryID := strings.TrimPrefix(callbackQuery.Data, "backToSubcategories:")

		// Получение объекта подкатегории по ID
		subcategory, err := GetSubcategoryByID(db, subcategoryID)
		if err != nil {
			log.Printf("Error getting subcategory '%s': %v", subcategoryID, err)
			return
		}

		deleteMsg := tgbotapi.NewDeleteMessage(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID)
		bot.Send(deleteMsg)

		// Получение списка подкатегорий для категории, к которой принадлежит текущая подкатегория
		totalPages, err := GetTotalPagesForCategory(db, itemsPerPage, subcategory.CategoryID)
		if err != nil {
			log.Printf("Error calculating total pages for category '%s': %v", subcategory.CategoryID, err)
			return
		}

		keyboard, err := CreateSubcategoryKeyboard(db, subcategory.CategoryID, "1", strconv.Itoa(totalPages))
		if err != nil {
			log.Printf("Error creating subcategory keyboard for category '%s': %v", subcategory.CategoryID, err)
			return
		}

		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "Выберите категорию:")
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
	}
}
func GetTotalPagesForCategory(db *gorm.DB, itemsPerPage int, categoryID string) (int, error) {
	var totalSubcategories int64
	if err := db.Model(&Subcategory{}).Where("category_id = ?", categoryID).Count(&totalSubcategories).Error; err != nil {
		return 0, err
	}

	if totalSubcategories == 0 {
		return 0, nil
	}

	totalPages := int(totalSubcategories) / itemsPerPage
	if int(totalSubcategories)%itemsPerPage != 0 {
		totalPages++
	}

	return totalPages, nil
}

func GetTotalPagesForService(db *gorm.DB, itemsPerPage int, subcategoryID string) (int, error) {
	var totalServices int64
	if err := db.Model(&Services{}).Where("category_id = ?", subcategoryID).Count(&totalServices).Error; err != nil {
		return 0, err
	}

	if totalServices == 0 {
		return 0, nil
	}

	totalServicePages := int(totalServices) / itemsPerPage
	if int(totalServices)%itemsPerPage != 0 {
		totalServicePages++
	}

	return totalServicePages, nil
}

func calculatePageRange(totalItems, itemsPerPage int, currentPage string) (startIndex, endIndex int) {
	pageIndex := 1
	if currentPage != "" {
		pageIndex, _ = strconv.Atoi(currentPage)
	}

	startIndex = (pageIndex - 1) * itemsPerPage
	endIndex = startIndex + itemsPerPage
	if endIndex > totalItems {
		endIndex = totalItems
	}
	return startIndex, endIndex
}
func createPaginationRow(categoryID string, currentPage int, totalPages int) []tgbotapi.InlineKeyboardButton {
	var paginationRow []tgbotapi.InlineKeyboardButton
	if currentPage > 1 {
		prevButton := tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("prevCat:%s:%d", categoryID, currentPage))
		paginationRow = append(paginationRow, prevButton)
	}
	pageInfoButton := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Страница %d из %d", currentPage, totalPages), "page_info")
	paginationRow = append(paginationRow, pageInfoButton)
	if currentPage < totalPages {
		nextButton := tgbotapi.NewInlineKeyboardButtonData("➡️ Вперед", fmt.Sprintf("nextCat:%s:%d", categoryID, currentPage))
		paginationRow = append(paginationRow, nextButton)
	}

	return paginationRow
}

func createServicePaginationRow(subcategoryID string, currentPage int, totalServicePages int) []tgbotapi.InlineKeyboardButton {
	var paginationRow []tgbotapi.InlineKeyboardButton
	if currentPage > 1 {
		prevButton := tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("prevServ:%s:%d", subcategoryID, currentPage))
		paginationRow = append(paginationRow, prevButton)
	}
	pageInfoButton := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("Страница %d из %d", currentPage, totalServicePages), "page_info")
	paginationRow = append(paginationRow, pageInfoButton)
	if currentPage < totalServicePages {
		nextButton := tgbotapi.NewInlineKeyboardButtonData("➡️ Вперед", fmt.Sprintf("nextServ:%s:%d", subcategoryID, currentPage))
		paginationRow = append(paginationRow, nextButton)
	}

	return paginationRow
}

func handleAddToFavoritesCallback(bot *tgbotapi.BotAPI, db *gorm.DB, callbackQuery *tgbotapi.CallbackQuery) {
	parts := strings.Split(callbackQuery.Data, ":")
	action := parts[0]
	serviceIDStr := parts[1]

	// Преобразование строки ID в int
	serviceID, err := strconv.ParseInt(serviceIDStr, 10, 64)
	if err != nil {
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callbackQuery.ID, "Ошибка: неверный идентификатор услуги"))
		return
	}

	userID := callbackQuery.Message.Chat.ID

	// Получение объекта услуги из базы данных
	var service Services
	if err := db.First(&service, serviceID).Error; err != nil {
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callbackQuery.ID, "Услуга не найдена"))
		return
	}

	var responseText string
	if action == "addFavorite" {
		err = AddServiceToFavorites(db, userID, service.ID)
		responseText = "Услуга добавлена в избранное"
	} else if action == "removeFavorite" {
		err = RemoveServiceFromFavorites(db, userID, service.ID)
		responseText = "Услуга удалена из избранного"
	}

	if err != nil {
		bot.AnswerCallbackQuery(tgbotapi.NewCallback(callbackQuery.ID, "Ошибка при обновлении избранных услуг"))
		return
	}

	bot.AnswerCallbackQuery(tgbotapi.NewCallback(callbackQuery.ID, responseText))
}
