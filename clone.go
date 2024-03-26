package main

import (
	"log"
	"time"

	"github.com/Cekretik/BoostBot/models"
	tgbotapi "github.com/Cekretik/telegram-bot-api-master"
	"gorm.io/gorm"
)

var initializedBots map[string]bool = make(map[string]bool)

func InitializeBot(db *gorm.DB, token string) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Printf("Ошибка при инициализации бота с токеном %s: %v", token, err)
		if err := db.Model(&models.BotOwners{}).Where("token = ?", token).Update("running", false).Error; err != nil {
			log.Printf("Не удалось обновить статус бота: %v", err)
		}
		return
	}

	log.Printf("Бот с токеном %s успешно запущен", token)
	ProcessMessages(bot, db)
}

func LoadActiveTokens(db *gorm.DB) ([]models.BotOwners, error) {
	var activeBots []models.BotOwners
	err := db.Where("running = ?", true).Find(&activeBots).Error
	if err != nil {
		return nil, err
	}
	return activeBots, nil
}

func RunBots(db *gorm.DB) {
	go func() {
		for token := range NewTokensChannel {
			if _, ok := initializedBots[token]; !ok {
				InitializeBot(db, token)
				initializedBots[token] = true
			}
		}
	}()

	for {
		activeBots, err := LoadActiveTokens(db)
		if err != nil {
			log.Printf("Ошибка при загрузке активных ботов: %v", err)
			time.Sleep(10 * time.Second) // Пауза перед следующей попыткой
			continue
		}

		for _, botOwner := range activeBots {
			if _, ok := initializedBots[botOwner.Token]; !ok {
				go InitializeBot(db, botOwner.Token)
				initializedBots[botOwner.Token] = true
			}
		}

		time.Sleep(10 * time.Minute)
	}
}

func CountUserBots(db *gorm.DB, userID int64) (int64, error) {
	var count int64
	err := db.Model(&models.BotOwners{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}
