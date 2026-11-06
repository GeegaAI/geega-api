package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"gorm.io/gorm"
)

// quotaResetDefaultAppliedKey 记录最近一次已对齐锚点的默认策略值，
// 该函数经 option 同步链路周期性重放，须防止锚点被反复重置
const quotaResetDefaultAppliedKey = "QuotaResetDefaultModeApplied"

// applyQuotaModeCond 追加"清零策略 = mode"过滤（每个用户都有明确策略，无继承）
func applyQuotaModeCond(tx *gorm.DB, mode string) *gorm.DB {
	return tx.Where("quota_reset_mode = ?", mode)
}

// MigrateQuotaResetModes 一次性数据迁移：历史空值（原"跟随系统默认"）落为每月固定日清零，
// 同时清理已废弃的系统默认策略配置项。幂等，可随启动重复执行
func MigrateQuotaResetModes() error {
	result := DB.Model(&User{}).
		Where("quota_reset_mode IS NULL OR quota_reset_mode = ''").
		Update("quota_reset_mode", operation_setting.QuotaResetModeMonthly)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		common.SysLog(fmt.Sprintf("quota reset: migrated %d users with unset reset policy to monthly", result.RowsAffected))
	}
	legacyKeys := []string{"general_setting.quota_reset_mode", quotaResetDefaultAppliedKey}
	for _, key := range legacyKeys {
		if err := DB.Where(&Option{Key: key}).Delete(&Option{}).Error; err != nil {
			return err
		}
	}
	return nil
}

// invalidateUserCaches 批量失效用户缓存
func invalidateUserCaches(users []User) {
	for _, u := range users {
		if err := invalidateUserCache(u.Id); err != nil {
			common.SysLog(fmt.Sprintf("quota reset: failed to invalidate cache for user %d: %s", u.Id, err.Error()))
		}
	}
}

func invalidateUserCachesByIds(ids []int) {
	for _, id := range ids {
		if err := invalidateUserCache(id); err != nil {
			common.SysLog(fmt.Sprintf("quota reset: failed to invalidate cache for user %d: %s", id, err.Error()))
		}
	}
}

func userIds(users []User) []int {
	ids := make([]int, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.Id)
	}
	return ids
}

// resetMonthlyCohort 将生效策略为 monthly 的用户（含禁用与管理员）额度清零，
// 返回受影响用户 id 与人数。仅处理额度非 0 的用户，减少无效缓存失效
func resetMonthlyCohort() ([]int, int64, error) {
	FlushBatchUpdates()
	var users []User
	err := applyQuotaModeCond(DB.Where("quota != 0"), operation_setting.QuotaResetModeMonthly).
		Select("id").Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	result := applyQuotaModeCond(DB.Model(&User{}).Where("quota != 0"), operation_setting.QuotaResetModeMonthly).
		Update("quota", 0)
	if result.Error != nil {
		return nil, 0, result.Error
	}
	return userIds(users), result.RowsAffected, nil
}

// EffectiveQuotaResetMode 用户生效清零策略（历史空值兜底为每月固定日清零）
func (u *User) EffectiveQuotaResetMode() string {
	if u.QuotaResetMode == "" {
		return operation_setting.QuotaResetModeMonthly
	}
	return u.QuotaResetMode
}

// DefaultQuotaResetMode 新建用户的默认清零策略：管理员账号默认永不清零，其余每月固定日清零
func DefaultQuotaResetMode(role int) string {
	if role >= common.RoleAdminUser {
		return operation_setting.QuotaResetModeNever
	}
	return operation_setting.QuotaResetModeMonthly
}

// ResetMonthlyQuotaUsers 将生效策略为 monthly 的用户额度清零，返回受影响人数
func ResetMonthlyQuotaUsers() (int64, error) {
	ids, count, err := resetMonthlyCohort()
	if err != nil {
		return 0, err
	}
	invalidateUserCachesByIds(ids)
	return count, nil
}

// ResetCycleQuotaUsers 清零已到期的周期清零用户，并同时清空锚点：
// 周期终结后回到"未开始计时"状态，续期需运营显式设置新生效时间；
// 归零后（发量不再自动起算）续期发量不会被下一轮 tick 误清
func ResetCycleQuotaUsers() (int64, error) {
	FlushBatchUpdates()
	due := func() *gorm.DB {
		return applyQuotaModeCond(DB.Model(&User{}).
			Where("quota != 0").
			Where("quota_next_reset_at > 0 AND quota_next_reset_at <= ?", common.GetTimestamp()),
			operation_setting.QuotaResetModeCycle)
	}
	var users []User
	if err := due().Select("id").Find(&users).Error; err != nil {
		return 0, err
	}
	result := due().Updates(map[string]interface{}{
		"quota":               0,
		"quota_next_reset_at": 0,
	})
	if result.Error != nil {
		return 0, result.Error
	}
	invalidateUserCaches(users)
	return result.RowsAffected, nil
}

