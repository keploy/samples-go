package main

import (
	"net/http"
	"unicode"

	"github.com/gin-gonic/gin"
)

type UserInformation struct {
	Name string
	Age  int
}

func IsValidName(s string) bool {
	if len(s) > 100 {
		return false
	}
	for _, r := range s {
		if r != ' ' && !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func IsValidAge(age int) bool {
	return age > 0
}

func validateInformation(ctx *gin.Context) {
	var user UserInformation
	if err := ctx.BindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
	}

	if !IsValidName(user.Name) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid name"})
		return
	}

	if !IsValidAge(user.Age) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid age"})
		return
	}

	ctx.JSON(200, nil)
}
