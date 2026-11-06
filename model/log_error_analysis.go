package model

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	errorAnalysisMaxLogRows       = 50000
	errorAnalysisMaxBreakdownRows = 500
	errorAnalysisSampleRunes      = 240
)

const (
	ErrorReasonTimeout             = "timeout"
	ErrorReasonUpstreamUnavailable = "upstream_unavailable"
	ErrorReasonRateLimited         = "rate_limited"
	ErrorReasonQuotaExhausted      = "quota_exhausted"
	ErrorReasonNoAvailableChannel  = "no_available_channel"
	ErrorReasonModelUnavailable    = "model_unavailable"
	ErrorReasonAuthFailed          = "auth_failed"
	ErrorReasonReasoningEffort     = "reasoning_effort_unsupported"
	ErrorReasonEndpointUnsupported = "endpoint_unsupported"
	ErrorReasonBadRequest          = "bad_request"
	ErrorReasonContentFiltered     = "content_filtered"
	ErrorReasonUpstreamError       = "upstream_error"
	ErrorReasonOther               = "other"
)

type ErrorAnalysisSummary struct {
	ErrorCount       int64   `json:"error_count"`
	RequestCount     int64   `json:"request_count"`
	ErrorRate        float64 `json:"error_rate"`
	AffectedUsers    int     `json:"affected_users"`
	AffectedChannels int     `json:"affected_channels"`
	AffectedModels   int     `json:"affected_models"`
}

type ErrorTrendPoint struct {
	Timestamp  int64 `json:"timestamp"`
	ErrorCount int64 `json:"error_count"`
}

type ErrorReasonStat struct {
	Code          string  `json:"code"`
	Count         int64   `json:"count"`
	Ratio         float64 `json:"ratio"`
	StatusCodes   []int   `json:"status_codes"`
	ChannelCount  int     `json:"channel_count"`
	SampleMessage string  `json:"sample_message"`
}

type ErrorBreakdownRow struct {
	ChannelId                int     `json:"channel_id"`
	ChannelName              string  `json:"channel_name"`
	ModelName                string  `json:"model_name"`
	ReasonCode               string  `json:"reason_code"`
	ErrorCount               int64   `json:"error_count"`
	ReasonRatio              float64 `json:"reason_ratio"`
	ChannelModelRequestCount int64   `json:"channel_model_request_count"`
	ChannelModelErrorCount   int64   `json:"channel_model_error_count"`
	ChannelModelErrorRate    float64 `json:"channel_model_error_rate"`
	StatusCodes              []int   `json:"status_codes"`
	UserCount                int     `json:"user_count"`
	SampleMessage            string  `json:"sample_message"`
}

type ErrorChannelStat struct {
	ChannelId      int     `json:"channel_id"`
	ChannelName    string  `json:"channel_name"`
	ChannelType    int     `json:"channel_type"`
	ErrorCount     int64   `json:"error_count"`
	RequestCount   int64   `json:"request_count"`
	ErrorRate      float64 `json:"error_rate"`
	TopReasonCode  string  `json:"top_reason_code"`
	TopReasonCount int64   `json:"top_reason_count"`
	ModelCount     int     `json:"model_count"`
	UserCount      int     `json:"user_count"`
	LastErrorAt    int64   `json:"last_error_at"`
}

type ErrorModelStat struct {
	ModelName      string  `json:"model_name"`
	ErrorCount     int64   `json:"error_count"`
	RequestCount   int64   `json:"request_count"`
	ErrorRate      float64 `json:"error_rate"`
	TopReasonCode  string  `json:"top_reason_code"`
	TopReasonCount int64   `json:"top_reason_count"`
	ChannelCount   int     `json:"channel_count"`
	UserCount      int     `json:"user_count"`
	LastErrorAt    int64   `json:"last_error_at"`
}

