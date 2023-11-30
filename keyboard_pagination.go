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

func calculatePageRange(totalItems, itemsPerPage int, currentPage string, totalPages int) (startIndex, endIndex int) {
	pageIndex, _ := strconv.Atoi(currentPage)
	if pageIndex < 1 {
		pageIndex = 1
	} else if pageIndex > totalPages {
		pageIndex = totalPages
	}

	startIndex = (pageIndex - 1) * itemsPerPage
	endIndex = startIndex + itemsPerPage
	if endIndex > totalItems {
		endIndex = totalItems
	}

	return startIndex, endIndex
}

func HandleCallbackQuery(bot *tgbotapi.BotAPI, db *gorm.DB, callbackQuery *tgbotapi.CallbackQuery, totalPages int) {
	if strings.HasPrefix(callbackQuery.Data, "category:") {
		categoryID := strings.TrimPrefix(callbackQuery.Data, "category:")

		totalPages, err := GetTotalPagesForCategory(db, itemsPerPage, categoryID)
		if err != nil {
			log.Println("Error calculating total pages:", err)
			return
		}

		var currentPage string
		if strings.Contains(callbackQuery.Data, "prev:") || strings.Contains(callbackQuery.Data, "next:") {
			parts := strings.Split(callbackQuery.Data, ":")
			currentPage = parts[2]
			if strings.Contains(callbackQuery.Data, "prev:") {
				prevPage, _ := strconv.Atoi(currentPage)
				if prevPage > 1 {
					prevPage--
				}
				currentPage = strconv.Itoa(prevPage)
			} else if strings.Contains(callbackQuery.Data, "next:") {
				nextPage, _ := strconv.Atoi(currentPage)
				if nextPage < totalPages {
					nextPage++
				}
				currentPage = strconv.Itoa(nextPage)
			}
		} else {
			currentPage = "1"
		}

		keyboard, err := CreatePromotionKeyboard(db, true, categoryID, currentPage, strconv.Itoa(totalPages))
		if err != nil {
			log.Println("Error creating promotion keyboard:", err)
			return
		}

		editMsg := tgbotapi.NewEditMessageReplyMarkup(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, keyboard)
		_, err = bot.Send(editMsg)
		if err != nil {
			log.Println("Error sending edit message reply markup:", err)
		}
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

func createPaginationRow(categoryID, currentPage, totalPages string) tgbotapi.InlineKeyboardMarkup {
	var buttons []tgbotapi.InlineKeyboardButton
	currentPageInt, err := strconv.Atoi(currentPage)
	if err != nil {
		log.Println("Error parsing current page:", err)
		currentPageInt = 1
	}
	totalPagesInt, _ := strconv.Atoi(totalPages)

	// Кнопка "назад"
	if currentPageInt > 1 {
		prevPage := strconv.Itoa(currentPageInt - 1)
		prevButton := tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("category:%s:prev:%s", categoryID, prevPage))
		buttons = append(buttons, prevButton)
	}

	// Кнопка "вперед"
	if currentPageInt < totalPagesInt {
		nextPage := strconv.Itoa(currentPageInt + 1)
		nextButton := tgbotapi.NewInlineKeyboardButtonData("Вперед ➡️", fmt.Sprintf("category:%s:next:%s", categoryID, nextPage))
		buttons = append(buttons, nextButton)
	}

	// Информация о текущей странице
	pageInfo := fmt.Sprintf("Страница %d из %d", currentPageInt, totalPagesInt)
	pageInfoButton := tgbotapi.NewInlineKeyboardButtonData(pageInfo, "page_info")
	buttons = append(buttons, pageInfoButton)

	// Создаем строку с кнопками
	row := []tgbotapi.InlineKeyboardButton(buttons)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)

	return keyboard
}
