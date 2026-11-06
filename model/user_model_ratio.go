package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// UserModelRatio 为单个用户的指定模型配置专属模型倍率。
// 生效语义：覆盖全局模型倍率，分组倍率照常相乘；未配置的模型完全回退全局倍率。
type UserModelRatio struct {
	Id        int     `json:"id" gorm:"primaryKey"`
	UserId    int     `json:"user_id" gorm:"uniqueIndex:idx_user_model_ratios,priority:1"`
	ModelName string  `json:"model_name" gorm:"type:varchar(255);uniqueIndex:idx_user_model_ratios,priority:2"`
	Ratio     float64 `json:"ratio"`
}

const userModelRatioCacheTTL = 60 * time.Second

type userModelRatioCacheEntry struct {
	ratios    map[string]float64
	expiresAt time.Time
}

var (
	userModelRatioCache   = make(map[int]userModelRatioCacheEntry)
	userModelRatioCacheMu sync.RWMutex
)

func userModelRatioCacheKey(userId int) string {
	return fmt.Sprintf("user_model_ratio:%d", userId)
}

func userModelRatioRedisTTLSeconds() int {
	ttl := common.RedisKeyCacheSeconds()
	if ttl <= 0 {
		return 60
	}
	return ttl
}

func copyUserModelRatios(src map[string]float64) map[string]float64 {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]float64, len(src))
	for name, ratio := range src {
		dst[name] = ratio
	}
	return dst
}

// GetUserModelRatios 返回用户全部专属模型倍率（模型名→倍率）；未配置时返回 nil。
// 读取顺序：进程内存缓存 → Redis → 数据库；缓存由写路径失效，内存缓存 TTL 兜底多节点陈旧窗口。
func GetUserModelRatios(userId int) map[string]float64 {
	if userId <= 0 {
		return nil
	}
	now := time.Now()
	userModelRatioCacheMu.RLock()
	entry, cached := userModelRatioCache[userId]
	userModelRatioCacheMu.RUnlock()
	if cached && now.Before(entry.expiresAt) {
		return copyUserModelRatios(entry.ratios)
	}

	ratios := loadUserModelRatios(userId)

	userModelRatioCacheMu.Lock()
	userModelRatioCache[userId] = userModelRatioCacheEntry{ratios: ratios, expiresAt: now.Add(userModelRatioCacheTTL)}
	userModelRatioCacheMu.Unlock()
	return copyUserModelRatios(ratios)
}

// GetUserModelRatio 返回用户对指定模型的专属倍率；未配置时 ok=false。
// 计费热路径入口：命中内存缓存时零分配。
func GetUserModelRatio(userId int, modelName string) (float64, bool) {
	if userId <= 0 || modelName == "" {
		return 0, false
	}
	now := time.Now()
	userModelRatioCacheMu.RLock()
	entry, cached := userModelRatioCache[userId]
	userModelRatioCacheMu.RUnlock()
	if cached && now.Before(entry.expiresAt) {
		ratio, ok := entry.ratios[modelName]
		return ratio, ok
	}

	ratios := GetUserModelRatios(userId)
	if len(ratios) == 0 {
		return 0, false
	}
	ratio, ok := ratios[modelName]
	return ratio, ok
}

func loadUserModelRatios(userId int) map[string]float64 {
	if common.RedisEnabled {
		if ratios, ok := readUserModelRatiosFromRedis(userId); ok {
			return ratios
		}
	}
	var rows []UserModelRatio
	if err := DB.Where("user_id = ?", userId).Find(&rows).Error; err != nil {
		common.SysError(fmt.Sprintf("failed to load user model ratios for user %d: %s", userId, err.Error()))
		return nil
	}
	if len(rows) == 0 {
		return nil
	}
	ratios := make(map[string]float64, len(rows))
	for _, row := range rows {
		ratios[row.ModelName] = row.Ratio
	}
	if common.RedisEnabled {
		writeUserModelRatiosToRedis(userId, ratios)
	}
	return ratios
}

func readUserModelRatiosFromRedis(userId int) (map[string]float64, bool) {
	data, err := common.RDB.Get(context.Background(), userModelRatioCacheKey(userId)).Result()
	if err != nil {
		return nil, false
	}
	var ratios map[string]float64
	if err := json.Unmarshal([]byte(data), &ratios); err != nil {
		return nil, false
	}
	if len(ratios) == 0 {
		return nil, false
	}
	return ratios, true
}

func writeUserModelRatiosToRedis(userId int, ratios map[string]float64) {
	data, err := json.Marshal(ratios)
	if err != nil {
		return
	}
	if err := common.RDB.Set(context.Background(), userModelRatioCacheKey(userId), data,
		time.Duration(userModelRatioRedisTTLSeconds())*time.Second).Err(); err != nil {
		common.SysError(fmt.Sprintf("failed to cache user model ratios for user %d: %s", userId, err.Error()))
	}
}

// ReplaceUserModelRatios 全量替换用户的专属模型倍率；ratios 为空时清空全部配置。
// 覆盖值语义为替换全局模型倍率，允许 0（该用户该模型免费），禁止负数与非有限值。
func ReplaceUserModelRatios(userId int, ratios map[string]float64) error {
	if userId <= 0 {
		return errors.New("invalid user id")
	}
	for name, ratio := range ratios {
		if name == "" {
			return errors.New("model name is empty")
		}
		if ratio < 0 || ratio >= 1 || math.IsInf(ratio, 0) || math.IsNaN(ratio) {
			return fmt.Errorf("invalid model discount for %s: must be a number in [0, 1)", name)
		}
	}
	var count int64
	if err := DB.Model(&User{}).Where("id = ?", userId).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("user not found")
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userId).Delete(&UserModelRatio{}).Error; err != nil {
			return err
		}
		if len(ratios) == 0 {
			return nil
		}
		rows := make([]UserModelRatio, 0, len(ratios))
		for name, ratio := range ratios {
			rows = append(rows, UserModelRatio{UserId: userId, ModelName: name, Ratio: ratio})
		}
		return tx.Create(&rows).Error
	})
	if err != nil {
		return err
	}
	invalidateUserModelRatioCache(userId)
	return nil
}

func invalidateUserModelRatioCache(userId int) {
	userModelRatioCacheMu.Lock()
	delete(userModelRatioCache, userId)
	userModelRatioCacheMu.Unlock()
	if common.RedisEnabled {
		if err := common.RDB.Del(context.Background(), userModelRatioCacheKey(userId)).Err(); err != nil {
			common.SysError(fmt.Sprintf("failed to invalidate user model ratio cache for user %d: %s", userId, err.Error()))
		}
	}
}