type ErrorAnalysisResult struct {
	StartTimestamp int64                `json:"start_timestamp"`
	EndTimestamp   int64                `json:"end_timestamp"`
	Bucket         string               `json:"bucket"`
	Truncated      bool                 `json:"truncated"`
	RowsTruncated  bool                 `json:"rows_truncated"`
	Summary        ErrorAnalysisSummary `json:"summary"`
	Trend          []ErrorTrendPoint    `json:"trend"`
	Reasons        []ErrorReasonStat    `json:"reasons"`
	Channels       []ErrorChannelStat   `json:"channels"`
	Models         []ErrorModelStat     `json:"models"`
	Rows           []ErrorBreakdownRow  `json:"rows"`
}

type errorAnalysisLogRow struct {
	Id        int    `gorm:"column:id"`
	CreatedAt int64  `gorm:"column:created_at"`
	UserId    int    `gorm:"column:user_id"`
	ModelName string `gorm:"column:model_name"`
	ChannelId int    `gorm:"column:channel_id"`
	Content   string `gorm:"column:content"`
	Other     string `gorm:"column:other"`
}

type errorAnalysisCountRow struct {
	ChannelId int    `gorm:"column:channel_id"`
	ModelName string `gorm:"column:model_name"`
	Type      int    `gorm:"column:type"`
	Total     int64  `gorm:"column:total"`
}

type errorAnalysisParsedRow struct {
	CreatedAt  int64
	UserId     int
	ChannelId  int
	ModelName  string
	StatusCode int
	ReasonCode string
	Sample     string
}

type errorAnalysisChannelModelKey struct {
	ChannelId int
	ModelName string
}

type errorAnalysisChannelMeta struct {
	Name string
	Type int
}

type errorAnalysisReasonAgg struct {
	Count       int64
	StatusCodes map[int]struct{}
	Channels    map[int]struct{}
	Users       map[int]struct{}
	Sample      string
}

type errorAnalysisBreakdownKey struct {
	ChannelId  int
	ModelName  string
	ReasonCode string
}

type errorAnalysisBreakdownAgg struct {
	Count       int64
	StatusCodes map[int]struct{}
	Users       map[int]struct{}
	Sample      string
}

type errorAnalysisChannelAgg struct {
	ErrorCount   int64
	Users        map[int]struct{}
	Models       map[string]struct{}
	LastErrorAt  int64
	ReasonCounts map[string]int64
}

type errorAnalysisModelAgg struct {
	ErrorCount   int64
	Users        map[int]struct{}
	Channels     map[int]struct{}
	LastErrorAt  int64
	ReasonCounts map[string]int64
}

