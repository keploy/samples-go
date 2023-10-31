package get

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/keploy/gin-redis/constants"
	"github.com/keploy/gin-redis/helpers/email"
	responseStruct "github.com/keploy/gin-redis/structure/response"
)

func VerificationCode(c *gin.Context) {
	requestEmail := c.Query("email")
	userName := c.Query("username")
	otp, err := email.SendEmail(requestEmail, userName, constants.VerifyYourself)

	if err != nil {
		resp := responseStruct.SuccessResponse{}
		resp.Message = err.Error()
		resp.Status = "false"
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	resp := responseStruct.SuccessResponse{}
	resp.OTP = fmt.Sprint(otp)
	resp.Status = "true"
	resp.Message = "OTP Generated successfully"
	c.JSON(http.StatusOK, resp)
}
