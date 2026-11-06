package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertAvailabilityCompensationLog(t *testing.T, channelId int, modelName string, createdAt int64, logType int, statusCode int) {
	t.Helper()
	other := ""
	if logType == LogTypeError {
		other = fmt.Sprintf(`{"status_code":%d}`, statusCode)
	}
	require.NoError(t, LOG_DB.Create(&Log{
		Type:      logType,
		CreatedAt: createdAt,
		ChannelId: channelId,
		ModelName: modelName,
		Other:     other,
	}).Error)
}

func findAvailabilityCompensationRow(t *testing.T, rows []AvailabilityCompensationRow, channelId int, modelName string) AvailabilityCompensationRow {
	t.Helper()
	for _, row := range rows {
		if row.ChannelId == channelId && row.ModelName == modelName {
			return row
		}
	}
	t.Fatalf("row %d/%s not found", channelId, modelName)
	return AvailabilityCompensationRow{}
}

func TestGetAvailabilityCompensation_TwoPerspectivesAndSupplierAggregation(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 101, Name: "Acme_modelA", Status: common.ChannelStatusEnabled}).Error)
	require.NoError(t, DB.Create(&Channel{Id: 102, Name: "Acme_modelB", Status: common.ChannelStatusEnabled}).Error)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	end := start + 10*60

	// 第 0 分钟：5 成功 + 5 错误（3 个服务端 500、2 个用户侧 400）
	for i := 0; i < 5; i++ {
		insertAvailabilityCompensationLog(t, 101, "model-a", start, LogTypeConsume, 0)
	}
	for i := 0; i < 3; i++ {
		insertAvailabilityCompensationLog(t, 101, "model-a", start, LogTypeError, 500)
	}
	for i := 0; i < 2; i++ {
		insertAvailabilityCompensationLog(t, 101, "model-a", start, LogTypeError, 400)
	}
	// 第 1 分钟：10 成功
	for i := 0; i < 10; i++ {
		insertAvailabilityCompensationLog(t, 101, "model-a", start+60, LogTypeConsume, 0)
	}
	// 第 2 分钟：1 成功 + 1 个服务端 500（两套口径均为错误分钟）
	insertAvailabilityCompensationLog(t, 101, "model-a", start+120, LogTypeConsume, 0)
	insertAvailabilityCompensationLog(t, 101, "model-a", start+120, LogTypeError, 500)
	// 另一渠道同一分钟：验证供应商级按“模型×渠道分钟行”汇总
	insertAvailabilityCompensationLog(t, 102, "model-b", start, LogTypeConsume, 0)
	insertAvailabilityCompensationLog(t, 102, "model-b", start, LogTypeError, 500)

	result, err := GetAvailabilityCompensation(start, end)
	require.NoError(t, err)

	assert.Equal(t, int64(10), result.TotalMinutes)
	assert.False(t, result.Snapshot)
	assert.Equal(t, 2, result.ChannelCount)
	assert.Equal(t, 2, result.ModelCount)
	assert.Equal(t, 1, result.SupplierCount)
	assert.Equal(t, start, result.DataStartTimestamp)
	assert.Equal(t, start+120+59, result.DataEndTimestamp)

	row := findAvailabilityCompensationRow(t, result.Rows, 101, "model-a")
	assert.Equal(t, "Acme", row.Supplier)
	assert.Equal(t, int64(16), row.OkRequests)
	assert.Equal(t, int64(6), row.FailAllRequests)
	assert.Equal(t, int64(4), row.FailServiceRequests)
	assert.Equal(t, 27.273, row.ErrorRateAll)
	assert.Equal(t, 20.0, row.ErrorRateService)
	assert.Equal(t, int64(2), row.ErrorMinutesGe50All)
	assert.Equal(t, int64(2), row.ErrorMinutesAnyAll)
	// 版本2 分母为成功 + 服务端错误：3/(5+3)=37.5%，不足 50%
	assert.Equal(t, int64(1), row.ErrorMinutesGe50Service)
	assert.Equal(t, int64(2), row.ErrorMinutesAnyService)
	assert.Equal(t, 80.0, row.AvailabilityGe50All)
	assert.Equal(t, 90.0, row.AvailabilityGe50Service)
	assert.Equal(t, AvailabilityCompensationTierBelow95, row.TierAll)
	assert.Equal(t, 10.0, row.CompensationPctAll)

	require.Len(t, result.Suppliers, 1)
	supplier := result.Suppliers[0]
	assert.Equal(t, "Acme", supplier.Supplier)
	assert.Equal(t, 2, supplier.ChannelCount)
	assert.Equal(t, int64(17), supplier.OkRequests)
	assert.Equal(t, int64(7), supplier.FailAllRequests)
	assert.Equal(t, int64(5), supplier.FailServiceRequests)
	// 同一分钟在两个渠道各记一次错误分钟，与导出脚本的汇总方式一致
	assert.Equal(t, int64(3), supplier.ErrorMinutesGe50All)
	assert.Equal(t, int64(2), supplier.ErrorMinutesGe50Service)
	assert.Equal(t, 70.0, supplier.AvailabilityGe50All)
	assert.Equal(t, 80.0, supplier.AvailabilityGe50Service)
}

func TestAvailabilityCompensationTier(t *testing.T) {
	cases := []struct {
		availability float64
		tier         string
		pct          float64
	}{
		{100, AvailabilityCompensationTierNone, 0},
		{99.91, AvailabilityCompensationTierNone, 0},
		{99.9, AvailabilityCompensationTier99, 5},
		{99.899, AvailabilityCompensationTier99, 5},
		{99, AvailabilityCompensationTier99, 5},
		{98.999, AvailabilityCompensationTier95, 8},
		{95, AvailabilityCompensationTier95, 8},
		{94.999, AvailabilityCompensationTierBelow95, 10},
	}
	for _, item := range cases {
		tier, pct := AvailabilityCompensationTier(item.availability)
		assert.Equal(t, item.tier, tier, "availability=%v", item.availability)
		assert.Equal(t, item.pct, pct, "availability=%v", item.availability)
	}
}

func TestIsAvailabilityCompensationServiceStatus(t *testing.T) {
	for _, status := range []int{0, 404, 408, 429, 500, 502, 529, 599} {
		assert.True(t, IsAvailabilityCompensationServiceStatus(status), "status=%d", status)
	}
	for _, status := range []int{400, 401, 403, 413, 418, 422} {
		assert.False(t, IsAvailabilityCompensationServiceStatus(status), "status=%d", status)
	}
}
