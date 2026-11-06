package model

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// 供应商可用性赔偿测算口径与 datap-migration/对账导出-2026-09/build_xlsx.py 保持一致：
// 按“模型 × 渠道 × 分钟”统计错误率，分钟错误率 ≥50% 记为错误分钟，
// 可用性 = (周期总分钟数 − 错误分钟数) / 周期总分钟数 × 100%，再按合同档位套赔偿比例。
const (
	AvailabilityCompensationTierNone    = "none"
	AvailabilityCompensationTier99      = "tier_99"
	AvailabilityCompensationTier95      = "tier_95"
	AvailabilityCompensationTierBelow95 = "below_95"
)

type AvailabilityCompensationRow struct {
	ModelName               string  `json:"model_name"`
	ChannelId               int     `json:"channel_id"`
	ChannelName             string  `json:"channel_name"`
	Supplier                string  `json:"supplier"`
	OkRequests              int64   `json:"ok_requests"`
	FailAllRequests         int64   `json:"fail_all_requests"`
	FailServiceRequests     int64   `json:"fail_service_requests"`
	ErrorRateAll            float64 `json:"error_rate_all"`
	ErrorRateService        float64 `json:"error_rate_service"`
	ErrorMinutesGe50All     int64   `json:"error_minutes_ge50_all"`
	ErrorMinutesAnyAll      int64   `json:"error_minutes_any_all"`
	ErrorMinutesGe50Service int64   `json:"error_minutes_ge50_service"`
	ErrorMinutesAnyService  int64   `json:"error_minutes_any_service"`
	AvailabilityGe50All     float64 `json:"availability_ge50_all"`
	AvailabilityAnyAll      float64 `json:"availability_any_all"`
	AvailabilityGe50Service float64 `json:"availability_ge50_service"`
	AvailabilityAnyService  float64 `json:"availability_any_service"`
	TierAll                 string  `json:"tier_all"`
	TierService             string  `json:"tier_service"`
	CompensationPctAll      float64 `json:"compensation_pct_all"`
	CompensationPctService  float64 `json:"compensation_pct_service"`
}

type AvailabilityCompensationSupplierRow struct {
	Supplier                string  `json:"supplier"`
	ChannelCount            int     `json:"channel_count"`
	OkRequests              int64   `json:"ok_requests"`
	FailAllRequests         int64   `json:"fail_all_requests"`
	FailServiceRequests     int64   `json:"fail_service_requests"`
	ErrorRateAll            float64 `json:"error_rate_all"`
	ErrorRateService        float64 `json:"error_rate_service"`
	ErrorMinutesGe50All     int64   `json:"error_minutes_ge50_all"`
	ErrorMinutesAnyAll      int64   `json:"error_minutes_any_all"`
	ErrorMinutesGe50Service int64   `json:"error_minutes_ge50_service"`
	ErrorMinutesAnyService  int64   `json:"error_minutes_any_service"`
	AvailabilityGe50All     float64 `json:"availability_ge50_all"`
	AvailabilityAnyAll      float64 `json:"availability_any_all"`
	AvailabilityGe50Service float64 `json:"availability_ge50_service"`
	AvailabilityAnyService  float64 `json:"availability_any_service"`
	TierAll                 string  `json:"tier_all"`
	TierService             string  `json:"tier_service"`
	CompensationPctAll      float64 `json:"compensation_pct_all"`
	CompensationPctService  float64 `json:"compensation_pct_service"`
}

type AvailabilityCompensationChannel struct {
	Id       int    `json:"id" gorm:"column:id"`
	Name     string `json:"name" gorm:"column:name"`
	Status   int    `json:"status" gorm:"column:status"`
	Supplier string `json:"supplier" gorm:"-"`
}

