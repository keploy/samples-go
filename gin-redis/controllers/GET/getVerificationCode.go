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
	otp, err := email.SendEmail(requestEmail, constants.VerifyYourself)

	if err != nil {
		resp := responseStruct.SuccessResponse{}
		resp.Message = err.Error()
		resp.Status = "false"
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	resp := responseStruct.SuccessResponse{}
	resp.Message = fmt.Sprint(otp)
	resp.Status = "true"
	c.JSON(http.StatusOK, resp)
}
