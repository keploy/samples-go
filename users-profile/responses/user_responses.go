// Creating a reusable struct to describe our API’s response.
package responses

// json:"status", json:"message", and json:"data" are known as struct tags.
// Struct tags allow us to attach meta-information to corresponding struct properties.
// In other words, we use them to reformat the JSON response returned by the API.
type UserResponse struct {
	Status  int                    `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}
