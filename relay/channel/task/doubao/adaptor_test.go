package doubao

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAdaptor(baseURL string) *TaskAdaptor {
	adaptor := &TaskAdaptor{}
	adaptor.Init(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: baseURL,
			ApiKey:         "sk-test",
		},
	})
	return adaptor
}

func withVideoTierPriceConfig(t *testing.T, jsonStr string) {
	t.Helper()
	snapshot := ratio_setting.VideoTierPrice2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateVideoTierPriceByJSONString("{}"))
		if snapshot != "" && snapshot != "{}" {
			require.NoError(t, ratio_setting.UpdateVideoTierPriceByJSONString(snapshot))
		}
	})
	require.NoError(t, ratio_setting.UpdateVideoTierPriceByJSONString(jsonStr))
}

func TestBuildRequestURLShouldUseV2VideoGenerationsPath(t *testing.T) {
	adaptor := newTestAdaptor("https://maas.51ima.com")

	uri, err := adaptor.BuildRequestURL(&relaycommon.RelayInfo{})
	require.NoError(t, err)
	assert.Equal(t, "https://maas.51ima.com/v2/video/generations", uri)
}

func TestFetchTaskShouldHitV2PathWithBearerToken(t *testing.T) {
	var gotPath string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"t1","status":"queued"}`))
	}))
	defer server.Close()

	adaptor := newTestAdaptor(server.URL)
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{"task_id": "task-123"}, "")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "/v2/video/generations/task-123", gotPath)
	assert.Equal(t, "Bearer sk-test", gotAuth)
}

func TestParseTaskResultShouldMapUpstreamStatuses(t *testing.T) {
	adaptor := &TaskAdaptor{}

	cases := []struct {
		name       string
		body       string
		wantStatus model.TaskStatus
		wantURL    string
		wantTokens int
	}{
		{
			name:       "queued",
			body:       `{"id":"t1","status":"queued"}`,
			wantStatus: model.TaskStatusQueued,
		},
		{
			name:       "running",
			body:       `{"id":"t1","status":"running"}`,
			wantStatus: model.TaskStatusInProgress,
		},
		{
			name:       "succeeded",
			body:       `{"id":"t1","status":"succeeded","content":{"video_url":"https://cdn.example/v.mp4"},"usage":{"completion_tokens":216900,"total_tokens":216900}}`,
			wantStatus: model.TaskStatusSuccess,
			wantURL:    "https://cdn.example/v.mp4",
			wantTokens: 216900,
		},
		{
			name:       "failed",
			body:       `{"id":"t1","status":"failed","error":{"code":"task_failed","message":"上游超时"}}`,
			wantStatus: model.TaskStatusFailure,
		},
		{
			name:       "cancelled",
			body:       `{"id":"t1","status":"cancelled"}`,
			wantStatus: model.TaskStatusFailure,
		},
		{
			name:       "unknown falls back to in progress",
			body:       `{"id":"t1","status":"mystery"}`,
			wantStatus: model.TaskStatusInProgress,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := adaptor.ParseTaskResult([]byte(tc.body))
			require.NoError(t, err)
			assert.Equal(t, string(tc.wantStatus), result.Status)
			if tc.wantURL != "" {
				assert.Equal(t, tc.wantURL, result.Url)
			}
			if tc.wantTokens > 0 {
				assert.Equal(t, tc.wantTokens, result.TotalTokens)
			}
		})
	}
}

func TestEstimateTierRatioShouldUseAdminConfig(t *testing.T) {
	withVideoTierPriceConfig(t, `{"doubao-seedance-2-0-260128":{"720p":100,"1080p":200}}`)

	ratio, ok := ratio_setting.GetTaskTierRatio("doubao-seedance-2-0-260128", "1080p", false)
	assert.True(t, ok)
	assert.InDelta(t, 2.0, ratio, 1e-9)
}

func TestEstimateTierRatioShouldReturnFalseWithoutConfig(t *testing.T) {
	withVideoTierPriceConfig(t, `{}`)

	_, ok := ratio_setting.GetTaskTierRatio("doubao-seedance-2-0-260128", "1080p", false)
	assert.False(t, ok)

	_, ok = ratio_setting.GetTaskTierRatio("unknown-model", "1080p", false)
	assert.False(t, ok)
}

func TestParseTaskResultShouldKeepPollingWhenStatusUnknownAndNoError(t *testing.T) {
	adaptor := &TaskAdaptor{}

	info, err := adaptor.ParseTaskResult([]byte(`{"id":"t1","status":"creating"}`))

	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusInProgress, info.Status)
}

func TestResolveTaskResolutionShouldPreferMetadataThenTopLevelThenDefault(t *testing.T) {
	req := relaycommon.TaskSubmitReq{Metadata: map[string]interface{}{"resolution": "1080p"}, Resolution: "720p"}
	assert.Equal(t, "1080p", resolveTaskResolution(req))

	req = relaycommon.TaskSubmitReq{Resolution: "4k"}
	assert.Equal(t, "4k", resolveTaskResolution(req))

	assert.Equal(t, defaultVideoResolution, resolveTaskResolution(relaycommon.TaskSubmitReq{}))
}

func TestEstimateVideoTokensShouldFollowOfficialFormula(t *testing.T) {
	assert.Equal(t, 108000, estimateVideoTokens("720p", 5))
	assert.Equal(t, 0, estimateVideoTokens("unknown", 5))
	assert.Equal(t, 0, estimateVideoTokens("720p", 0))
}

func TestParseTaskResultShouldEstimateTokensWhenUsageMissing(t *testing.T) {
	adaptor := &TaskAdaptor{}

	info, err := adaptor.ParseTaskResult([]byte(`{"id":"t1","status":"succeeded","resolution":"720p","duration":5,"content":{"video_url":"https://example.com/a.mp4"}}`))

	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, info.Status)
	assert.Equal(t, 108000, info.TotalTokens)
	assert.Equal(t, "https://example.com/a.mp4", info.Url)
}

func TestParseTaskResultShouldKeepUpstreamUsage(t *testing.T) {
	adaptor := &TaskAdaptor{}

	info, err := adaptor.ParseTaskResult([]byte(`{"id":"t1","status":"succeeded","resolution":"720p","duration":5,"usage":{"completion_tokens":108900,"total_tokens":108900}}`))

	require.NoError(t, err)
	assert.Equal(t, 108900, info.TotalTokens)
}

func TestPreConsumeMultiplierShouldMatchEstimatedTokenRatio(t *testing.T) {
	// 720p/5s = 108,000 token，基准预扣等价于 QuotaPerUnit/2 = 250,000 token。
	assert.InDelta(t, 108000.0/250000.0, preConsumeMultiplier(estimateVideoTokens("720p", 5)), 1e-9)
	assert.Equal(t, 1.0, preConsumeMultiplier(0))
}

func TestMaxInputVideoSecondsShouldFollowModelCap(t *testing.T) {
	assert.Equal(t, 30, maxInputVideoSeconds("doubao-seedance-2-5-260628"))
	assert.Equal(t, 15, maxInputVideoSeconds("doubao-seedance-2-0-260128"))
	assert.Equal(t, 15, maxInputVideoSeconds("unknown-model"))
}

func TestPreConsumeMultiplierShouldIncludeMaxInputDuration(t *testing.T) {
	// 720p：输出 5s + 输入上限 15s = 20s → 432,000 token。
	tokens := estimateVideoTokens("720p", 5+maxInputVideoSeconds("doubao-seedance-2-0-260128"))

	assert.Equal(t, 432000, tokens)
	assert.InDelta(t, 432000.0/250000.0, preConsumeMultiplier(tokens), 1e-9)
}
