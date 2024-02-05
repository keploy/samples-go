// Package routes implements the router function
package routes

import (
	"S3-Keploy/bucket"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type Bucket struct {
	BucketName string `json:"name"`
}

func Register(app *fiber.App, awsService bucket.Basics) {
	app.Get("/list", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"buckets": awsService.ListAllBuckets(),
		})
	})

	app.Delete("/delete", func(c *fiber.Ctx) error {
		m := c.Queries()
		// return c.SendString(awsService.deleteOneBucket(m["bucket"]))
		return c.JSON(fiber.Map{
			"msg": awsService.DeleteOneBucket(m["bucket"]),
		})
	})

	app.Post("/create", func(c *fiber.Ctx) error {
		bucket := new(Bucket)
		if err := c.BodyParser(bucket); err != nil {
			fmt.Println("error = ", err)
			return c.SendStatus(200)
		}
		// return c.SendString(awsService.createOneBucket(bucket.BucketName))
		return c.JSON(fiber.Map{
			"msg": awsService.CreateOneBucket(bucket.BucketName),
		})
	})

	app.Post("/upload", func(c *fiber.Ctx) error {
		file, err := c.FormFile("filename")
		m := c.Queries()
		if err != nil {
			fmt.Println(err)
		}

		return c.JSON(awsService.UploadFile(file.Filename, m["bucket"]))
	})

	app.Get("/getallobjects", func(c *fiber.Ctx) error {
		m := c.Queries()
		return c.JSON(fiber.Map{
			"buckets": awsService.GetAllObjects(m["bucket"]),
		})
	})

	app.Delete("/deleteallobjects", func(c *fiber.Ctx) error {
		m := c.Queries()
		return c.JSON(fiber.Map{
			"msg": awsService.DeleteAllObjects(m["bucket"]),
		})
	})

	app.Put("/replacefile", func(c *fiber.Ctx) error {
		file, err := c.FormFile("filename")
		m := c.Queries()
		if err != nil {
			fmt.Println(err)
		}

		return c.JSON(awsService.UploadFile(file.Filename, m["bucket"]))
	})
}
