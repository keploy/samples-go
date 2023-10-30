package services

import (
	"errors"
	"strconv"

	"github.com/keploy/gin-redis/helpers/redis"
	requestStruct "github.com/keploy/gin-redis/structure/request"
)

func Verifycode(request requestStruct.OTPRequest) error {
	otp, err := redis.RedisSession().Get(request.Email).Result()
	if err != nil {
		return errors.New("Couldn't find any related OTP, kindly re-enter")
	}

	if otp != strconv.Itoa(request.OTP) {
		return errors.New("OTP didn't match, email isn't verified")
	}

	return nil
}
