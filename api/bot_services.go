package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Cekretik/BoostBot/models"
)

const (
	apiCategoriesEndpoint     = "https://api.stagesmm.com/categories"
	apiSubcategoriesEndpoint  = "https://api.stagesmm.com/subcategories/"
	apiServicesEndpointFormat = "https://api.stagesmm.com/services?search=&limit=25000&subcategory_id=%s&pagination=1&order=DESC&order_by=id"
)

func FetchCategoriesFromAPI() ([]models.Category, error) {
	resp, err := http.Get(apiCategoriesEndpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	//fmt.Println(string(body))

	var categories []models.Category
	err = json.Unmarshal(body, &categories)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

func FetchSubcategoriesFromAPI(categoryID string) ([]models.Subcategory, error) {
	resp, err := http.Get(apiSubcategoriesEndpoint + categoryID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	//fmt.Println(string(body))

	var subcategories []models.Subcategory
	err = json.Unmarshal(body, &subcategories)
	if err != nil {
		return nil, err
	}

	return subcategories, nil
}

func FetchServicesFromAPI(subcategoryID string) ([]models.Services, error) {
	apiUrl := fmt.Sprintf(apiServicesEndpointFormat, subcategoryID)
	resp, err := http.Get(apiUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// fmt.Println(string(body))

	var response struct {
		Services []models.Services `json:"services"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return nil, err
	}

	return response.Services, nil
}
