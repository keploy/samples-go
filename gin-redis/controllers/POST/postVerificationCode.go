package post

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/keploy/gin-redis/config"
	"github.com/keploy/gin-redis/helpers/token"
	"github.com/keploy/gin-redis/services"
	requestStruct "github.com/keploy/gin-redis/structure/request"
	responseStruct "github.com/keploy/gin-redis/structure/response"
	"github.com/keploy/gin-redis/utils"
)

func VerifyCode(c *gin.Context) {
	verifyRequest := requestStruct.OTPRequest{}
	if err := c.ShouldBind(&verifyRequest); err != nil {
		c.JSON(422, utils.SendErrorResponse(err))
		return
	}

	err := services.Verifycode(verifyRequest)

	if err != nil {
		resp := responseStruct.SuccessResponse{}
		resp.Message = err.Error()
		resp.Status = "false"
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	resp := responseStruct.SuccessResponse{}

	requesterEmail := strings.Split(verifyRequest.Email, "@")
	resp.Message, err = token.GenerateToken(requesterEmail[1], config.Get().JWT)
	if err != nil {
		resp := responseStruct.SuccessResponse{}
		resp.Message = err.Error()
		resp.Status = "false"
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	resp.Status = "true"
	c.JSON(http.StatusOK, resp)
}
