// path: controllers/locate.go
package controllers

import (
	"fmt"
	"math"

	"github.com/gofiber/fiber/v2"
)

type LocateReq struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func HandleLocate(c *fiber.Ctx) error {
	var req LocateReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	label := fmt.Sprintf("Lat %.5f, Lng %.5f (~%dm)", req.Lat, req.Lon, int(math.Round(distanceRough(req.Lat, req.Lon))))
	return c.JSON(fiber.Map{"label": label})
}

// NOT real distance; placeholder for human-readable label
func distanceRough(lat, lon float64) float64 { return math.Sqrt(lat*lat+lon*lon) * 1000 }
