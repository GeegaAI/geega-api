package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var userModelRatioTestUserSeq int

func setupUserModelRatioTest(t *testing.T) {
	t.Helper()
	truncateTables(t)

	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = oldRedisEnabled
	})
	// 清空进程内 L1 缓存，避免用例间通过缓存互相污染
	userModelRatioCacheMu.Lock()
	userModelRatioCache = make(map[int]userModelRatioCacheEntry)
	userModelRatioCacheMu.Unlock()
}

func createModelRatioTestUser(t *testing.T) User {
	t.Helper()
	userModelRatioTestUserSeq++
	user := User{
		Username:    fmt.Sprintf("model-ratio-user-%d", userModelRatioTestUserSeq),
		Password:    "unused-password-hash",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AuthVersion: 1,
		AffCode:     fmt.Sprintf("model-ratio-aff-%d", userModelRatioTestUserSeq),
	}
	require.NoError(t, DB.Create(&user).Error)
	return user
}

func TestGetUserModelRatio_should_return_override_when_configured(t *testing.T) {
	setupUserModelRatioTest(t)
	user := createModelRatioTestUser(t)

	require.NoError(t, ReplaceUserModelRatios(user.Id, map[string]float64{
		"gpt-4o": 0.8,
		"o3":     0,
	}))

	ratio, ok := GetUserModelRatio(user.Id, "gpt-4o")
	assert.True(t, ok)
	assert.Equal(t, 0.8, ratio)

	freeRatio, freeOk := GetUserModelRatio(user.Id, "o3")
	assert.True(t, freeOk)
	assert.Equal(t, 0.0, freeRatio)

	_, missing := GetUserModelRatio(user.Id, "gpt-4.1")
	assert.False(t, missing)
}

func TestGetUserModelRatio_should_return_false_when_user_has_no_override(t *testing.T) {
	setupUserModelRatioTest(t)
	user := createModelRatioTestUser(t)

	_, ok := GetUserModelRatio(user.Id, "gpt-4o")
	assert.False(t, ok)

	// 未配置用户不落任何行
	var count int64
	require.NoError(t, DB.Model(&UserModelRatio{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestReplaceUserModelRatios_should_full_replace_previous_config(t *testing.T) {
	setupUserModelRatioTest(t)
	user := createModelRatioTestUser(t)

	require.NoError(t, ReplaceUserModelRatios(user.Id, map[string]float64{
		"gpt-4o": 0.8,
		"o3":     0.2,
	}))
	require.NoError(t, ReplaceUserModelRatios(user.Id, map[string]float64{
		"gpt-4.1": 0.5,
	}))

	ratios := GetUserModelRatios(user.Id)
	require.Len(t, ratios, 1)
	assert.Equal(t, 0.5, ratios["gpt-4.1"])
	_, ok := GetUserModelRatio(user.Id, "gpt-4o")
	assert.False(t, ok)
}

func TestReplaceUserModelRatios_should_clear_all_when_ratios_empty(t *testing.T) {
	setupUserModelRatioTest(t)
	user := createModelRatioTestUser(t)

	require.NoError(t, ReplaceUserModelRatios(user.Id, map[string]float64{"gpt-4o": 0.8}))
	require.NoError(t, ReplaceUserModelRatios(user.Id, map[string]float64{}))

	assert.Nil(t, GetUserModelRatios(user.Id))
	_, ok := GetUserModelRatio(user.Id, "gpt-4o")
	assert.False(t, ok)
}

func TestReplaceUserModelRatios_should_reject_invalid_input(t *testing.T) {
	setupUserModelRatioTest(t)
	user := createModelRatioTestUser(t)

	err := ReplaceUserModelRatios(user.Id, map[string]float64{"gpt-4o": -0.1})
	assert.Error(t, err)

	err = ReplaceUserModelRatios(user.Id, map[string]float64{"gpt-4o": 1.0})
	assert.Error(t, err)

	err = ReplaceUserModelRatios(user.Id, map[string]float64{"gpt-4o": 1.5})
	assert.Error(t, err)

	err = ReplaceUserModelRatios(user.Id, map[string]float64{"": 0.5})
	assert.Error(t, err)

	err = ReplaceUserModelRatios(999999, map[string]float64{"gpt-4o": 0.5})
	assert.Error(t, err)

	// 非法输入不得落库
	var count int64
	require.NoError(t, DB.Model(&UserModelRatio{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestGetUserModelRatio_should_isolate_users(t *testing.T) {
	setupUserModelRatioTest(t)
	userA := createModelRatioTestUser(t)
	userB := createModelRatioTestUser(t)

	require.NoError(t, ReplaceUserModelRatios(userA.Id, map[string]float64{"gpt-4o": 0.8}))

	ratioA, okA := GetUserModelRatio(userA.Id, "gpt-4o")
	assert.True(t, okA)
	assert.Equal(t, 0.8, ratioA)

	_, okB := GetUserModelRatio(userB.Id, "gpt-4o")
	assert.False(t, okB)
}
