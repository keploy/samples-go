package handlers

import (
	"github.com/gofiber/fiber/v2"
	"your-project/internal/services"
)

type CartHandler struct {
	service *services.CartService
}

func NewCartHandler(service *services.CartService) *CartHandler {
	return &CartHandler{service: service}
}

func (h *CartHandler) Update(c *fiber.Ctx) error {
	userID := c.Params("userId")
	
	var cart map[string]int
	if err := c.BodyParser(&cart); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := h.service.UpdateCart(c.Context(), userID, cart); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Cart updated successfully",
	})
}

func (h *CartHandler) Get(c *fiber.Ctx) error {
	userID := c.Params("userId")

	cart, err := h.service.GetCart(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(cart)
}