// GetErrorAnalysis 只读聚合 type=5 错误日志。timezoneOffset 为客户端 JS
// getTimezoneOffset() 的值，用于按用户本地时区切分趋势桶。
func GetErrorAnalysis(startTimestamp, endTimestamp int64, channelId int, modelName, group string, timezoneOffset *int) (*ErrorAnalysisResult, error) {
	loc := time.Local
	if timezoneOffset != nil {
		loc = time.FixedZone("client", -*timezoneOffset*60)
	}

	errorQuery := func() *gorm.DB {
		tx := LOG_DB.Table("logs").
			Where("type = ?", LogTypeError).
			Where("created_at >= ? AND created_at <= ?", startTimestamp, endTimestamp)
		return applyErrorAnalysisFilters(tx, channelId, modelName, group)
	}

	var errorTotal int64
	if err := errorQuery().Count(&errorTotal).Error; err != nil {
		return nil, err
	}

	logRows := make([]errorAnalysisLogRow, 0, 256)
	if errorTotal > 0 {
		order := "created_at desc, id desc"
		if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
			order = clickHouseLogOrder("")
		}
		err := errorQuery().
			Select("id, created_at, user_id, model_name, channel_id, content, other").
			Order(order).
			Limit(errorAnalysisMaxLogRows).
			Find(&logRows).Error
		if err != nil {
			return nil, err
		}
	}

	countQuery := LOG_DB.Table("logs").
		Select("channel_id, model_name, type, count(*) as total").
		Where("type IN ?", []int{LogTypeConsume, LogTypeError}).
		Where("created_at >= ? AND created_at <= ?", startTimestamp, endTimestamp)
	countQuery = applyErrorAnalysisFilters(countQuery, channelId, modelName, group)
	var countRows []errorAnalysisCountRow
	if err := countQuery.Group("channel_id, model_name, type").Scan(&countRows).Error; err != nil {
		return nil, err
	}

	var totalRequests int64
	channelModelRequests := make(map[errorAnalysisChannelModelKey]int64, len(countRows))
	channelModelErrors := make(map[errorAnalysisChannelModelKey]int64, len(countRows))
	channelRequests := make(map[int]int64, len(countRows))
	channelErrors := make(map[int]int64, len(countRows))
	modelRequests := make(map[string]int64, len(countRows))
	modelErrors := make(map[string]int64, len(countRows))
	for _, row := range countRows {
		key := errorAnalysisChannelModelKey{ChannelId: row.ChannelId, ModelName: row.ModelName}
		totalRequests += row.Total
		channelModelRequests[key] += row.Total
		channelRequests[row.ChannelId] += row.Total
		modelRequests[row.ModelName] += row.Total
		if row.Type == LogTypeError {
			channelModelErrors[key] += row.Total
			channelErrors[row.ChannelId] += row.Total
			modelErrors[row.ModelName] += row.Total
		}
	}

	parsedRows := make([]errorAnalysisParsedRow, 0, len(logRows))
	reasonAggs := make(map[string]*errorAnalysisReasonAgg)
	breakdownAggs := make(map[errorAnalysisBreakdownKey]*errorAnalysisBreakdownAgg)
	channelAggs := make(map[int]*errorAnalysisChannelAgg)
	modelAggs := make(map[string]*errorAnalysisModelAgg)
	channelModelParsedErrors := make(map[errorAnalysisChannelModelKey]int64)
	affectedUsers := make(map[int]struct{})
	affectedChannels := make(map[int]struct{})
	affectedModels := make(map[string]struct{})

	for _, row := range logRows {
		other, _ := common.StrToMap(row.Other)
		statusCode := errorAnalysisJSONInt(other, "status_code")
		elementChannelId := row.ChannelId
		if elementChannelId == 0 {
			elementChannelId = errorAnalysisJSONInt(other, "channel_id")
		}
		reasonCode := NormalizeErrorReason(
			statusCode,
			errorAnalysisJSONString(other, "error_type"),
			errorAnalysisJSONString(other, "error_code"),
			errorAnalysisJSONString(other, "request_path"),
			row.Content,
		)
		sample := buildErrorAnalysisSample(row.Content, errorAnalysisJSONString(other, "error_type"), errorAnalysisJSONString(other, "error_code"), statusCode)

		parsedRows = append(parsedRows, errorAnalysisParsedRow{
			CreatedAt:  row.CreatedAt,
			UserId:     row.UserId,
			ChannelId:  elementChannelId,
			ModelName:  row.ModelName,
			StatusCode: statusCode,
			ReasonCode: reasonCode,
			Sample:     sample,
		})

		reasonAgg := reasonAggs[reasonCode]
		if reasonAgg == nil {
			reasonAgg = &errorAnalysisReasonAgg{
				StatusCodes: make(map[int]struct{}),
				Channels:    make(map[int]struct{}),
				Users:       make(map[int]struct{}),
			}
			reasonAggs[reasonCode] = reasonAgg
		}
		reasonAgg.Count++
		addErrorAnalysisInt(reasonAgg.StatusCodes, statusCode)
		reasonAgg.Channels[elementChannelId] = struct{}{}
		addErrorAnalysisUser(reasonAgg.Users, row.UserId)
		if reasonAgg.Sample == "" {
			reasonAgg.Sample = sample
		}

		breakdownKey := errorAnalysisBreakdownKey{ChannelId: elementChannelId, ModelName: row.ModelName, ReasonCode: reasonCode}
		breakdownAgg := breakdownAggs[breakdownKey]
		if breakdownAgg == nil {
			breakdownAgg = &errorAnalysisBreakdownAgg{
				StatusCodes: make(map[int]struct{}),
				Users:       make(map[int]struct{}),
			}
			breakdownAggs[breakdownKey] = breakdownAgg
		}
		breakdownAgg.Count++
		addErrorAnalysisInt(breakdownAgg.StatusCodes, statusCode)
		addErrorAnalysisUser(breakdownAgg.Users, row.UserId)
		if breakdownAgg.Sample == "" {
			breakdownAgg.Sample = sample
		}

		channelAgg := channelAggs[elementChannelId]
		if channelAgg == nil {
			channelAgg = &errorAnalysisChannelAgg{
				Users:        make(map[int]struct{}),
				Models:       make(map[string]struct{}),
				ReasonCounts: make(map[string]int64),
			}
			channelAggs[elementChannelId] = channelAgg
		}
		channelAgg.ErrorCount++
		addErrorAnalysisUser(channelAgg.Users, row.UserId)
		channelAgg.Models[row.ModelName] = struct{}{}
		channelAgg.ReasonCounts[reasonCode]++
		if row.CreatedAt > channelAgg.LastErrorAt {
			channelAgg.LastErrorAt = row.CreatedAt
		}

		modelAgg := modelAggs[row.ModelName]
		if modelAgg == nil {
			modelAgg = &errorAnalysisModelAgg{
				Users:        make(map[int]struct{}),
				Channels:     make(map[int]struct{}),
				ReasonCounts: make(map[string]int64),
			}
			modelAggs[row.ModelName] = modelAgg
		}
		modelAgg.ErrorCount++
		addErrorAnalysisUser(modelAgg.Users, row.UserId)
		modelAgg.Channels[elementChannelId] = struct{}{}
		modelAgg.ReasonCounts[reasonCode]++
		if row.CreatedAt > modelAgg.LastErrorAt {
			modelAgg.LastErrorAt = row.CreatedAt
		}

		channelModelKey := errorAnalysisChannelModelKey{ChannelId: elementChannelId, ModelName: row.ModelName}
		channelModelParsedErrors[channelModelKey]++
		addErrorAnalysisUser(affectedUsers, row.UserId)
		affectedChannels[elementChannelId] = struct{}{}
		affectedModels[row.ModelName] = struct{}{}
	}

	channelIds := make(map[int]struct{}, len(channelAggs)+len(channelErrors))
	for id := range channelAggs {
		channelIds[id] = struct{}{}
	}
	for id, count := range channelErrors {
		if count > 0 {
			channelIds[id] = struct{}{}
		}
	}
	channelMeta := loadErrorAnalysisChannelMeta(channelIds)

	summary := ErrorAnalysisSummary{
		ErrorCount:       errorTotal,
		RequestCount:     totalRequests,
		AffectedUsers:    len(affectedUsers),
		AffectedChannels: len(affectedChannels),
		AffectedModels:   len(affectedModels),
	}
	if totalRequests > 0 {
		summary.ErrorRate = float64(errorTotal) / float64(totalRequests)
	}

	reasons := make([]ErrorReasonStat, 0, len(reasonAggs))
	for code, agg := range reasonAggs {
		reason := ErrorReasonStat{
			Code:          code,
			Count:         agg.Count,
			StatusCodes:   sortedErrorAnalysisInts(agg.StatusCodes),
			ChannelCount:  len(agg.Channels),
			SampleMessage: agg.Sample,
		}
		if errorTotal > 0 {
			reason.Ratio = float64(agg.Count) / float64(errorTotal)
		}
		reasons = append(reasons, reason)
	}
	sort.Slice(reasons, func(i, j int) bool {
		if reasons[i].Count != reasons[j].Count {
			return reasons[i].Count > reasons[j].Count
		}
		return reasons[i].Code < reasons[j].Code
	})

	breakdownRows := make([]ErrorBreakdownRow, 0, len(breakdownAggs))
	for key, agg := range breakdownAggs {
		channelModelKey := errorAnalysisChannelModelKey{ChannelId: key.ChannelId, ModelName: key.ModelName}
		errorCount := channelModelErrors[channelModelKey]
		if errorCount == 0 {
			errorCount = channelModelParsedErrors[channelModelKey]
		}
		requestCount := channelModelRequests[channelModelKey]
		if requestCount == 0 {
			requestCount = errorCount
		}
		row := ErrorBreakdownRow{
			ChannelId:                key.ChannelId,
			ChannelName:              channelMeta[key.ChannelId].Name,
			ModelName:                key.ModelName,
			ReasonCode:               key.ReasonCode,
			ErrorCount:               agg.Count,
			ChannelModelRequestCount: requestCount,
			ChannelModelErrorCount:   errorCount,
			StatusCodes:              sortedErrorAnalysisInts(agg.StatusCodes),
			UserCount:                len(agg.Users),
			SampleMessage:            agg.Sample,
		}
		if errorTotal > 0 {
			row.ReasonRatio = float64(agg.Count) / float64(errorTotal)
		}
		if requestCount > 0 {
			row.ChannelModelErrorRate = float64(errorCount) / float64(requestCount)
		}
		breakdownRows = append(breakdownRows, row)
	}
	sort.Slice(breakdownRows, func(i, j int) bool {
		if breakdownRows[i].ErrorCount != breakdownRows[j].ErrorCount {
			return breakdownRows[i].ErrorCount > breakdownRows[j].ErrorCount
		}
		if breakdownRows[i].ChannelId != breakdownRows[j].ChannelId {
			return breakdownRows[i].ChannelId < breakdownRows[j].ChannelId
		}
		if breakdownRows[i].ModelName != breakdownRows[j].ModelName {
			return breakdownRows[i].ModelName < breakdownRows[j].ModelName
		}
		return breakdownRows[i].ReasonCode < breakdownRows[j].ReasonCode
	})
	rowsTruncated := len(breakdownRows) > errorAnalysisMaxBreakdownRows
	if rowsTruncated {
		breakdownRows = breakdownRows[:errorAnalysisMaxBreakdownRows]
	}

	channels := make([]ErrorChannelStat, 0, len(channelIds))
	for channelId := range channelIds {
		agg := channelAggs[channelId]
		errorCount := channelErrors[channelId]
		if errorCount == 0 && agg != nil {
			errorCount = agg.ErrorCount
		}
		if errorCount == 0 {
			continue
		}
		requestCount := channelRequests[channelId]
		if requestCount == 0 {
			requestCount = errorCount
		}
		stat := ErrorChannelStat{
			ChannelId:    channelId,
			ChannelName:  channelMeta[channelId].Name,
			ChannelType:  channelMeta[channelId].Type,
			ErrorCount:   errorCount,
			RequestCount: requestCount,
		}
		if requestCount > 0 {
			stat.ErrorRate = float64(errorCount) / float64(requestCount)
		}
		if agg != nil {
			stat.TopReasonCode, stat.TopReasonCount = topErrorAnalysisReason(agg.ReasonCounts)
			stat.ModelCount = len(agg.Models)
			stat.UserCount = len(agg.Users)
			stat.LastErrorAt = agg.LastErrorAt
		}
		channels = append(channels, stat)
	}
	sort.Slice(channels, func(i, j int) bool {
		if channels[i].ErrorCount != channels[j].ErrorCount {
			return channels[i].ErrorCount > channels[j].ErrorCount
		}
		return channels[i].ChannelId < channels[j].ChannelId
	})

	modelNames := make(map[string]struct{}, len(modelAggs)+len(modelErrors))
	for name := range modelAggs {
		modelNames[name] = struct{}{}
	}
	for name, count := range modelErrors {
		if count > 0 {
			modelNames[name] = struct{}{}
		}
	}
	models := make([]ErrorModelStat, 0, len(modelNames))
	for name := range modelNames {
		agg := modelAggs[name]
		errorCount := modelErrors[name]
		if errorCount == 0 && agg != nil {
			errorCount = agg.ErrorCount
		}
		if errorCount == 0 {
			continue
		}
		requestCount := modelRequests[name]
		if requestCount == 0 {
			requestCount = errorCount
		}
		stat := ErrorModelStat{
			ModelName:    name,
			ErrorCount:   errorCount,
			RequestCount: requestCount,
		}
		if requestCount > 0 {
			stat.ErrorRate = float64(errorCount) / float64(requestCount)
		}
		if agg != nil {
			stat.TopReasonCode, stat.TopReasonCount = topErrorAnalysisReason(agg.ReasonCounts)
			stat.ChannelCount = len(agg.Channels)
			stat.UserCount = len(agg.Users)
			stat.LastErrorAt = agg.LastErrorAt
		}
		models = append(models, stat)
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].ErrorCount != models[j].ErrorCount {
			return models[i].ErrorCount > models[j].ErrorCount
		}
		return models[i].ModelName < models[j].ModelName
	})

	bucket := "day"
	if endTimestamp-startTimestamp <= 48*60*60 {
		bucket = "hour"
	}

	return &ErrorAnalysisResult{
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		Bucket:         bucket,
		Truncated:      errorTotal > int64(len(logRows)),
		RowsTruncated:  rowsTruncated,
		Summary:        summary,
		Trend:          buildErrorAnalysisTrend(parsedRows, startTimestamp, endTimestamp, bucket, loc),
		Reasons:        reasons,
		Channels:       channels,
		Models:         models,
		Rows:           breakdownRows,
	}, nil
}

