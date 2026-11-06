package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func useQuotaResetDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &Option{}, &QuotaImportRecord{}, &QuotaImportDetail{}))
	DB = db
	t.Cleanup(func() {
		DB = previousDB
	})
	return db
}

func createQuotaResetUser(t *testing.T, db *gorm.DB, username string, role int, quota int, mode string, nextReset int64) User {
	t.Helper()
	user := User{
		Username:         username,
		Password:         "test-password",
		Role:             role,
		Status:           common.UserStatusEnabled,
		Quota:            quota,
		QuotaResetMode:   mode,
		QuotaNextResetAt: nextReset,
		AffCode:          fmt.Sprintf("aff-%s", username), // AffCode 有唯一索引
	}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func userQuotaById(t *testing.T, db *gorm.DB, id int) User {
	t.Helper()
	var user User
	require.NoError(t, db.First(&user, id).Error)
	return user
}

func TestResetMonthlyQuotaUsersRespectsEffectiveMode(t *testing.T) {
	db := useQuotaResetDB(t)
	follower := createQuotaResetUser(t, db, "follower", common.RoleCommonUser, 100, "", 0)
	monthly := createQuotaResetUser(t, db, "monthly", common.RoleCommonUser, 200, operation_setting.QuotaResetModeMonthly, 0)
	cycle := createQuotaResetUser(t, db, "cycle", common.RoleCommonUser, 300, operation_setting.QuotaResetModeCycle, time.Now().Add(time.Hour).Unix())
	never := createQuotaResetUser(t, db, "never", common.RoleCommonUser, 400, operation_setting.QuotaResetModeNever, 0)
	admin := createQuotaResetUser(t, db, "admin", common.RoleAdminUser, 500, "", 0)

	count, err := ResetMonthlyQuotaUsers()
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	assert.Equal(t, 0, userQuotaById(t, db, follower.Id).Quota, "'' follows default monthly and is cleared")
	assert.Equal(t, 0, userQuotaById(t, db, monthly.Id).Quota, "explicit monthly is cleared")
	assert.Equal(t, 300, userQuotaById(t, db, cycle.Id).Quota, "cycle user must not be cleared by monthly reset")
	assert.Equal(t, 400, userQuotaById(t, db, never.Id).Quota, "never user must not be cleared")
	assert.Equal(t, 0, userQuotaById(t, db, admin.Id).Quota, "admin follows the same reset policy as everyone else")
}

func TestResetCycleQuotaUsersClearsDueAndConsumesAnchor(t *testing.T) {
	db := useQuotaResetDB(t)
	now := time.Now().Unix()
	due := createQuotaResetUser(t, db, "due", common.RoleCommonUser, 100, operation_setting.QuotaResetModeCycle, now-10)
	future := createQuotaResetUser(t, db, "future", common.RoleCommonUser, 200, operation_setting.QuotaResetModeCycle, now+3600)
	exhausted := createQuotaResetUser(t, db, "exhausted", common.RoleCommonUser, 0, operation_setting.QuotaResetModeCycle, now-10)
	monthly := createQuotaResetUser(t, db, "monthly", common.RoleCommonUser, 300, operation_setting.QuotaResetModeMonthly, 0)
	adminDue := createQuotaResetUser(t, db, "admin-due", common.RoleAdminUser, 400, operation_setting.QuotaResetModeCycle, now-10)

	count, err := ResetCycleQuotaUsers()
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	after := userQuotaById(t, db, due.Id)
	assert.Equal(t, 0, after.Quota)
	assert.Equal(t, int64(0), after.QuotaNextResetAt,
		"anchor is consumed at cycle end so a renewal grant is not cleared by the next tick")
	assert.Equal(t, 200, userQuotaById(t, db, future.Id).Quota, "not-yet-due cycle user untouched")
	assert.Equal(t, 0, userQuotaById(t, db, exhausted.Id).Quota)
	assert.Equal(t, now-10, userQuotaById(t, db, exhausted.Id).QuotaNextResetAt, "quota=0 user is not scanned")
	assert.Equal(t, 300, userQuotaById(t, db, monthly.Id).Quota, "monthly user untouched by cycle reset")
	assert.Equal(t, 0, userQuotaById(t, db, adminDue.Id).Quota, "admin follows the same cycle reset as everyone else")
}

func TestFillQuotaScheduleForMonthlyAndCycle(t *testing.T) {
	db := useQuotaResetDB(t)
	now := time.Now()
	cycleSeconds := operation_setting.GetQuotaResetCycleSeconds()
	anchor := now.Unix() + 3600

	monthly := createQuotaResetUser(t, db, "monthly", common.RoleCommonUser, 100, operation_setting.QuotaResetModeMonthly, 0)
	cycle := createQuotaResetUser(t, db, "cycle", common.RoleCommonUser, 100, operation_setting.QuotaResetModeCycle, anchor)
	unstarted := createQuotaResetUser(t, db, "unstarted", common.RoleCommonUser, 100, operation_setting.QuotaResetModeCycle, 0)
	expired := createQuotaResetUser(t, db, "expired", common.RoleCommonUser, 100, operation_setting.QuotaResetModeCycle, now.Unix()-10)
	never := createQuotaResetUser(t, db, "never", common.RoleCommonUser, 100, operation_setting.QuotaResetModeNever, 0)

	users := []*User{&monthly, &cycle, &unstarted, &expired, &never}
	fillQuotaSchedule(users)

	expectedEffective, expectedReset := operation_setting.MonthlyResetWindow(now)
	assert.Equal(t, expectedEffective.Unix(), monthly.QuotaEffectiveAt, "monthly shows period start")
	assert.Equal(t, expectedReset.Unix(), monthly.QuotaResetAt, "monthly shows next reset time")
	assert.LessOrEqual(t, monthly.QuotaEffectiveAt, now.Unix())
	assert.Greater(t, monthly.QuotaResetAt, now.Unix())

	assert.Equal(t, anchor, cycle.QuotaResetAt)
	assert.Equal(t, anchor-cycleSeconds, cycle.QuotaEffectiveAt)

	assert.Equal(t, int64(0), unstarted.QuotaEffectiveAt, "cycle without anchor has no timeline")
	assert.Equal(t, int64(0), unstarted.QuotaResetAt)

	assert.Equal(t, int64(0), expired.QuotaEffectiveAt, "expired anchor no longer shows the previous cycle")
	assert.Equal(t, int64(0), expired.QuotaResetAt)

	assert.Equal(t, int64(0), never.QuotaEffectiveAt, "never policy shows no timeline")
	assert.Equal(t, int64(0), never.QuotaResetAt)
}

func TestCycleResetClearsQuotaGrantedMidCycle(t *testing.T) {
	db := useQuotaResetDB(t)
	now := time.Now().Unix()
	anchor := now + 3600
	user := createQuotaResetUser(t, db, "mid", common.RoleCommonUser, 600, operation_setting.QuotaResetModeCycle, anchor)

	// 期中新增额度（含叠加发放）：只改余额，不清零时间不变，到期与原有额度一起清零
	require.NoError(t, db.Model(&User{}).Where("id = ?", user.Id).
		Update("quota", 900).Error)
	assert.Equal(t, anchor, userQuotaById(t, db, user.Id).QuotaNextResetAt,
		"mid-cycle top-up must not move the expiry")

	require.NoError(t, db.Model(&User{}).Where("id = ?", user.Id).
		Update("quota_next_reset_at", now-1).Error)
	count, err := ResetCycleQuotaUsers()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	after := userQuotaById(t, db, user.Id)
	assert.Equal(t, 0, after.Quota, "the whole balance including mid-cycle top-ups is cleared")
	assert.Equal(t, int64(0), after.QuotaNextResetAt,
		"anchor is consumed at cycle end; the next period starts at a new explicit effective time")
}

func TestGetConsumableUserQuota(t *testing.T) {
	db := useQuotaResetDB(t)
	now := time.Now().Unix()
	cycleSeconds := operation_setting.GetQuotaResetCycleSeconds()

	monthly := createQuotaResetUser(t, db, "m", common.RoleCommonUser, 100, operation_setting.QuotaResetModeMonthly, 0)
	never := createQuotaResetUser(t, db, "n", common.RoleCommonUser, 200, operation_setting.QuotaResetModeNever, 0)
	admin := createQuotaResetUser(t, db, "a", common.RoleAdminUser, 300, operation_setting.QuotaResetModeCycle, now+3600)
	adminUnstarted := createQuotaResetUser(t, db, "a-unstarted", common.RoleAdminUser, 320, operation_setting.QuotaResetModeCycle, 0)
	active := createQuotaResetUser(t, db, "act", common.RoleCommonUser, 400, operation_setting.QuotaResetModeCycle, now+3600)
	futureStart := createQuotaResetUser(t, db, "future", common.RoleCommonUser, 500, operation_setting.QuotaResetModeCycle, now+cycleSeconds+3600)
	unstarted := createQuotaResetUser(t, db, "unstarted", common.RoleCommonUser, 600, operation_setting.QuotaResetModeCycle, 0)
	due := createQuotaResetUser(t, db, "due", common.RoleCommonUser, 700, operation_setting.QuotaResetModeCycle, now-10)

	quota, err := GetConsumableUserQuota(monthly.Id)
	require.NoError(t, err)
	assert.Equal(t, 100, quota, "monthly users are always consumable")

	quota, err = GetConsumableUserQuota(never.Id)
	require.NoError(t, err)
	assert.Equal(t, 200, quota, "never users are always consumable")

	quota, err = GetConsumableUserQuota(admin.Id)
	require.NoError(t, err)
	assert.Equal(t, 300, quota, "admin on an active cycle is consumable like any user")

	_, err = GetConsumableUserQuota(adminUnstarted.Id)
	require.ErrorIs(t, err, ErrUserQuotaNotEffective, "admin follows the same cycle gate as everyone else")

	quota, err = GetConsumableUserQuota(active.Id)
	require.NoError(t, err)
	assert.Equal(t, 400, quota, "active cycle (effective time reached, anchor not due) is consumable")

	_, err = GetConsumableUserQuota(futureStart.Id)
	require.ErrorIs(t, err, ErrUserQuotaNotEffective, "quota before the effective time is not consumable")

	_, err = GetConsumableUserQuota(unstarted.Id)
	require.ErrorIs(t, err, ErrUserQuotaNotEffective, "cycle without a set effective time is not consumable")

	_, err = GetConsumableUserQuota(due.Id)
	require.ErrorIs(t, err, ErrUserQuotaNotEffective, "due cycle (awaiting tick clear) is not consumable")
}

func TestPrepareForInsertDefaultsResetModeByRole(t *testing.T) {
	db := useQuotaResetDB(t)

	admin := User{Username: "new-admin", Password: "test-password", Role: common.RoleAdminUser, AffCode: "aff-new-admin"}
	require.NoError(t, admin.prepareForInsert(db))
	assert.Equal(t, operation_setting.QuotaResetModeNever, admin.QuotaResetMode,
		"new admin accounts default to never reset")

	member := User{Username: "new-member", Password: "test-password", Role: common.RoleCommonUser, AffCode: "aff-new-member"}
	require.NoError(t, member.prepareForInsert(db))
	assert.Equal(t, operation_setting.QuotaResetModeMonthly, member.QuotaResetMode,
		"new common users still default to monthly reset")
}

func TestMigrateQuotaResetModesBackfillsLegacyEmptyValues(t *testing.T) {
	db := useQuotaResetDB(t)
	legacy := createQuotaResetUser(t, db, "legacy", common.RoleCommonUser, 100, "", 0)
	explicit := createQuotaResetUser(t, db, "explicit", common.RoleCommonUser, 100, operation_setting.QuotaResetModeCycle, 12345)
	require.NoError(t, db.Create(&Option{Key: "general_setting.quota_reset_mode", Value: "cycle"}).Error)
	require.NoError(t, db.Create(&Option{Key: quotaResetDefaultAppliedKey, Value: "cycle"}).Error)

	require.NoError(t, MigrateQuotaResetModes())

	assert.Equal(t, operation_setting.QuotaResetModeMonthly, userQuotaById(t, db, legacy.Id).QuotaResetMode,
		"legacy empty policy becomes monthly")
	assert.Equal(t, operation_setting.QuotaResetModeCycle, userQuotaById(t, db, explicit.Id).QuotaResetMode,
		"explicit policies are untouched")

	var count int64
	require.NoError(t, db.Model(&Option{}).
		Where(&Option{Key: "general_setting.quota_reset_mode"}).
		Count(&count).Error)
	assert.Equal(t, int64(0), count, "retired default-policy option is removed")
	require.NoError(t, db.Model(&Option{}).
		Where(&Option{Key: quotaResetDefaultAppliedKey}).
		Count(&count).Error)
	assert.Equal(t, int64(0), count, "retired alignment marker is removed")
}

func TestSetUserQuotaResetMode(t *testing.T) {
	db := useQuotaResetDB(t)
	withQuota := createQuotaResetUser(t, db, "u1", common.RoleCommonUser, 100, "", 0)
	noQuota := createQuotaResetUser(t, db, "u2", common.RoleCommonUser, 0, "", 0)
	admin := createQuotaResetUser(t, db, "root-like", common.RoleAdminUser, 100, "", 0)
	now := time.Now().Unix()
	cycleSeconds := operation_setting.GetQuotaResetCycleSeconds()

	// 有余额但未指定生效时间：不写入默认起算时间，等待运营指定
	next, err := SetUserQuotaResetMode(withQuota.Id, operation_setting.QuotaResetModeCycle, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), next)
	assert.Equal(t, int64(0), userQuotaById(t, db, withQuota.Id).QuotaNextResetAt)

	// 指定生效时间：锚点 = 生效时间 + 周期
	start := now - 3600
	next, err = SetUserQuotaResetMode(withQuota.Id, operation_setting.QuotaResetModeCycle, start)
	require.NoError(t, err)
	assert.Equal(t, start+cycleSeconds, next)
	assert.Equal(t, start+cycleSeconds, userQuotaById(t, db, withQuota.Id).QuotaNextResetAt)

	// 生效时间加周期已过期：拒绝（用户无进行中的周期）
	expiredTarget := createQuotaResetUser(t, db, "u3", common.RoleCommonUser, 100, "", 0)
	_, err = SetUserQuotaResetMode(expiredTarget.Id, operation_setting.QuotaResetModeCycle, now-cycleSeconds)
	assert.Error(t, err)

	// 周期进行中：禁止改生效时间，也不允许清空周期；同值重复提交幂等
	active := createQuotaResetUser(t, db, "u4", common.RoleCommonUser, 100, operation_setting.QuotaResetModeCycle, now+3600)
	_, err = SetUserQuotaResetMode(active.Id, operation_setting.QuotaResetModeCycle, now-3600)
	assert.Error(t, err, "cannot change the effective time within an active cycle")
	_, err = SetUserQuotaResetMode(active.Id, operation_setting.QuotaResetModeCycle, 0)
	assert.Error(t, err, "cannot clear an active cycle")
	next, err = SetUserQuotaResetMode(active.Id, operation_setting.QuotaResetModeCycle, now+3600-cycleSeconds)
	require.NoError(t, err)
	assert.Equal(t, now+3600, next, "resubmitting the current effective time keeps the anchor")

	// 无余额且未指定生效时间：不开始计时，也不因后续发量自动起算
	next, err = SetUserQuotaResetMode(noQuota.Id, operation_setting.QuotaResetModeCycle, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), next)
	assert.Equal(t, int64(0), userQuotaById(t, db, noQuota.Id).QuotaNextResetAt)

	// 无余额但指定生效时间：拒绝（设置生效周期必须要有额度）
	_, err = SetUserQuotaResetMode(noQuota.Id, operation_setting.QuotaResetModeCycle, now)
	assert.Error(t, err)

	// 转永不清零：锚点清空
	next, err = SetUserQuotaResetMode(withQuota.Id, operation_setting.QuotaResetModeNever, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), next)
	assert.Equal(t, int64(0), userQuotaById(t, db, withQuota.Id).QuotaNextResetAt)

	_, err = SetUserQuotaResetMode(withQuota.Id, "bogus", 0)
	assert.Error(t, err)

	next, err = SetUserQuotaResetMode(admin.Id, operation_setting.QuotaResetModeCycle, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), next)
	assert.Equal(t, operation_setting.QuotaResetModeCycle, userQuotaById(t, db, admin.Id).QuotaResetMode,
		"admin accounts can be configured like everyone else")
}
