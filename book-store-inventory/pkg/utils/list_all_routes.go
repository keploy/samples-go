package utils

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// https://stackoverflow.com/questions/287871/how-do-i-print-colored-text-to-the-terminal
// https://stackoverflow.com/a/26445590
var Reset = "\033[0m"

var Blue = "\033[34m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Red = "\033[31m"

var BlueBg = "\033[44m"
var GreenBg = "\033[42m"
var YellowBg = "\033[43m"
var RedBg = "\033[41m"
var MagentaBg = "\033[45m"

var Magenta = "\033[35m"

var Cyan = "\033[36m"

// var Gray = "\033[37m"
// var White = "\033[97m"

func ListAllAvailableRoutes(rtr *gin.Engine) {

	fmt.Println()
	// fmt.Println("============================================")
	fmt.Println("==============" + Cyan + "Available Routes" + Reset + "==============")
	fmt.Println()

	// https://github.com/gin-gonic/gin/issues/569
	for _, item := range rtr.Routes() {
		color := Blue
		switch item.Method {
		case "GET":
			// color = Blue
			color = BlueBg
		case "POST":
			// color = Green
			color = GreenBg
		case "PUT":
			// color = Yellow
			color = YellowBg
		case "DELETE":
			// color = Red
			color = RedBg
		case "PATCH":
			color = MagentaBg
		}
		fmt.Println("|"+color+" "+item.Method+" "+Reset+"|", item.Path)
		// println("method:", item.Method, "path:", item.Path)
	}
	fmt.Println()
	fmt.Println("============================================")
	fmt.Println()
}
