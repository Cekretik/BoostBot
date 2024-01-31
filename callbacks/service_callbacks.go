package callbacks

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

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