func applyErrorAnalysisFilters(tx *gorm.DB, channelId int, modelName, group string) *gorm.DB {
	if channelId != 0 {
		tx = tx.Where("channel_id = ?", channelId)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if group != "" {
		tx = tx.Where(logGroupCol+" = ?", group)
	}
	return tx
}

func loadErrorAnalysisChannelMeta(channelIds map[int]struct{}) map[int]errorAnalysisChannelMeta {
	meta := make(map[int]errorAnalysisChannelMeta, len(channelIds))
	if len(channelIds) == 0 || DB == nil {
		return meta
	}
	ids := make([]int, 0, len(channelIds))
	for id := range channelIds {
		if id != 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return meta
	}
	var rows []struct {
		Id   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
		Type int    `gorm:"column:type"`
	}
	if err := DB.Table("channels").Select("id, name, type").Where("id IN ?", ids).Find(&rows).Error; err != nil {
		common.SysError("failed to load channel metadata for error analysis: " + err.Error())
		return meta
	}
	for _, row := range rows {
		meta[row.Id] = errorAnalysisChannelMeta{Name: row.Name, Type: row.Type}
	}
	return meta
}

func buildErrorAnalysisTrend(rows []errorAnalysisParsedRow, startTimestamp, endTimestamp int64, bucket string, loc *time.Location) []ErrorTrendPoint {
	counts := make(map[int64]int64, len(rows))
	for _, row := range rows {
		counts[errorAnalysisBucketStart(row.CreatedAt, bucket, loc)]++
	}
	points := make([]ErrorTrendPoint, 0, 32)
	for timestamp := errorAnalysisBucketStart(startTimestamp, bucket, loc); timestamp <= endTimestamp; timestamp = nextErrorAnalysisBucket(timestamp, bucket, loc) {
		points = append(points, ErrorTrendPoint{
			Timestamp:  timestamp,
			ErrorCount: counts[timestamp],
		})
	}
	return points
}

func errorAnalysisBucketStart(timestamp int64, bucket string, loc *time.Location) int64 {
	t := time.Unix(timestamp, 0).In(loc)
	if bucket == "hour" {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, loc).Unix()
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc).Unix()
}