type AvailabilityCompensationResult struct {
	StartTimestamp     int64                                 `json:"start_timestamp"`
	EndTimestamp       int64                                 `json:"end_timestamp"`
	TotalMinutes       int64                                 `json:"total_minutes"`
	Snapshot           bool                                  `json:"snapshot"`
	DataStartTimestamp int64                                 `json:"data_start_timestamp"`
	DataEndTimestamp   int64                                 `json:"data_end_timestamp"`
	ChannelCount       int                                   `json:"channel_count"`
	ModelCount         int                                   `json:"model_count"`
	SupplierCount      int                                   `json:"supplier_count"`
	Rows               []AvailabilityCompensationRow         `json:"rows"`
	Suppliers          []AvailabilityCompensationSupplierRow `json:"suppliers"`
	Channels           []AvailabilityCompensationChannel     `json:"-"`
}

type availabilityCompensationRawCount struct {
	ChannelId  int    `gorm:"column:channel_id"`
	ModelName  string `gorm:"column:model_name"`
	Minute     int64  `gorm:"column:minute"`
	Type       int    `gorm:"column:type"`
	StatusCode int    `gorm:"column:status_code"`
	Total      int64  `gorm:"column:total"`
}

type availabilityCompensationPairKey struct {
	ChannelId int
	ModelName string
}

type availabilityCompensationCounts struct {
	Ok              int64
	FailAll         int64
	FailService     int64
	ErrorMinGe50All int64
	ErrorMinAnyAll  int64
	ErrorMinGe50Svc int64
	ErrorMinAnySvc  int64
}

type availabilityCompensationSupplierAgg struct {
	availabilityCompensationCounts
	Channels map[int]struct{}
}

// GetAvailabilityCompensation 只读聚合日志库，生成供应商可用性赔偿测算。
// endTimestamp 为开区间：与 startTimestamp 的差值必须为整分钟。
func GetAvailabilityCompensation(startTimestamp, endTimestamp int64) (*AvailabilityCompensationResult, error) {
	totalMinutes := (endTimestamp - startTimestamp) / 60
	if totalMinutes < 0 {
		totalMinutes = 0
	}
	channels, err := loadAvailabilityCompensationChannels()
	if err != nil {
		return nil, err
	}
	result := &AvailabilityCompensationResult{
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		TotalMinutes:   totalMinutes,
		Snapshot:       endTimestamp > time.Now().Unix(),
		Rows:           []AvailabilityCompensationRow{},
		Suppliers:      []AvailabilityCompensationSupplierRow{},
		Channels:       channels,
	}

	queryEnd := endTimestamp
	if now := time.Now().Unix(); queryEnd > now {
		queryEnd = now
	}
	if queryEnd <= startTimestamp {
		return result, nil
	}

	rawRows, err := queryAvailabilityCompensationCounts(startTimestamp, queryEnd)
	if err != nil {
		return nil, err
	}
	channelMeta := make(map[int]AvailabilityCompensationChannel, len(channels))
	for _, channel := range channels {
		channelMeta[channel.Id] = channel
	}
	rows, suppliers, dataStart, dataEnd := aggregateAvailabilityCompensation(totalMinutes, rawRows, channelMeta)
	result.Rows = rows
	result.Suppliers = suppliers
	result.DataStartTimestamp = dataStart
	result.DataEndTimestamp = dataEnd

	channelIds := make(map[int]struct{})
	modelNames := make(map[string]struct{})
	for _, row := range rows {
		channelIds[row.ChannelId] = struct{}{}
		modelNames[row.ModelName] = struct{}{}
	}
	result.ChannelCount = len(channelIds)
	result.ModelCount = len(modelNames)
	result.SupplierCount = len(suppliers)
	return result, nil
}

func loadAvailabilityCompensationChannels() ([]AvailabilityCompensationChannel, error) {
	var channels []AvailabilityCompensationChannel
	if err := DB.Table("channels").Select("id, name, status").Order("id").Find(&channels).Error; err != nil {
		return nil, err
	}
	for i := range channels {
		channels[i].Supplier = AvailabilityCompensationSupplier(channels[i].Id, channels[i].Name)
	}
	return channels, nil
}

