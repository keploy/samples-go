package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func TestApi(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "API Working 💞 [Private route]",
	})
}
