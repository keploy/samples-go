package services

import (
	"encoding/json"
	"errors"

	"github.com/keploy/gin-redis/helpers/redis"
	"github.com/keploy/gin-redis/structure"
	requeststruct "github.com/keploy/gin-redis/structure/request"
)

func Verifycode(request requeststruct.OTPRequest) (string, error) {
	jsonData, err := redis.Session().Get(request.Email).Result()
	if err != nil {
		return "", err
	}
	var data structure.OTPData
	err = json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		return "", err
	}
	if data.OTP != request.OTP {
		return "", errors.New("OTP didn't match, email isn't verified")
	}

	return data.UserName, nil
}
