package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

// Params holds the parsed pagination values for a single request.
type Params struct {
	Page   int
	Limit  int
	Offset int
}

// Parse extracts page and limit from the request query string.
// Applies defaults and caps limit at maxLimit (100).
// Never returns an error — invalid values fall back to defaults silently.
func Parse(c *gin.Context) Params {
	page := parsePositiveInt(c.Query("page"), defaultPage)
	limit := min(parsePositiveInt(c.Query("limit"), defaultLimit), maxLimit)

	return Params{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}

func parsePositiveInt(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
