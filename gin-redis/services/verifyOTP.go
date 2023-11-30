package services

import (
	"encoding/json"
	"errors"

	"github.com/keploy/gin-redis/helpers/redis"
	"github.com/keploy/gin-redis/structure"
	requestStruct "github.com/keploy/gin-redis/structure/request"
)

func Verifycode(request requestStruct.OTPRequest) (string, error) {
	jsonData, err := redis.RedisSession().Get(request.Email).Result()
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
