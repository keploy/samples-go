package email

import (
	"encoding/json"
	"time"

	"github.com/keploy/gin-redis/helpers/redis"
	"github.com/keploy/gin-redis/services"
	"github.com/keploy/gin-redis/structure"
)

func SendEmail(to, userName, subject string) (int, error) {
	otp := services.GenerateOTP()
	data := structure.OTPData{
		OTP:      otp,
		UserName: userName,
	}

	// Convert the OTPData struct to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}

	// Store the JSON string in Redis
	err = redis.RedisSession().Set(to, jsonData, 4*time.Hour).Err()
	return otp, err
}