// AvailabilityCompensationSupplier 沿用对账导出脚本的规则：渠道名 “_” 前缀即供应商。
func AvailabilityCompensationSupplier(channelId int, channelName string) string {
	name := strings.TrimSpace(channelName)
	if name == "" {
		return fmt.Sprintf("渠道%d", channelId)
	}
	if index := strings.Index(name, "_"); index > 0 {
		name = strings.TrimSpace(name[:index])
	}
	if name == "" {
		return fmt.Sprintf("渠道%d", channelId)
	}
	return name
}

func queryAvailabilityCompensationCounts(startTimestamp, endTimestamp int64) ([]availabilityCompensationRawCount, error) {
	minuteExpr := availabilityCompensationMinuteExpr()
	statusExpr := availabilityCompensationStatusCodeExpr()
	selectExpr := "channel_id, model_name, " + minuteExpr + " AS minute, type, " +
		statusExpr + " AS status_code, count(*) AS total"
	tx := LOG_DB.Table("logs").
		Select(selectExpr).
		Where("type IN ?", []int{LogTypeConsume, LogTypeError}).
		Where("created_at >= ? AND created_at < ?", startTimestamp, endTimestamp).
		Where("channel_id > 0").
		Group("channel_id, model_name, " + minuteExpr + ", type, " + statusExpr)
	var rows []availabilityCompensationRawCount
	if err := tx.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func availabilityCompensationMinuteExpr() string {
	switch common.LogDatabaseType() {
	case common.DatabaseTypeClickHouse:
		return "intDiv(created_at, 60)"
	case common.DatabaseTypeMySQL:
		return "created_at DIV 60"
	default:
		return "created_at / 60"
	}
}

func availabilityCompensationStatusCodeExpr() string {
	switch common.LogDatabaseType() {
	case common.DatabaseTypeClickHouse:
		return "JSONExtractInt(other, 'status_code')"
	case common.DatabaseTypePostgreSQL:
		return "COALESCE((NULLIF(other, '')::jsonb ->> 'status_code')::bigint, 0)"
	case common.DatabaseTypeMySQL:
		return "IF(JSON_VALID(other), IFNULL(CAST(JSON_UNQUOTE(JSON_EXTRACT(other, '$.status_code')) AS SIGNED), 0), 0)"
	default:
		return "CASE WHEN json_valid(other) THEN COALESCE(CAST(json_extract(other, '$.status_code') AS INTEGER), 0) ELSE 0 END"
	}
}

// IsAvailabilityCompensationServiceStatus 服务端错误口径：5xx、404、408、429、529
// 及无状态码的连接级错误；400/401/403/413 等用户侧错误不计入版本2。
func IsAvailabilityCompensationServiceStatus(statusCode int) bool {
	return statusCode == 0 || statusCode == 404 || statusCode == 408 ||
		statusCode == 429 || statusCode == 529 || statusCode >= 500
}

func aggregateAvailabilityCompensation(
	totalMinutes int64,
	rawRows []availabilityCompensationRawCount,
	channelMeta map[int]AvailabilityCompensationChannel,
) ([]AvailabilityCompensationRow, []AvailabilityCompensationSupplierRow, int64, int64) {
	type minuteKey struct {
		ChannelId int
		ModelName string
		Minute    int64
	}
	minuteBuckets := make(map[minuteKey]*availabilityCompensationCounts)
	var dataStart, dataEnd int64

	for _, raw := range rawRows {
		key := minuteKey{ChannelId: raw.ChannelId, ModelName: raw.ModelName, Minute: raw.Minute}
		bucket := minuteBuckets[key]
		if bucket == nil {
			bucket = &availabilityCompensationCounts{}
			minuteBuckets[key] = bucket
		}
		if raw.Type == LogTypeConsume {
			bucket.Ok += raw.Total
		} else if raw.Type == LogTypeError {
			bucket.FailAll += raw.Total
			if IsAvailabilityCompensationServiceStatus(raw.StatusCode) {
				bucket.FailService += raw.Total
			}
		}
		if timestamp := raw.Minute * 60; dataStart == 0 || timestamp < dataStart {
			dataStart = timestamp
		}
		if timestamp := raw.Minute*60 + 59; timestamp > dataEnd {
			dataEnd = timestamp
		}
	}

	pairs := make(map[availabilityCompensationPairKey]*availabilityCompensationCounts)
	supplierAggs := make(map[string]*availabilityCompensationSupplierAgg)
	for key, bucket := range minuteBuckets {
		ok := bucket.Ok
		failAll := bucket.FailAll
		failService := bucket.FailService

		pairKey := availabilityCompensationPairKey{ChannelId: key.ChannelId, ModelName: key.ModelName}
		pair := pairs[pairKey]
		if pair == nil {
			pair = &availabilityCompensationCounts{}
			pairs[pairKey] = pair
		}
		accumulateAvailabilityMinute(pair, ok, failAll, failService)

		channel, found := channelMeta[key.ChannelId]
		supplier := ""
		if found {
			supplier = channel.Supplier
		}
		if supplier == "" {
			supplier = AvailabilityCompensationSupplier(key.ChannelId, channel.Name)
		}
		supplierAgg := supplierAggs[supplier]
		if supplierAgg == nil {
			supplierAgg = &availabilityCompensationSupplierAgg{Channels: make(map[int]struct{})}
			supplierAggs[supplier] = supplierAgg
		}
		accumulateAvailabilityMinute(&supplierAgg.availabilityCompensationCounts, ok, failAll, failService)
		supplierAgg.Channels[key.ChannelId] = struct{}{}
	}

	rows := make([]AvailabilityCompensationRow, 0, len(pairs))
	for key, counts := range pairs {
		channel := channelMeta[key.ChannelId]
		supplier := channel.Supplier
		if supplier == "" {
			supplier = AvailabilityCompensationSupplier(key.ChannelId, channel.Name)
		}
		row := AvailabilityCompensationRow{
			ModelName:               key.ModelName,
			ChannelId:               key.ChannelId,
			ChannelName:             availabilityCompensationChannelName(channel, key.ChannelId),
			Supplier:                supplier,
			OkRequests:              counts.Ok,
			FailAllRequests:         counts.FailAll,
			FailServiceRequests:     counts.FailService,
			ErrorMinutesGe50All:     counts.ErrorMinGe50All,
			ErrorMinutesAnyAll:      counts.ErrorMinAnyAll,
			ErrorMinutesGe50Service: counts.ErrorMinGe50Svc,
			ErrorMinutesAnyService:  counts.ErrorMinAnySvc,
			ErrorRateAll:            availabilityCompensationRate(counts.FailAll, counts.Ok+counts.FailAll),
			ErrorRateService:        availabilityCompensationRate(counts.FailService, counts.Ok+counts.FailService),
			AvailabilityGe50All:     availabilityCompensationPercent(totalMinutes, counts.ErrorMinGe50All),
			AvailabilityAnyAll:      availabilityCompensationPercent(totalMinutes, counts.ErrorMinAnyAll),
			AvailabilityGe50Service: availabilityCompensationPercent(totalMinutes, counts.ErrorMinGe50Svc),
			AvailabilityAnyService:  availabilityCompensationPercent(totalMinutes, counts.ErrorMinAnySvc),
		}
		row.TierAll, row.CompensationPctAll = AvailabilityCompensationTier(row.AvailabilityGe50All)
		row.TierService, row.CompensationPctService = AvailabilityCompensationTier(row.AvailabilityGe50Service)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].FailAllRequests != rows[j].FailAllRequests {
			return rows[i].FailAllRequests > rows[j].FailAllRequests
		}
		if rows[i].ChannelId != rows[j].ChannelId {
			return rows[i].ChannelId < rows[j].ChannelId
		}
		return rows[i].ModelName < rows[j].ModelName
	})

	suppliers := make([]AvailabilityCompensationSupplierRow, 0, len(supplierAggs))
	for supplier, agg := range supplierAggs {
		row := AvailabilityCompensationSupplierRow{
			Supplier:                supplier,
			ChannelCount:            len(agg.Channels),
			OkRequests:              agg.Ok,
			FailAllRequests:         agg.FailAll,
			FailServiceRequests:     agg.FailService,
			ErrorMinutesGe50All:     agg.ErrorMinGe50All,
			ErrorMinutesAnyAll:      agg.ErrorMinAnyAll,
			ErrorMinutesGe50Service: agg.ErrorMinGe50Svc,
			ErrorMinutesAnyService:  agg.ErrorMinAnySvc,
			ErrorRateAll:            availabilityCompensationRate(agg.FailAll, agg.Ok+agg.FailAll),
			ErrorRateService:        availabilityCompensationRate(agg.FailService, agg.Ok+agg.FailService),
			AvailabilityGe50All:     availabilityCompensationPercent(totalMinutes, agg.ErrorMinGe50All),
			AvailabilityAnyAll:      availabilityCompensationPercent(totalMinutes, agg.ErrorMinAnyAll),
			AvailabilityGe50Service: availabilityCompensationPercent(totalMinutes, agg.ErrorMinGe50Svc),
			AvailabilityAnyService:  availabilityCompensationPercent(totalMinutes, agg.ErrorMinAnySvc),
		}
		row.TierAll, row.CompensationPctAll = AvailabilityCompensationTier(row.AvailabilityGe50All)
		row.TierService, row.CompensationPctService = AvailabilityCompensationTier(row.AvailabilityGe50Service)
		suppliers = append(suppliers, row)
	}
	sort.Slice(suppliers, func(i, j int) bool {
		if suppliers[i].FailAllRequests != suppliers[j].FailAllRequests {
			return suppliers[i].FailAllRequests > suppliers[j].FailAllRequests
		}
		return suppliers[i].Supplier < suppliers[j].Supplier
	})

	return rows, suppliers, dataStart, dataEnd
}

