// path: controllers/helpers.go
package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// --- keep your existing content, plus randString, parseBool, getenv ---

// why: single source for human-friendly area labels
func areaLabel(lat, lon float64, acc *int) string {
	accStr := ""
	if acc != nil && *acc > 0 {
		accStr = fmt.Sprintf(" · GPS ±%dm", *acc)
	}
	return fmt.Sprintf("Near %.3f, %.3f%s", round3(lat), round3(lon), accStr)
}

func round3(f float64) float64 { return float64(int(f*1000)) / 1000.0 }

type ErrorResp struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func badReq(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(ErrorResp{OK: false, Error: msg})
}
func serverErr(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResp{OK: false, Error: err.Error()})
}

// parseBool understands common truthy strings.
func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

// getenv returns env var value or default.
func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// randString returns a short, safe random string (hex) of length n.
func randString(n int) string {
	if n <= 0 {
		n = 6
	}
	// hex doubles length; allocate enough bytes to cover n hex chars
	b := make([]byte, (n+1)/2)
	_, _ = rand.Read(b) // crypto-safe; errors are extremely rare
	s := hex.EncodeToString(b)
	if len(s) > n {
		return s[:n]
	}
	return s
}
