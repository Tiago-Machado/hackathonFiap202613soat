package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	limiter   *redis_rate.Limiter
	perMinute int
}

func NewRateLimiter(rdb *redis.Client, perMinute int) *RateLimiter {
	return &RateLimiter{limiter: redis_rate.NewLimiter(rdb), perMinute: perMinute}
}

func (rl *RateLimiter) PerUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "ip:" + c.ClientIP()
		if userID, ok := c.Get("userID"); ok {
			key = "user:" + userID.(string)
		}

		res, err := rl.limiter.Allow(c.Request.Context(), key, redis_rate.PerMinute(rl.perMinute))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable,
				gin.H{"error": "limitador indisponível, tente novamente"})
			return
		}

		c.Header("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		if res.Allowed == 0 {
			c.Header("Retry-After", strconv.Itoa(int(res.RetryAfter.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"error": "muitas requisições, aguarde um momento"})
			return
		}

		c.Next()
	}
}
