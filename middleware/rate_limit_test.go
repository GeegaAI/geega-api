package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func useRateLimitMiniRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()

	previousRedisEnabled := common.RedisEnabled
	previousRedisClient := common.RDB
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	require.NoError(t, redisClient.Ping(context.Background()).Err())

	common.RedisEnabled = true
	common.RDB = redisClient
	t.Cleanup(func() {
		_ = redisClient.Close()
		common.RedisEnabled = previousRedisEnabled
		common.RDB = previousRedisClient
	})

	return redisServer, redisClient
}

func performRateLimitRequest(router http.Handler, path string, remoteAddr string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.RemoteAddr = remoteAddr
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestRedisIPRateLimiterThresholdTTLAndNamespace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer, _ := useRateLimitMiniRedis(t)

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.GET("/limited", rateLimitFactory(2, 37, "TEST"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	remoteAddr := "192.0.2.10:12345"
	legacyKey := "rateLimit:TEST192.0.2.10"
	_, err := redisServer.Push(legacyKey, "legacy-list-entry")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/limited", remoteAddr).Code)
	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/limited", remoteAddr).Code)
	limitedResponse := performRateLimitRequest(router, "/limited", remoteAddr)
	assert.Equal(t, http.StatusTooManyRequests, limitedResponse.Code)
	assert.Equal(t, "37", limitedResponse.Header().Get("Retry-After"))

	key := redisIPRateLimitKey("TEST", "192.0.2.10")
	count, err := redisServer.Get(key)
	require.NoError(t, err)
	assert.Equal(t, "3", count)
	assert.Equal(t, 37*time.Second, redisServer.TTL(key))
	assert.True(t, redisServer.Exists(legacyKey), "the v2 counter must not touch an old list key")
}

func TestRedisUserRateLimiterUsesSharedFixedWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer, _ := useRateLimitMiniRedis(t)

	router := gin.New()
	router.GET(
		"/limited",
		func(c *gin.Context) { c.Set("id", 42) },
		userRateLimitFactory(1, 23, "USER"),
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
	)

	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/limited", "192.0.2.20:12345").Code)
	assert.Equal(t, http.StatusTooManyRequests, performRateLimitRequest(router, "/limited", "198.51.100.20:12345").Code)

	key := redisUserRateLimitKey("USER", 42)
	assert.True(t, redisServer.Exists(key))
	assert.Equal(t, 23*time.Second, redisServer.TTL(key))
}

func TestRedisEmailVerificationRateLimiterPreservesResponseAndTTL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer, _ := useRateLimitMiniRedis(t)

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.GET("/verify", EmailVerificationRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	remoteAddr := "192.0.2.30:12345"
	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/verify", remoteAddr).Code)
	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/verify", remoteAddr).Code)
	response := performRateLimitRequest(router, "/verify", remoteAddr)
	assert.Equal(t, http.StatusTooManyRequests, response.Code)
	assert.JSONEq(t, `{"success":false,"message":"发送过于频繁，请等待 30 秒后再试"}`, response.Body.String())

	key := redisIPRateLimitKey(EmailVerificationRateLimitMark, "192.0.2.30")
	assert.True(t, redisServer.Exists(key))
	assert.Equal(t, time.Duration(EmailVerificationDuration)*time.Second, redisServer.TTL(key))
}

func TestRedisFixedWindowIsAtomicUnderConcurrency(t *testing.T) {
	redisServer, _ := useRateLimitMiniRedis(t)
	const (
		requestCount = 20
		maximumCount = 7
		duration     = int64(41)
	)
	key := redisIPRateLimitKey("CONCURRENT", "192.0.2.40")

	var allowedCount atomic.Int64
	errorsFound := make(chan error, requestCount)
	var waitGroup sync.WaitGroup
	waitGroup.Add(requestCount)
	for range requestCount {
		go func() {
			defer waitGroup.Done()
			allowed, _, _, err := redisFixedWindowTake(context.Background(), key, maximumCount, duration)
			if err != nil {
				errorsFound <- err
				return
			}
			if allowed {
				allowedCount.Add(1)
			}
		}()
	}
	waitGroup.Wait()
	close(errorsFound)
	for err := range errorsFound {
		require.NoError(t, err)
	}

	assert.Equal(t, int64(maximumCount), allowedCount.Load())
	count, err := redisServer.Get(key)
	require.NoError(t, err)
	assert.Equal(t, "20", count)
	assert.Equal(t, time.Duration(duration)*time.Second, redisServer.TTL(key))
}

func TestRedisFixedWindowResetsAtBoundary(t *testing.T) {
	redisServer, _ := useRateLimitMiniRedis(t)
	const duration = int64(10)
	key := redisIPRateLimitKey("BOUNDARY", "192.0.2.50")

	for range 2 {
		allowed, _, _, err := redisFixedWindowTake(context.Background(), key, 2, duration)
		require.NoError(t, err)
		assert.True(t, allowed)
	}
	allowed, _, _, err := redisFixedWindowTake(context.Background(), key, 2, duration)
	require.NoError(t, err)
	assert.False(t, allowed)

	// This reset is intentional fixed-window behavior. A client can consume one
	// full allowance immediately before and another immediately after a boundary.
	redisServer.FastForward(time.Duration(duration) * time.Second)
	for range 2 {
		allowed, _, _, err = redisFixedWindowTake(context.Background(), key, 2, duration)
		require.NoError(t, err)
		assert.True(t, allowed)
	}
}

func TestRedisFixedWindowRepairsCounterWithoutTTL(t *testing.T) {
	redisServer, _ := useRateLimitMiniRedis(t)
	const duration = int64(29)
	key := redisIPRateLimitKey("MISSING-TTL", "192.0.2.51")
	redisServer.Set(key, "5")

	allowed, count, ttl, err := redisFixedWindowTake(context.Background(), key, 3, duration)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, int64(6), count)
	assert.Equal(t, duration, ttl)
	assert.Equal(t, time.Duration(duration)*time.Second, redisServer.TTL(key))

	redisServer.FastForward(time.Duration(duration) * time.Second)
	assert.False(t, redisServer.Exists(key), "a recovered counter must not remain permanently rate-limited")
}

