package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"your-project/internal/services"
)

type RatingHandler struct {
	service *services.RatingService
}

func NewRatingHandler(service *services.RatingService) *RatingHandler {
	return &RatingHandler{service: service}
}

func (h *RatingHandler) Create(c *fiber.Ctx) error {
	idStr := c.Params("id")
	productID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid product ID",
		})
	}

	var request struct {
		Score float64 `json:"score" validate:"required,min=1,max=5"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := h.service.CreateRating(productID, request.Score); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Rating added successfully",
	})
}

func (h *RatingHandler) GetByProduct(c *fiber.Ctx) error {
	idStr := c.Params("id")
	productID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid product ID",
		})
	}

	ratings, err := h.service.GetRatingsByProduct(productID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	summary, err := h.service.GetRatingSummary(productID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"ratings": ratings,
		"summary": summary,
	})
}