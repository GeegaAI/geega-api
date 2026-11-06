package operation_setting

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func preserveQuotaResetSchedule(t *testing.T) {
	t.Helper()
	general := GetGeneralSetting()
	previousDay, previousHour := general.QuotaResetDay, general.QuotaResetHour
	t.Cleanup(func() {
		general.QuotaResetDay, general.QuotaResetHour = previousDay, previousHour
	})
}

func TestGetQuotaResetDayHourFallsBackOnInvalidConfig(t *testing.T) {
	preserveQuotaResetSchedule(t)
	general := GetGeneralSetting()

	general.QuotaResetDay, general.QuotaResetHour = 0, -1
	day, hour := GetQuotaResetDayHour()
	assert.Equal(t, 1, day)
	assert.Equal(t, 0, hour)

	general.QuotaResetDay, general.QuotaResetHour = 29, 24
	day, hour = GetQuotaResetDayHour()
	assert.Equal(t, 1, day)
	assert.Equal(t, 0, hour)

	general.QuotaResetDay, general.QuotaResetHour = 5, 6
	day, hour = GetQuotaResetDayHour()
	assert.Equal(t, 5, day)
	assert.Equal(t, 6, hour)
}

func TestMonthlyResetWindow(t *testing.T) {
	preserveQuotaResetSchedule(t)
	general := GetGeneralSetting()
	general.QuotaResetDay, general.QuotaResetHour = 5, 6
	loc := time.Local

	// 本月重置时刻之前：本周期从上月 5 日 6 点起算
	effective, next := MonthlyResetWindow(time.Date(2026, 10, 3, 12, 0, 0, 0, loc))
	assert.Equal(t, time.Date(2026, 9, 5, 6, 0, 0, 0, loc), effective)
	assert.Equal(t, time.Date(2026, 10, 5, 6, 0, 0, 0, loc), next)

	// 重置时刻之后：下一次跨月
	effective, next = MonthlyResetWindow(time.Date(2026, 10, 28, 12, 0, 0, 0, loc))
	assert.Equal(t, time.Date(2026, 10, 5, 6, 0, 0, 0, loc), effective)
	assert.Equal(t, time.Date(2026, 11, 5, 6, 0, 0, 0, loc), next)

	// 跨年滚动
	effective, next = MonthlyResetWindow(time.Date(2026, 12, 20, 0, 0, 0, 0, loc))
	assert.Equal(t, time.Date(2026, 12, 5, 6, 0, 0, 0, loc), effective)
	assert.Equal(t, time.Date(2027, 1, 5, 6, 0, 0, 0, loc), next)

	// 恰好落在重置时刻：本周期从此刻起算
	effective, next = MonthlyResetWindow(time.Date(2026, 10, 5, 6, 0, 0, 0, loc))
	assert.Equal(t, time.Date(2026, 10, 5, 6, 0, 0, 0, loc), effective)
	assert.Equal(t, time.Date(2026, 11, 5, 6, 0, 0, 0, loc), next)
}
