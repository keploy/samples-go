package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"your-project/internal/services"
)

type TagHandler struct {
	service *services.TagService
}

func NewTagHandler(service *services.TagService) *TagHandler {
	return &TagHandler{service: service}
}

func (h *TagHandler) AddTags(c *fiber.Ctx) error {
	idStr := c.Params("id")
	productID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid product ID",
		})
	}

	var request struct {
		Tags []string `json:"tags" validate:"required"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := h.service.AddTags(productID, request.Tags); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Tags added successfully",
	})
}

func (h *TagHandler) GetByProduct(c *fiber.Ctx) error {
	idStr := c.Params("id")
	productID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid product ID",
		})
	}

	tags, err := h.service.GetTagsByProduct(productID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"tags": tags,
	})
}

func (h *TagHandler) GetProductsByTag(c *fiber.Ctx) error {
	tag := c.Params("tag")

	products, err := h.service.GetProductsByTag(tag)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(products)
}