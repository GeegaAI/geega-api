package doubao

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/QuantumNous/new-api/constant"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

// ============================
// Request / Response structures
// ============================

type ContentItem struct {
	Type      string    `json:"type,omitempty"`
	Text      string    `json:"text,omitempty"`
	ImageURL  *MediaURL `json:"image_url,omitempty"`
	VideoURL  *MediaURL `json:"video_url,omitempty"`
	AudioURL  *MediaURL `json:"audio_url,omitempty"`
	DraftTask *struct {
		ID string `json:"id,omitempty"`
	} `json:"draft_task,omitempty"`
	Role string `json:"role,omitempty"`
}

type MediaURL struct {
	URL string `json:"url,omitempty"`
}

type requestPayload struct {
	Model                 string         `json:"model"`
	Content               []ContentItem  `json:"content,omitempty"`
	CallbackURL           string         `json:"callback_url,omitempty"`
	ReturnLastFrame       *dto.BoolValue `json:"return_last_frame,omitempty"`
	ServiceTier           string         `json:"service_tier,omitempty"`
	ExecutionExpiresAfter *dto.IntValue  `json:"execution_expires_after,omitempty"`
	GenerateAudio         *dto.BoolValue `json:"generate_audio,omitempty"`
	Draft                 *dto.BoolValue `json:"draft,omitempty"`
	Tools                 []struct {
		Type string `json:"type,omitempty"`
	} `json:"tools,omitempty"`
	SafetyIdentifier string                 `json:"safety_identifier,omitempty"`
	Priority         *dto.IntValue          `json:"priority,omitempty"`
	Resolution       string                 `json:"resolution,omitempty"`
	Ratio            string                 `json:"ratio,omitempty"`
	Duration         *dto.IntValue          `json:"duration,omitempty"`
	Frames           *dto.IntValue          `json:"frames,omitempty"`
	Seed             *dto.IntValue          `json:"seed,omitempty"`
	CameraFixed      *dto.BoolValue         `json:"camera_fixed,omitempty"`
	Watermark        *dto.BoolValue         `json:"watermark,omitempty"`
	OutputFormat     string                 `json:"output_format,omitempty"`
	Extra            map[string]interface{} `json:"extra,omitempty"`
}

type responsePayload struct {
	ID string `json:"id"` // task_id
}

