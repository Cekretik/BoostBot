package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	apiCategoriesEndpoint     = "https://api.stagesmm.com/categories"
	apiSubcategoriesEndpoint  = "https://api.stagesmm.com/subcategories/"
	apiServicesEndpointFormat = "https://api.stagesmm.com/services?search=&limit=25000&subcategory_id=%s&pagination=1&order=DESC&order_by=id"
)

func fetchCategoriesFromAPI() ([]Category, error) {
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

	var categories []Category
	err = json.Unmarshal(body, &categories)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

func fetchSubcategoriesFromAPI(categoryID string) ([]Subcategory, error) {
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

	var subcategories []Subcategory
	err = json.Unmarshal(body, &subcategories)
	if err != nil {
		return nil, err
	}

	return subcategories, nil
}

func fetchServicesFromAPI(subcategoryID string) ([]Service, error) {
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

	fmt.Println(string(body))

	var services []Service
	err = json.Unmarshal(body, &services)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return nil, err
	}

	return services, nil
}
