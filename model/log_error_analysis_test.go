package model

import (
	"testing"
	"time"
)

func TestNormalizeErrorReason(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		errorType   string
		errorCode   string
		requestPath string
		content     string
		want        string
	}{
		{name: "gateway timeout", statusCode: 504, want: ErrorReasonTimeout},
		{name: "upstream timeout text", statusCode: 400, content: "upstream request timeout", want: ErrorReasonTimeout},
		{name: "no available channel", statusCode: 418, content: "当前分组下无可用渠道", want: ErrorReasonNoAvailableChannel},
		{name: "empty teapot", statusCode: 418, want: ErrorReasonNoAvailableChannel},
		{name: "responses endpoint unsupported", statusCode: 404, content: "/v1/responses is not supported by upstream", want: ErrorReasonEndpointUnsupported},
		{name: "responses endpoint path only", statusCode: 404, requestPath: "/v1/responses", want: ErrorReasonEndpointUnsupported},
		{name: "model not found on responses path", statusCode: 404, requestPath: "/v1/responses", content: "model_not_found", want: ErrorReasonModelUnavailable},
		{name: "model not found", statusCode: 404, requestPath: "/v1/chat/completions", content: "model_not_found", want: ErrorReasonModelUnavailable},
		{name: "model offline", statusCode: 400, content: "模型已下线", want: ErrorReasonModelUnavailable},
		{name: "quota exhausted", statusCode: 429, content: "insufficient_quota: balance is not enough", want: ErrorReasonQuotaExhausted},
		{name: "rate limited", statusCode: 429, content: "rate limit reached", want: ErrorReasonRateLimited},
		{name: "auth failed", statusCode: 401, want: ErrorReasonAuthFailed},
		{name: "reasoning effort unsupported", statusCode: 400, content: "reasoning_effort only supports low or high", want: ErrorReasonReasoningEffort},
		{name: "content filtered", statusCode: 400, content: "rejected by content policy", want: ErrorReasonContentFiltered},
		{name: "bad request", statusCode: 400, want: ErrorReasonBadRequest},
		{name: "upstream unavailable", statusCode: 503, want: ErrorReasonUpstreamUnavailable},
		{name: "upstream server error", statusCode: 500, want: ErrorReasonUpstreamError},
		{name: "other", statusCode: 0, content: "unexpected failure", want: ErrorReasonOther},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NormalizeErrorReason(test.statusCode, test.errorType, test.errorCode, test.requestPath, test.content)
			if got != test.want {
				t.Fatalf("NormalizeErrorReason() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestErrorAnalysisBucketStartUsesClientTimezone(t *testing.T) {
	loc := time.FixedZone("client", 8*60*60)
	timestamp := time.Date(2026, 10, 23, 0, 30, 0, 0, loc).Unix()

	hourBucket := errorAnalysisBucketStart(timestamp, "hour", loc)
	dayBucket := errorAnalysisBucketStart(timestamp, "day", loc)

	if hourBucket != time.Date(2026, 10, 23, 0, 0, 0, 0, loc).Unix() {
		t.Fatalf("hour bucket = %d, want 00:00", hourBucket)
	}
	if dayBucket != time.Date(2026, 10, 23, 0, 0, 0, 0, loc).Unix() {
		t.Fatalf("day bucket = %d, want 00:00", dayBucket)
	}
	if nextErrorAnalysisBucket(dayBucket, "day", loc) != time.Date(2026, 10, 24, 0, 0, 0, 0, loc).Unix() {
		t.Fatalf("next day bucket is incorrect")
	}
}
