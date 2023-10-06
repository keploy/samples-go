package POST

import (
	"net/http"
	"user-onboarding/constants"
	helpers "user-onboarding/helpers/userAction"
	structs "user-onboarding/struct"
	response "user-onboarding/struct/response"
	"user-onboarding/utils"

	"github.com/gin-gonic/gin"
)

func CreateUser(c *gin.Context) {
	formRequest := structs.SignUp{}

	if err := c.ShouldBind(&formRequest); err != nil {
		c.JSON(422, utils.SendErrorResponse(err))
		return
	}
	ctx := c.Request.Context()
	resp := response.SuccessResponse{}

	err := helpers.UserDetails(ctx, &formRequest) //the DAO level
	if err != nil {
		resp.Status = "Failed"
		resp.Message = err.Error()
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	token, tokenerror := utils.GenerateToken()

	if tokenerror != nil {
		resp.Status = constants.API_FAILED_STATUS
		resp.Message = "User Created,Please login"
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	resp.Status = "Success"
	resp.Message = "User created successfully"
	resp.Token = token

	c.JSON(http.StatusOK, resp)

}
