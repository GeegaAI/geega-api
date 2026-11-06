package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 删除当月月度清零留痕后，独立标记仍能阻止当月重复清零
func TestDeleteQuotaImportRecordKeepsMonthlyResetGuard(t *testing.T) {
	db := useQuotaResetDB(t)
	month := time.Now().Format("2006-01")
	record, err := CreateAutoResetRecord(3, month, time.Now().Unix())
	require.NoError(t, err)

	require.NoError(t, DeleteQuotaImportRecord(record.Id))

	var count int64
	require.NoError(t, db.Model(&QuotaImportRecord{}).Where("id = ?", record.Id).Count(&count).Error)
	assert.Equal(t, int64(0), count, "record is deleted")

	done, err := HasResetRecordForMonth(month)
	require.NoError(t, err)
	assert.True(t, done, "monthly guard survives history deletion")
}

// 已执行的导入任务也可以删除，且只删记录明细、不影响用户余额
func TestDeleteQuotaImportRecordRemovesExecutedTaskAndDetails(t *testing.T) {
	db := useQuotaResetDB(t)
	user := createQuotaResetUser(t, db, "del-user", common.RoleCommonUser, 100, operation_setting.QuotaResetModeMonthly, 0)

	now := time.Now().Unix()
	record := &QuotaImportRecord{
		Type:        QuotaImportTypeUpload,
		Filename:    "del.xlsx",
		Status:      QuotaImportStatusPending,
		TotalRows:   1,
		CreatedTime: now,
	}
	details := []QuotaImportDetail{
		{Username: "del-user", AmountYuan: 100, Quota: 50000000, CreatedTime: now},
	}
	require.NoError(t, CreateQuotaImportRecord(record, details))
	_, err := ExecuteQuotaImportRecord(record.Id)
	require.NoError(t, err)

	require.NoError(t, DeleteQuotaImportRecord(record.Id))

	var recordCount, detailCount int64
	require.NoError(t, db.Model(&QuotaImportRecord{}).Where("id = ?", record.Id).Count(&recordCount).Error)
	require.NoError(t, db.Model(&QuotaImportDetail{}).Where("record_id = ?", record.Id).Count(&detailCount).Error)
	assert.Equal(t, int64(0), recordCount, "executed task record is deleted")
	assert.Equal(t, int64(0), detailCount, "details are deleted together")
	assert.Equal(t, 50000000, userQuotaById(t, db, user.Id).Quota, "deleting history never touches balances")
}
