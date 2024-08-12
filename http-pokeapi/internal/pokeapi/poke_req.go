package pokeapi

import (
	"encoding/json"
	"fmt"
	models "http-pokeapi/internal/models"
	"io"
	"net/http"
)

const base_url = "https://pokeapi.co/api/v2"

type Client struct {
	http.Client
}

func (client *Client) Pokemon(name string) (models.Pokemon, error) {
	end_url := "/pokemon/"
	full_url := base_url + end_url + name

	req, err := http.NewRequest("GET", full_url, nil)

	if err != nil {
		return models.Pokemon{}, err
	}

	res, err := client.Do(req)

	if err != nil {
		return models.Pokemon{}, err
	}

	if res.StatusCode > 399 {
		return models.Pokemon{}, fmt.Errorf("bad status code : %v", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return models.Pokemon{}, err
	}

	defer res.Body.Close()

	pokeinfo := models.Pokemon{}
	err = json.Unmarshal(data, &pokeinfo)
	if err != nil {
		return models.Pokemon{}, err
	}
	return pokeinfo, nil

}

func (client *Client) LocationArearesponse() (models.Location, error) {
	end_url := "/location-area"
	full_url := base_url + end_url

	req, err := http.NewRequest("GET", full_url, nil)

	if err != nil {
		return models.Location{}, err
	}

	res, err := client.Do(req)

	if err != nil {
		return models.Location{}, err
	}

	if res.StatusCode > 399 {
		return models.Location{}, fmt.Errorf("bad status code : %v", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return models.Location{}, err
	}

	defer res.Body.Close()

	LocationAreaValues := models.Location{}
	err = json.Unmarshal(data, &LocationAreaValues)
	if err != nil {
		return models.Location{}, err
	}
	return LocationAreaValues, nil

}

func (c *Client) Pokelocationres(arg string) (models.Pokelocation, error) {
	end_url := "/location-area/"
	full_url := base_url + end_url + arg

	req, err := http.NewRequest("GET", full_url, nil)

	if err != nil {
		return models.Pokelocation{}, err
	}

	res, err := c.Do(req)

	if err != nil {
		return models.Pokelocation{}, err
	}

	if res.StatusCode > 399 {
		return models.Pokelocation{}, fmt.Errorf("bad status code : %v", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return models.Pokelocation{}, err
	}

	defer res.Body.Close()

	LocationAreaValues := models.Pokelocation{}
	err = json.Unmarshal(data, &LocationAreaValues)
	if err != nil {
		return models.Pokelocation{}, err
	}
	return LocationAreaValues, nil

}
