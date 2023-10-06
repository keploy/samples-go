package server

import (
	"user-onboarding/config"
	"user-onboarding/routes"
	aws "user-onboarding/services/s3Bucket"
)

func Init() {

	config := config.Get() //getting all the env configs
	aws.Init()
	r := routes.NewRouter()        //initialsing routes
	r.Run(":" + config.ServerPort) //running the server at port
}
