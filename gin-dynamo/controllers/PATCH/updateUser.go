package PATCH

import (
	"net/http"
	helpers "user-onboarding/helpers/userAction"
	structs "user-onboarding/struct"
	response "user-onboarding/struct/response"
	"user-onboarding/utils"

	"github.com/gin-gonic/gin"
)

func UpdateUser(c *gin.Context) {
	formRequest := structs.UserDetails{}

	if err := c.ShouldBind(&formRequest); err != nil {
		c.JSON(422, utils.SendErrorResponse(err))
		return
	}
	ctx := c.Request.Context()
	resp := response.SuccessResponse{}

	err := helpers.UpdateUserDetails(ctx, &formRequest) //the DAO level
	if err != nil {
		resp.Status = "Failed"
		resp.Message = err.Error()
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	resp.Status = "Success"
	resp.Message = "User details updated successfully"

	c.JSON(http.StatusOK, resp)

}
