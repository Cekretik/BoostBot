package callbacks

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

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