type responseTask struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Status  string `json:"status"`
	Content struct {
		VideoURL string `json:"video_url"`
	} `json:"content"`
	Seed            int    `json:"seed"`
	Resolution      string `json:"resolution"`
	Duration        int    `json:"duration"`
	Ratio           string `json:"ratio"`
	FramesPerSecond int    `json:"framespersecond"`
	ServiceTier     string `json:"service_tier"`
	Tools           []struct {
		Type string `json:"type"`
	} `json:"tools"`
	Usage struct {
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
		ToolUsage        struct {
			WebSearch int `json:"web_search"`
		} `json:"tool_usage"`
	} `json:"usage"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

// ValidateRequestAndSetAction parses body, validates fields and sets default action.
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *taskdto.TaskError) {
	// Accept only POST /v1/video/generations as "generate" action.
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

// BuildRequestURL constructs the upstream URL.
// 上游网关（如中科金 DeZhu MaaS）使用 /v2/video/generations 统一入口。
func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/v2/video/generations", a.baseURL), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

// EstimateBilling 根据请求 metadata 中的输出分辨率与是否包含视频输入，返回相对基准价的计费 OtherRatio。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	hasVideo := hasVideoInTaskRequest(req)
	resolution := resolveTaskResolution(req)
	ratio, ok := ratio_setting.GetTaskTierRatio(info.OriginModelName, resolution, hasVideo)
	if !ok || ratio == 1.0 {
		return nil
	}
	return map[string]float64{"video_input": ratio}
}

// EstimatePreConsumeMultiplier 按"分辨率 × 时长"估算的 token 量放大预扣额度，避免 4k/长时长任务
// 预扣不足把余额打穿。该倍数只作用于预扣，最终仍按上游实际 token 结算。
func (a *TaskAdaptor) EstimatePreConsumeMultiplier(c *gin.Context, info *relaycommon.RelayInfo) float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return 1
	}
	seconds := taskDurationSeconds(req)
	if seconds <= 0 {
		return 1
	}
	// 含视频输入时，官方 token 用量还包含输入视频时长；预扣阶段拿不到真实时长，
	// 按模型支持的最大输入时长保守预估（宁可多冻结，完成后由结算退回）。
	if hasVideoInTaskRequest(req) {
		seconds += maxInputVideoSeconds(info.OriginModelName)
	}
	return preConsumeMultiplier(estimateVideoTokens(resolveTaskResolution(req), seconds))
}

// maxInputVideoSeconds 返回模型支持的输入视频最大时长（秒）。
// 官方口径：Seedance 2.5 最长 30 秒，2.0 系列最长 15 秒。
func maxInputVideoSeconds(modelName string) int {
	name := strings.ToLower(strings.TrimSpace(modelName))
	if strings.Contains(name, "seedance-2-5") || strings.Contains(name, "seedance-2.5") {
		return 30
	}
	return 15
}

// preConsumeMultiplier 把估算 token 换算成相对基准预扣的倍数。
// 基准预扣额度 = 模型倍率/2 × QuotaPerUnit，等价于 QuotaPerUnit/2 个 token。
func preConsumeMultiplier(tokens int) float64 {
	if tokens <= 0 {
		return 1
	}
	return float64(tokens) * 2 / common.QuotaPerUnit
}

const defaultVideoResolution = "720p"

// videoResolutionPixels 各分辨率档在 16:9 下的像素数（token 只与像素面积相关）。
var videoResolutionPixels = map[string]float64{
	"480p":  854 * 480,
	"720p":  1280 * 720,
	"768p":  1024 * 768,
	"1080p": 1920 * 1080,
	"2k":    2560 * 1440,
	"4k":    3840 * 2160,
}

// resolveTaskResolution 取请求中的分辨率：优先 metadata.resolution，其次顶层 resolution，最后默认 720p。
func resolveTaskResolution(req relaycommon.TaskSubmitReq) string {
	if v, _ := req.Metadata["resolution"].(string); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if strings.TrimSpace(req.Resolution) != "" {
		return strings.TrimSpace(req.Resolution)
	}
	return defaultVideoResolution
}

// hasVideoInTaskRequest 判断请求是否包含视频输入：兼容 metadata.content 与顶层 content。
func hasVideoInTaskRequest(req relaycommon.TaskSubmitReq) bool {
	if hasVideoInMetadata(req.Metadata) {
		return true
	}
	for _, item := range req.Content {
		if item == nil {
			continue
		}
		if itemType, _ := item["type"].(string); itemType == "video_url" {
			return true
		}
		if _, ok := item["video_url"]; ok {
			return true
		}
	}
	return false
}

// taskDurationSeconds 取请求时长：优先 seconds 字符串，其次 duration 整数。
func taskDurationSeconds(req relaycommon.TaskSubmitReq) int {
	if sec, err := strconv.Atoi(strings.TrimSpace(req.Seconds)); err == nil && sec > 0 {
		return sec
	}
	if req.Duration > 0 {
		return req.Duration
	}
	return 0
}

// estimateVideoTokens 按官方口径估算 token 用量：像素 × 24fps × 时长 / 1024。
func estimateVideoTokens(resolution string, duration int) int {
	if duration <= 0 {
		return 0
	}
	pixels, ok := videoResolutionPixels[strings.ToLower(strings.TrimSpace(resolution))]
	if !ok {
		return 0
	}
	return int(math.Round(pixels * 24 * float64(duration) / 1024))
}

// hasVideoInMetadata 直接检查 metadata 的 content 数组是否包含 video_url 条目，
// 避免构建完整的上游 requestPayload。
func hasVideoInMetadata(metadata map[string]interface{}) bool {
	if metadata == nil {
		return false
	}
	contentRaw, ok := metadata["content"]
	if !ok {
		return false
	}
	contentSlice, ok := contentRaw.([]interface{})
	if !ok {
		return false
	}
	for _, item := range contentSlice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if itemMap["type"] == "video_url" {
			return true
		}
		if _, has := itemMap["video_url"]; has {
			return true
		}
	}
	return false
}

// BuildRequestBody converts request into Doubao specific format.
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	body, err := a.convertToRequestPayload(&req)
	if err != nil {
		return nil, errors.Wrap(err, "convert request payload failed")
	}
	if info.IsModelMapped {
		body.Model = info.UpstreamModelName
	} else {
		info.UpstreamModelName = body.Model
	}
	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// DoRequest delegates to common helper.
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response, returns taskID etc.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *taskdto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	// Parse Doubao response
	var dResp responsePayload
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	if dResp.ID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName

	c.JSON(http.StatusOK, ov)
	return dResp.ID, responseBody, nil
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/v2/video/generations/%s", baseUrl, taskID)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq) (*requestPayload, error) {
	r := requestPayload{
		Model:   req.Model,
		Content: []ContentItem{},
	}

	// Add images if present
	if req.HasImage() {
		for _, imgURL := range req.Images {
			r.Content = append(r.Content, ContentItem{
				Type: "image_url",
				ImageURL: &MediaURL{
					URL: imgURL,
				},
			})
		}
	}

	metadata := req.Metadata
	if err := taskcommon.UnmarshalMetadata(metadata, &r); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}

	// 兼容中科金/火山风格的顶层 content：素材直接放顶层时合并进 content 数组。
	if len(req.Content) > 0 {
		contentBytes, err := common.Marshal(req.Content)
		if err != nil {
			return nil, errors.Wrap(err, "marshal top-level content failed")
		}
		var items []ContentItem
		if err := common.Unmarshal(contentBytes, &items); err != nil {
			return nil, errors.Wrap(err, "unmarshal top-level content failed")
		}
		r.Content = append(items, r.Content...)
		if strings.TrimSpace(req.Prompt) == "" {
			var texts []string
			for _, item := range items {
				if item.Type == "text" && strings.TrimSpace(item.Text) != "" {
					texts = append(texts, item.Text)
				}
			}
			if len(texts) > 0 {
				req.Prompt = strings.Join(texts, "\n")
			}
		}
	}

	if r.Resolution == "" {
		r.Resolution = strings.TrimSpace(req.Resolution)
	}
	if r.Resolution == "" {
		r.Resolution = defaultVideoResolution
	}
	if r.Ratio == "" {
		r.Ratio = strings.TrimSpace(req.Ratio)
	}

	if sec := taskDurationSeconds(*req); sec > 0 {
		r.Duration = lo.ToPtr(dto.IntValue(sec))
	}

	r.Content = lo.Reject(r.Content, func(c ContentItem, _ int) bool { return c.Type == "text" })
	r.Content = append(r.Content, ContentItem{
		Type: "text",
		Text: req.Prompt,
	})

	return &r, nil
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	resTask := responseTask{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{
		Code: 0,
	}

	// Map Doubao status to internal status
	switch resTask.Status {
	case "pending", "queued":
		taskResult.Status = model.TaskStatusQueued
		taskResult.Progress = "10%"
	case "processing", "running":
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "50%"
	case "succeeded":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		taskResult.Url = resTask.Content.VideoURL
		// 解析 usage 信息用于按倍率计费
		taskResult.CompletionTokens = resTask.Usage.CompletionTokens
		taskResult.TotalTokens = resTask.Usage.TotalTokens
		// 上游未返回 usage 时按官方口径估算，避免结算回退成按倍率/2 的预扣额度。
		if taskResult.TotalTokens <= 0 {
			if estimated := estimateVideoTokens(resTask.Resolution, resTask.Duration); estimated > 0 {
				taskResult.CompletionTokens = estimated
				taskResult.TotalTokens = estimated
			}
		}
	case "failed":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = resTask.Error.Message
	case "cancelled", "canceled":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = "task cancelled"
	default:
		// Unknown status, treat as processing
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "30%"
	}

	return &taskResult, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	var dResp responseTask
	if err := common.Unmarshal(originTask.Data, &dResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal doubao task data failed")
	}

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = originTask.TaskID
	openAIVideo.TaskID = originTask.TaskID
	openAIVideo.Status = originTask.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(originTask.Progress)
	openAIVideo.SetMetadata("url", dResp.Content.VideoURL)
	openAIVideo.CreatedAt = originTask.CreatedAt
	openAIVideo.CompletedAt = originTask.UpdatedAt
	openAIVideo.Model = originTask.Properties.OriginModelName

	if dResp.Status == "failed" {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: dResp.Error.Message,
			Code:    dResp.Error.Code,
		}
	}

	return common.Marshal(openAIVideo)
}
