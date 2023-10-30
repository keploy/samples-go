package email

import (
	"time"

	"github.com/keploy/gin-redis/helpers/redis"
	"github.com/keploy/gin-redis/services"
)

func SendEmail(to, subject string) (int, error) {
	otp := services.GenerateOTP()
	err := redis.RedisSession().Set(to, otp, 4*time.Hour).Err()
	return otp, err
}
