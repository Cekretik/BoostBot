package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Cekretik/BoostBot/models"
	"github.com/joho/godotenv"
)

var apiOrdersEndpoint string
var Token string

func init() {
	// Загрузка переменных окружения
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Инициализация глобальных переменных
	apiOrdersEndpoint = os.Getenv("API_ORDERS_ENDPOINT")
	Token = os.Getenv("STAGESMM_TOKEN")
}
func FetchOrders() ([]models.ServiceDetails, error) {

	client := &http.Client{}
	req, err := http.NewRequest("GET", apiOrdersEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", Token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var serviceDetails []models.ServiceDetails
	err = json.Unmarshal(body, &serviceDetails)
	if err != nil {
		return nil, err
	}

	return serviceDetails, nil
}

func CreateOrder(order models.Order, token string) (models.UserOrders, error) {
	client := &http.Client{}
	// Создание данных для запроса из структуры Order
	data := map[string]interface{}{
		"id":            order.ID,
		"serviceId":     order.ServiceID,
		"link":          order.Link,
		"quantity":      order.Quantity,
		"keywords":      order.Keywords,
		"comments":      order.Comments,
		"usernames":     order.Usernames,
		"hashtags":      order.Hashtags,
		"hashtag":       order.Hashtag,
		"username":      order.Username,
		"answer_number": order.AnswerNumber,
		"min":           order.Min,
		"max":           order.Max,
		"delay":         order.Delay,
	}

	// Удаляем пустые или нулевые поля
	for key, value := range data {
		if val, ok := value.(string); ok && val == "" {
			delete(data, key)
		}
		if val, ok := value.(int); ok && val == 0 {
			delete(data, key)
		}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return models.UserOrders{}, err
	}

	req, err := http.NewRequest("POST", apiOrdersEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return models.UserOrders{}, err
	}

	req.Header.Add("Authorization", token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return models.UserOrders{}, err
	}
	defer resp.Body.Close()

	var responseOrder models.UserOrders
	if err := json.NewDecoder(resp.Body).Decode(&responseOrder); err != nil {
		return models.UserOrders{}, err
	}

	return responseOrder, nil
}

type RatesResponse struct {
	RUB float64 `json:"RUB"`
}

var CurrentRate float64

func GetCurrencyRate() (float64, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.stagesmm.com/rates", nil)
	if err != nil {
		return 0, err
	}

	req.Header.Add("Authorization", Token)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// log.Printf("Response body: %s", string(body))

	var rate float64
	rate, err = strconv.ParseFloat(strings.TrimSpace(string(body)), 64)
	if err != nil {
		log.Printf("Error parsing rate: %v", err)
		return 0, err
	}

	// log.Printf("Currency rate: %f", rate)
	return rate, nil
}

func UpdateCurrencyRatePeriodically() {
	for {
		rate, err := GetCurrencyRate()
		if err != nil {
			log.Printf("Error getting currency rate: %v", err)
		} else {
			CurrentRate = rate
			log.Printf("Updated currency rate: %f", CurrentRate)
		}
		time.Sleep(1 * time.Hour)
	}
}

func GetCurrentCurrencyRate() float64 {
	return CurrentRate
}
