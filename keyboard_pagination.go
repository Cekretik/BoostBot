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

		totalPages, err := GetTotalPagesForSubcategory(db, itemsPerPage, categoryID)
		if err != nil {
			log.Println("Error calculating total pages:", err)
			return
		}
		keyboard, err := CreatePromotionKeyboard(db, true, categoryID, "1", strconv.Itoa(totalPages))
		if err != nil {
			log.Println("Error creating promotion keyboard:", err)
			return
		}
		editMsg := tgbotapi.NewEditMessageReplyMarkup(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, keyboard)
		bot.Send(editMsg)

	} else if strings.HasPrefix(callbackQuery.Data, "prev:") || strings.HasPrefix(callbackQuery.Data, "next:") {
		parts := strings.Split(callbackQuery.Data, ":")
		categoryID, currentPage := parts[1], parts[2]
		if strings.HasPrefix(callbackQuery.Data, "prev:") {

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

		totalPages, err := GetTotalPagesForSubcategory(db, itemsPerPage, categoryID)
		if err != nil {
			log.Println("Error calculating total pages:", err)
			return
		}
		keyboard, err := CreatePromotionKeyboard(db, true, categoryID, currentPage, strconv.Itoa(totalPages))
		if err != nil {
			log.Println("Error creating promotion keyboard:", err)
			return
		}
		editMsg := tgbotapi.NewEditMessageReplyMarkup(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, keyboard)
		bot.Send(editMsg)
	}
}

func GetTotalPagesForSubcategory(db *gorm.DB, itemsPerPage int, categoryID string) (int, error) {
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
func createPaginationRow(categoryID, currentPage, totalPages string) []tgbotapi.InlineKeyboardButton {
	prevButton := tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("prev:%s:%s", categoryID, currentPage))
	nextButton := tgbotapi.NewInlineKeyboardButtonData("➡️ Вперед", fmt.Sprintf("next:%s:%s", categoryID, currentPage))
	infoText := fmt.Sprintf("Страница %s из %s", currentPage, totalPages)
	infoButton := tgbotapi.NewInlineKeyboardButtonData(infoText, "info:page")

	row := []tgbotapi.InlineKeyboardButton{prevButton, infoButton, nextButton}
	return row
}
