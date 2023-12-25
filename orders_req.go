package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

var apiOrdersEndpoint string
var token string

func init() {
	// Загрузка переменных окружения
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Инициализация глобальных переменных
	apiOrdersEndpoint = os.Getenv("API_ORDERS_ENDPOINT")
	token = os.Getenv("STAGESMM_TOKEN")
}
func fetchOrders() ([]ServiceDetails, error) {

	client := &http.Client{}
	req, err := http.NewRequest("GET", apiOrdersEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var serviceDetails []ServiceDetails
	err = json.Unmarshal(body, &serviceDetails)
	if err != nil {
		return nil, err
	}

	return serviceDetails, nil
}

func createOrder(order Order, token string) (UserOrders, error) {
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
		return UserOrders{}, err
	}

	req, err := http.NewRequest("POST", apiOrdersEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return UserOrders{}, err
	}

	req.Header.Add("Authorization", token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return UserOrders{}, err
	}
	defer resp.Body.Close()

	var responseOrder UserOrders
	if err := json.NewDecoder(resp.Body).Decode(&responseOrder); err != nil {
		return UserOrders{}, err
	}

	return responseOrder, nil
}
