package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

const (
	apiOrdersEndpoint string = "https://api.stagesmm.com/orders"
	token             string = "VTJGc2RHVmtYMThJU0JpbmIwT2FlTE4yK1JoMWR3OWdBWW56SkxRM09MTG5YeERpNzdVS1lEd21qcCsza25xRm1zVHBzK3gwY0dDVFdjN2FUQzFZZXNoamNVRENTRlBJTWpPcFV2QXBhaUE9"
)

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

func createOrder(order Order, token string) (Order, error) {
	client := &http.Client{}
	// Создание данных для запроса из структуры Order
	data := map[string]interface{}{
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
		return Order{}, err
	}

	req, err := http.NewRequest("POST", apiOrdersEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return Order{}, err
	}

	req.Header.Add("Authorization", token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return Order{}, err
	}
	defer resp.Body.Close()

	var responseOrder Order
	if err := json.NewDecoder(resp.Body).Decode(&responseOrder); err != nil {
		return Order{}, err
	}

	return responseOrder, nil
}