// SetUserQuotaResetMode 设置单个用户的清零策略。
// 转为 cycle 时：
//   - effectiveAt > 0：锚点 = 生效时间 + 周期（运营指定的周期起算时刻），生效时间加周期不得早于当前时刻；
//   - effectiveAt == 0：不写默认起算时间，等待运营显式设置生效时间
//
// 周期进行中（锚点未到期）一律拒绝改动，避免绕过入口改写或静默重开计时。
// 转出 cycle 时清空锚点。返回 cycle 模式的下次清零时间（未开始计时或非 cycle 恒 0）
func SetUserQuotaResetMode(userId int, mode string, effectiveAt int64) (int64, error) {
	if !operation_setting.IsValidQuotaResetMode(mode) {
		return 0, errors.New("非法的清零策略")
	}
	var user User
	if err := DB.Select("id, quota, quota_reset_mode, quota_next_reset_at").First(&user, userId).Error; err != nil {
		return 0, err
	}
	now := common.GetTimestamp()
	cycleSeconds := operation_setting.GetQuotaResetCycleSeconds()
	updates := map[string]interface{}{"quota_reset_mode": mode}
	next := int64(0)
	if mode == operation_setting.QuotaResetModeCycle {
		// 周期进行中不得改生效时间，也不允许清空周期
		if user.QuotaNextResetAt > now &&
			(effectiveAt <= 0 || effectiveAt+cycleSeconds != user.QuotaNextResetAt) {
			return 0, errors.New("当前处于生效周期内，不能修改额度生效时间")
		}
		if effectiveAt > 0 {
			// 设置生效时间必须已有额度：无余额的周期用户需先分配额度
			if user.Quota <= 0 {
				return 0, errors.New("设置生效时间前请先为该用户分配额度")
			}
			next = effectiveAt + cycleSeconds
			if next <= now {
				return 0, errors.New("额度生效时间加周期已早于当前时间，请调整生效时间")
			}
		}
		updates["quota_next_reset_at"] = next
	} else {
		updates["quota_next_reset_at"] = 0
	}
	if err := DB.Model(&User{}).Where("id = ?", userId).Updates(updates).Error; err != nil {
		return 0, err
	}
	return next, nil
}

// ErrUserQuotaNotEffective 周期清零用户钱包额度未进入生效期，不可消费
var ErrUserQuotaNotEffective = errors.New("quota is not yet effective")

// GetConsumableUserQuota 返回用户钱包余额，并对周期清零用户做生效期门禁。
// 生效期 = [锚点-周期, 锚点)；期外（未设置生效时间 / 生效时间未到 / 周期已到期待清零）
// 返回 ErrUserQuotaNotEffective，防止未生效的额度被消费。锚点为 0（未设置生效时间）视为未生效，
// 避免"发量后未设生效时间"的余额被无限期使用。monthly / never 恒可消费。
func GetConsumableUserQuota(userId int) (int, error) {
	var user User
	if err := DB.Select("id, quota, quota_reset_mode, quota_next_reset_at").First(&user, userId).Error; err != nil {
		return 0, err
	}
	if user.EffectiveQuotaResetMode() != operation_setting.QuotaResetModeCycle {
		return user.Quota, nil
	}
	if user.QuotaNextResetAt <= 0 {
		return 0, fmt.Errorf("%w: 等待管理员设置额度生效时间", ErrUserQuotaNotEffective)
	}
	now := common.GetTimestamp()
	effectiveAt := user.QuotaNextResetAt - operation_setting.GetQuotaResetCycleSeconds()
	if now < effectiveAt {
		return 0, fmt.Errorf("%w: 生效时间为 %s", ErrUserQuotaNotEffective,
			time.Unix(effectiveAt, 0).Format("2006-01-02 15:04:05"))
	}
	if now >= user.QuotaNextResetAt {
		return 0, fmt.Errorf("%w: 周期已到期，额度待清零", ErrUserQuotaNotEffective)
	}
	return user.Quota, nil
}