func nextErrorAnalysisBucket(timestamp int64, bucket string, loc *time.Location) int64 {
	t := time.Unix(timestamp, 0).In(loc)
	if bucket == "hour" {
		return t.Add(time.Hour).Unix()
	}
	return t.AddDate(0, 0, 1).Unix()
}

func topErrorAnalysisReason(counts map[string]int64) (string, int64) {
	bestCode := ""
	var bestCount int64
	for code, count := range counts {
		if count > bestCount || (count == bestCount && (bestCode == "" || code < bestCode)) {
			bestCode = code
			bestCount = count
		}
	}
	return bestCode, bestCount
}

func addErrorAnalysisInt(set map[int]struct{}, value int) {
	if value != 0 {
		set[value] = struct{}{}
	}
}

func addErrorAnalysisUser(set map[int]struct{}, userId int) {
	if userId != 0 {
		set[userId] = struct{}{}
	}
}

func sortedErrorAnalysisInts(set map[int]struct{}) []int {
	values := make([]int, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Ints(values)
	return values
}

func errorAnalysisJSONInt(source map[string]interface{}, key string) int {
	if source == nil {
		return 0
	}
	switch value := source[key].(type) {
	case float64:
		return int(value)
	case float32:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	case json.Number:
		number, _ := value.Int64()
		return int(number)
	case string:
		number, _ := strconv.Atoi(strings.TrimSpace(value))
		return number
	default:
		return 0
	}
}

func errorAnalysisJSONString(source map[string]interface{}, key string) string {
	if source == nil {
		return ""
	}
	value, ok := source[key]
	if !ok || value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func buildErrorAnalysisSample(content, errorType, errorCode string, statusCode int) string {
	sample := strings.Join(strings.Fields(content), " ")
	if sample == "" {
		parts := make([]string, 0, 2)
		if errorType != "" {
			parts = append(parts, errorType)
		}
		if errorCode != "" {
			parts = append(parts, errorCode)
		}
		if len(parts) == 0 && statusCode != 0 {
			return fmt.Sprintf("HTTP %d", statusCode)
		}
		sample = strings.Join(parts, " / ")
	}
	runes := []rune(sample)
	if len(runes) > errorAnalysisSampleRunes {
		return string(runes[:errorAnalysisSampleRunes]) + "…"
	}
	return sample
}

// NormalizeErrorReason 将上游错误归一化为稳定原因码。错误正文可能被脱敏或为空，
// 因此按状态码兜底；具体关键字命中优先于状态码兜底。
func NormalizeErrorReason(statusCode int, errorType, errorCode, requestPath, content string) string {
	text := strings.ToLower(strings.Join([]string{errorType, errorCode, content}, " "))
	has := func(keywords ...string) bool {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				return true
			}
		}
		return false
	}
	responsesEndpoint := strings.Contains(strings.ToLower(requestPath), "/responses") || has("/v1/responses", "responses api", "responses endpoint")

	switch {
	case statusCode == 504 || statusCode == 524 || has("timeout", "timed out", "deadline exceeded", "超时"):
		return ErrorReasonTimeout
	case statusCode == 418 && (has("无可用渠道", "no available channel", "no channel available") || strings.TrimSpace(content) == ""):
		return ErrorReasonNoAvailableChannel
	case has("model_not_found", "model not found", "model_not_exist", "model does not exist", "no such model", "unsupported model", "模型不存在", "模型未找到", "模型下线", "模型已下线"):
		return ErrorReasonModelUnavailable
	case has("insufficient quota", "insufficient_quota", "quota exceeded", "quota_exceeded", "余额不足", "额度不足", "欠费", "arrears"):
		return ErrorReasonQuotaExhausted
	case statusCode == 429:
		return ErrorReasonRateLimited
	case statusCode == 401 || statusCode == 403:
		return ErrorReasonAuthFailed
	case has("reasoning_effort", "reasoning effort"):
		return ErrorReasonReasoningEffort
	case has("content policy", "content_filter", "content filter", "safety system", "敏感", "内容审核"):
		return ErrorReasonContentFiltered
	case responsesEndpoint && (statusCode == 404 || statusCode == 405 || statusCode == 501 || has("not support", "unsupported", "not implemented", "不支持")):
		return ErrorReasonEndpointUnsupported
	case statusCode == 502 || statusCode == 503:
		return ErrorReasonUpstreamUnavailable
	case statusCode == 400 || statusCode == 422:
		return ErrorReasonBadRequest
	case statusCode >= 500:
		return ErrorReasonUpstreamError
	case statusCode == 418:
		return ErrorReasonNoAvailableChannel
	case statusCode == 404:
		return ErrorReasonModelUnavailable
	default:
		return ErrorReasonOther
	}
}