func TestAuthSessionRateLimitUsesDedicatedWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer, _ := useRateLimitMiniRedis(t)

	previousEnable := common.AuthSessionRateLimitEnable
	previousNum := common.AuthSessionRateLimitNum
	previousDuration := common.AuthSessionRateLimitDuration
	common.AuthSessionRateLimitEnable = true
	common.AuthSessionRateLimitNum = 2
	common.AuthSessionRateLimitDuration = 53
	t.Cleanup(func() {
		common.AuthSessionRateLimitEnable = previousEnable
		common.AuthSessionRateLimitNum = previousNum
		common.AuthSessionRateLimitDuration = previousDuration
	})

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.GET("/session", AuthSessionRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	remoteAddr := "192.0.2.70:12345"
	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/session", remoteAddr).Code)
	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/session", remoteAddr).Code)
	limitedResponse := performRateLimitRequest(router, "/session", remoteAddr)
	assert.Equal(t, http.StatusTooManyRequests, limitedResponse.Code)
	assert.Equal(t, "53", limitedResponse.Header().Get("Retry-After"))

	key := redisIPRateLimitKey("AS", "192.0.2.70")
	assert.True(t, redisServer.Exists(key))
	assert.Equal(t, 53*time.Second, redisServer.TTL(key))
}

func TestAuthSessionRateLimitDoesNotConsumeCriticalBudget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, _ = useRateLimitMiniRedis(t)

	previousAuthSessionEnable := common.AuthSessionRateLimitEnable
	previousCriticalEnable := common.CriticalRateLimitEnable
	previousCriticalNum := common.CriticalRateLimitNum
	common.AuthSessionRateLimitEnable = true
	common.CriticalRateLimitEnable = true
	common.CriticalRateLimitNum = 3
	t.Cleanup(func() {
		common.AuthSessionRateLimitEnable = previousAuthSessionEnable
		common.CriticalRateLimitEnable = previousCriticalEnable
		common.CriticalRateLimitNum = previousCriticalNum
	})

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.GET("/session", AuthSessionRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.GET("/critical", CriticalRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	remoteAddr := "192.0.2.71:12345"
	for range common.CriticalRateLimitNum {
		assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/critical", remoteAddr).Code)
	}
	assert.Equal(t, http.StatusTooManyRequests, performRateLimitRequest(router, "/critical", remoteAddr).Code)

	// Session endpoints must keep working after the critical budget is spent.
	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/session", remoteAddr).Code)
}

func TestGlobalWebRateLimitSkipsStaticResources(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer, _ := useRateLimitMiniRedis(t)

	previousEnable := common.GlobalWebRateLimitEnable
	previousNum := common.GlobalWebRateLimitNum
	previousDuration := common.GlobalWebRateLimitDuration
	common.GlobalWebRateLimitEnable = true
	common.GlobalWebRateLimitNum = 2
	common.GlobalWebRateLimitDuration = 71
	t.Cleanup(func() {
		common.GlobalWebRateLimitEnable = previousEnable
		common.GlobalWebRateLimitNum = previousNum
		common.GlobalWebRateLimitDuration = previousDuration
	})

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.Use(GlobalWebRateLimit())
	router.NoRoute(func(c *gin.Context) {
		c.String(http.StatusOK, "response")
	})

	remoteAddr := "192.0.2.80:12345"
	staticPaths := []string{
		"/assets/static/js/index-abc123.js",
		"/assets/static/css/index-def456.css",
		"/assets/logo-dark.svg",
		"/favicon.ico",
		"/logo.png",
		"/pay-apple.png",
	}
	for _, path := range staticPaths {
		for range 5 {
			response := performRateLimitRequest(router, path, remoteAddr)
			assert.Equal(t, http.StatusOK, response.Code, "static path %s must bypass the web limiter", path)
		}
	}

	// Documents (SPA routes) still count against the budget.
	assert.Equal(t, http.StatusOK, performRateLimitRequest(router, "/", remoteAddr).Code)
	assert.Equal(t, http.StatusOK, performRateLimitRequest(router, "/sign-in", remoteAddr).Code)
	limited := performRateLimitRequest(router, "/dashboard", remoteAddr)
	assert.Equal(t, http.StatusTooManyRequests, limited.Code)
	assert.Equal(t, "71", limited.Header().Get("Retry-After"))

	// Static resources must remain reachable after the document budget is spent.
	assert.Equal(t, http.StatusOK, performRateLimitRequest(router, "/assets/static/js/index-abc123.js", remoteAddr).Code)

	key := redisIPRateLimitKey("GW", "192.0.2.80")
	count, err := redisServer.Get(key)
	require.NoError(t, err)
	assert.Equal(t, "3", count, "only document navigations may consume the web budget")
}

func TestIsWebStaticResourceClassification(t *testing.T) {
	staticPaths := []string{
		"/assets/app.js",
		"/assets/a/b/c.chunk.css",
		"/favicon.ico",
		"/logo.png",
		"/some/thing.min.mjs",
		"/fonts/inter.woff2",
		"/hero.webp",
		"/sitemap.txt",
	}
	for _, path := range staticPaths {
		assert.True(t, isWebStaticResource(path), path)
	}

	documentPaths := []string{
		"/",
		"/sign-in",
		"/dashboard",
		"/console/channel",
		"/oauth/github",
		"/api/status",
		"/v1/chat/completions",
	}
	for _, path := range documentPaths {
		assert.False(t, isWebStaticResource(path), path)
	}
}

func TestRedisFailurePolicies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, redisClient := useRateLimitMiniRedis(t)
	require.NoError(t, redisClient.Close())

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.GET("/ip", rateLimitFactory(1, 30, "FAIL-IP"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.GET(
		"/user",
		func(c *gin.Context) { c.Set("id", 7) },
		userRateLimitFactory(1, 30, "FAIL-USER"),
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
	)
	router.GET("/email", EmailVerificationRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	ipResponse := performRateLimitRequest(router, "/ip", "192.0.2.60:12345")
	assert.Equal(t, http.StatusInternalServerError, ipResponse.Code)
	assert.Empty(t, ipResponse.Body.String())
	userResponse := performRateLimitRequest(router, "/user", "192.0.2.61:12345")
	assert.Equal(t, http.StatusInternalServerError, userResponse.Code)
	assert.Empty(t, userResponse.Body.String())
	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/email", "192.0.2.62:12345").Code)
}
