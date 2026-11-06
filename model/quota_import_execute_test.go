package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 多次执行只覆盖额度：文件值直接写入余额，不做叠加也不做扣减
func TestExecuteQuotaImportRecordOverwritesOnRepeatedRuns(t *testing.T) {
	db := useQuotaResetDB(t)
	user := createQuotaResetUser(t, db, "repeat", common.RoleCommonUser, 999, operation_setting.QuotaResetModeMonthly, 0)

	run := func(quota int) *QuotaImportRecord {
		now := time.Now().Unix()
		record := &QuotaImportRecord{
			Type:        QuotaImportTypeUpload,
			Filename:    "repeat.xlsx",
			Status:      QuotaImportStatusPending,
			TotalRows:   1,
			CreatedTime: now,
		}
		details := []QuotaImportDetail{
			{Username: "repeat", AmountYuan: float64(quota) / 500000, Quota: quota, CreatedTime: now},
		}
		require.NoError(t, CreateQuotaImportRecord(record, details))
		got, err := ExecuteQuotaImportRecord(record.Id)
		require.NoError(t, err)
		return got
	}

	first := run(50000000)
	assert.Equal(t, 1, first.SuccessRows)
	assert.Equal(t, 50000000, userQuotaById(t, db, user.Id).Quota)

	second := run(125000000)
	assert.Equal(t, 1, second.SuccessRows)
	assert.Equal(t, 125000000, userQuotaById(t, db, user.Id).Quota,
		"second run overwrites the balance instead of adding 50M to it")

	third := run(10000000)
	assert.Equal(t, 1, third.SuccessRows)
	assert.Equal(t, 10000000, userQuotaById(t, db, user.Id).Quota,
		"a smaller file value overwrites as-is, not treated as a deduction")
}

// 月度额度设置只做额度赋值：不给任何人清零；名单外用户额度保持原样。
// 判定只看"用户是否存在 + 生效策略是否 monthly"，角色不参与
func TestExecuteQuotaImportRecordAssignsMonthlyUsersOnly(t *testing.T) {
	db := useQuotaResetDB(t)
	monthly := createQuotaResetUser(t, db, "m-user", common.RoleCommonUser, 999, operation_setting.QuotaResetModeMonthly, 0)
	adminMonthly := createQuotaResetUser(t, db, "a-monthly", common.RoleAdminUser, 999, operation_setting.QuotaResetModeMonthly, 0)
	outsider := createQuotaResetUser(t, db, "outsider", common.RoleCommonUser, 888, operation_setting.QuotaResetModeMonthly, 0)
	adminNever := createQuotaResetUser(t, db, "a-never", common.RoleAdminUser, 999, operation_setting.QuotaResetModeNever, 0)
	cycle := createQuotaResetUser(t, db, "c-user", common.RoleCommonUser, 999, operation_setting.QuotaResetModeCycle, 12345)
	never := createQuotaResetUser(t, db, "n-user", common.RoleCommonUser, 999, operation_setting.QuotaResetModeNever, 0)

	now := time.Now().Unix()
	record := &QuotaImportRecord{
		Type:        QuotaImportTypeUpload,
		Filename:    "verify.xlsx",
		Status:      QuotaImportStatusPending,
		TotalRows:   5,
		CreatedTime: now,
	}
	details := []QuotaImportDetail{
		{Username: "m-user", AmountYuan: 100, Quota: 50000000, CreatedTime: now},
		{Username: "a-monthly", AmountYuan: 150, Quota: 75000000, CreatedTime: now},
		{Username: "a-never", AmountYuan: 180, Quota: 90000000, CreatedTime: now},
		{Username: "c-user", AmountYuan: 200, Quota: 100000000, CreatedTime: now},
		{Username: "n-user", AmountYuan: 300, Quota: 150000000, CreatedTime: now},
	}
	require.NoError(t, CreateQuotaImportRecord(record, details))

	got, err := ExecuteQuotaImportRecord(record.Id)
	require.NoError(t, err)

	assert.Equal(t, 2, got.SuccessRows, "monthly users including admins get the file amount")
	assert.Equal(t, 3, got.SkippedRows)
	assert.Equal(t, 0, got.UnmatchedRows)
	assert.Equal(t, 0, got.FailedRows)
	assert.Equal(t, 0, got.ResetCount, "import never clears quotas")
	assert.Equal(t, QuotaImportStatusPartial, got.Status)

	assert.Equal(t, 50000000, userQuotaById(t, db, monthly.Id).Quota, "monthly user gets the file amount")
	assert.Equal(t, 75000000, userQuotaById(t, db, adminMonthly.Id).Quota,
		"admin on the monthly policy is assigned like any other user")
	assert.Equal(t, 888, userQuotaById(t, db, outsider.Id).Quota,
		"monthly users not in the list keep their balance; import does not clear")
	assert.Equal(t, 999, userQuotaById(t, db, adminNever.Id).Quota, "admin on no-reset policy is skipped")
	assert.Equal(t, 999, userQuotaById(t, db, cycle.Id).Quota, "cycle user is skipped entirely")
	assert.Equal(t, 999, userQuotaById(t, db, never.Id).Quota, "no-reset user is skipped entirely")
	assert.Equal(t, int64(12345), userQuotaById(t, db, cycle.Id).QuotaNextResetAt,
		"cycle anchor is untouched by Excel execution")
}
