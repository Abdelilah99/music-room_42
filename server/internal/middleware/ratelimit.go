package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	limiter "github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// NewRateLimiter returns a Gin middleware that enforces the given rate.
// rateStr uses ulule/limiter format: "<count>-<period>" where period is
// S (second), M (minute), H (hour), or D (day). Example: "100-M".
// Panics on invalid format so misconfiguration is caught at startup.
func NewRateLimiter(rateStr string) gin.HandlerFunc {
	rate, err := limiter.NewRateFromFormatted(rateStr)
	if err != nil {
		panic(fmt.Sprintf("middleware: invalid rate limit %q: %v", rateStr, err))
	}
	store := memory.NewStore()
	instance := limiter.New(store, rate)
	return mgin.NewMiddleware(instance)
}
