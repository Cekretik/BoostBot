package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

var itemsPerPage = 10

func HandleCallbackQuery(bot *tgbotapi.BotAPI, db *gorm.DB, callbackQuery *tgbotapi.CallbackQuery, totalPages int) {
	if strings.HasPrefix(callbackQuery.Data, "category:") {
		categoryID := strings.TrimPrefix(callbackQuery.Data, "category:")

		totalPages, err := GetTotalPagesForCategory(db, itemsPerPage, categoryID)
		if err != nil {
			log.Println("Error calculating total pages:", err)
			return
		}
		keyboard, err := CreateSubcategoryKeyboard(db, categoryID, "1", strconv.Itoa(totalPages))
		if err != nil {
			log.Println("Error creating promotion keyboard:", err)
			return
		}
		editMsg := tgbotapi.NewEditMessageReplyMarkup(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, keyboard)
		bot.Send(editMsg)

	} else if strings.HasPrefix(callbackQuery.Data, "prevCat:") || strings.HasPrefix(callbackQuery.Data, "nextCat:") {
		parts := strings.Split(callbackQuery.Data, ":")
		categoryID, currentPage := parts[1], parts[2]
		if strings.HasPrefix(callbackQuery.Data, "prevCat:") {

			prevPage, _ := strconv.Atoi(currentPage)
			if prevPage > 1 {
				prevPage--
			}
			currentPage = strconv.Itoa(prevPage)
		} else {

			nextPage, _ := strconv.Atoi(currentPage)
			nextPage++
			currentPage = strconv.Itoa(nextPage)
		}

		totalPages, err := GetTotalPagesForCategory(db, itemsPerPage, categoryID)
		if err != nil {
			log.Println("Error calculating total pages:", err)
			return
		}
		keyboard, err := CreateSubcategoryKeyboard(db, categoryID, currentPage, strconv.Itoa(totalPages))
		if err != nil {
			log.Println("Error creating promotion keyboard:", err)
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

		editMsg := tgbotapi.NewEditMessageReplyMarkup(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, keyboard)
		bot.Send(editMsg)

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
		} else if action == "nextServ" {
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
	if err := db.Model(&Service{}).Where("category_id = ?", subcategoryID).Count(&totalServices).Error; err != nil {
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
