package middleware

import (
	"log/slog"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 输出结构化请求日志（slog JSON），user_id 在 Auth 之后才有值。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path + redactQuery(c.Request.URL.RawQuery)

		c.Next()

		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
		}
		if userID, ok := UserID(c); ok {
			attrs = append(attrs, "user_id", userID)
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		switch {
		case c.Writer.Status() >= 500:
			slog.Error("request", attrs...)
		case c.Writer.Status() >= 400:
			slog.Warn("request", attrs...)
		default:
			slog.Info("request", attrs...)
		}
	}
}

// sensitiveQueryKeys 是必须掩码后再入日志的查询参数。
// ticket 是音频播放票据：它是「能听这条录音」的凭据，落进日志等于鉴权形同虚设，
// 且日志的可见范围通常远大于 API Key。
var sensitiveQueryKeys = []string{"ticket", "token", "api_key", "apikey", "access_token"}

// redactQuery 对敏感查询参数做掩码，返回可直接拼接的 "?k=v&..." 形式。
func redactQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		// 解析失败时宁可不记录查询串，也不要冒险原样落盘。
		return "?<unparsable>"
	}
	for _, key := range sensitiveQueryKeys {
		if values.Has(key) {
			values.Set(key, "***")
		}
	}
	return "?" + values.Encode()
}
