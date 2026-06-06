package middleware

import (
	"rize-api/pkg/jwt"
	"rize-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

const UserIDKey = "userID"

func Auth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Cookies("rize_token")
		if token == "" {
			return response.Unauthorized(c)
		}
		claims, err := jwt.Verify(token, jwtSecret)
		if err != nil {
			return response.Unauthorized(c)
		}
		c.Locals(UserIDKey, claims.UserID)
		return c.Next()
	}
}

func GetUserID(c *fiber.Ctx) string {
	id, _ := c.Locals(UserIDKey).(string)
	return id
}
