package constants

import "time"

const API_SUCCESS_STATUS = "Success"
const API_FAILED_STATUS = "Failed"
const LOGIN_TOKEN_VALID_TIME_SEC = 155520000
const CacheTtlVeryLong = time.Second * 86400 * 7

const ApiFailStatus = "Fail"

var INVALID_TOKEN_RESPONSE = map[string]interface{}{
	"status":  ApiFailStatus,
	"message": "Invalid or No Token",
}
var IndexElasticSearch = map[string]string{
	"A": "forms",
	"B": "responses",
	"C": "answers",
	"D": "questions",
}