func accumulateAvailabilityMinute(counts *availabilityCompensationCounts, ok, failAll, failService int64) {
	counts.Ok += ok
	counts.FailAll += failAll
	counts.FailService += failService
	if failAll > 0 {
		counts.ErrorMinAnyAll++
		if failAll*100 >= (ok+failAll)*50 {
			counts.ErrorMinGe50All++
		}
	}
	if failService > 0 {
		counts.ErrorMinAnySvc++
		if failService*100 >= (ok+failService)*50 {
			counts.ErrorMinGe50Svc++
		}
	}
}

func availabilityCompensationChannelName(channel AvailabilityCompensationChannel, channelId int) string {
	if channel.Name != "" {
		return channel.Name
	}
	return fmt.Sprintf("渠道%d", channelId)
}

func availabilityCompensationRate(fail, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(float64(fail)*100000/float64(total)) / 1000
}

func availabilityCompensationPercent(totalMinutes, errorMinutes int64) float64 {
	if totalMinutes <= 0 {
		return 100
	}
	return math.Round(float64(totalMinutes-errorMinutes)*100000/float64(totalMinutes)) / 1000
}

// AvailabilityCompensationTier 返回档位代码与赔偿比例（%）。档位判定使用四舍五入到
// 3 位小数后的可用性；99.9%（含）落在 5% 档，大于 99.9% 无赔偿。
func AvailabilityCompensationTier(availability float64) (string, float64) {
	switch {
	case availability > 99.9:
		return AvailabilityCompensationTierNone, 0
	case availability >= 99:
		return AvailabilityCompensationTier99, 5
	case availability >= 95:
		return AvailabilityCompensationTier95, 8
	default:
		return AvailabilityCompensationTierBelow95, 10
	}
}
