package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 账号删除即清零余额，与清零策略无关（软删除保留记录但额度归 0）
func TestSoftDeletedUserQuotaIsCleared(t *testing.T) {
	setupUserUpdateTestState(t)
	user := createUserBindTestUser(t)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("quota", 123456).Error)

	require.NoError(t, user.Delete())

	var reloaded User
	require.NoError(t, DB.Unscoped().First(&reloaded, user.Id).Error)
	assert.Equal(t, 0, reloaded.Quota, "quota is cleared the moment the account is deleted")
	assert.True(t, reloaded.DeletedAt.Valid, "account is soft deleted")
}

// 硬删除（管理员删除）直接移除记录，删除后按账号已不存在处理
func TestHardDeletedUserRecordIsRemoved(t *testing.T) {
	setupUserUpdateTestState(t)
	user := createUserBindTestUser(t)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("quota", 123456).Error)

	require.NoError(t, HardDeleteUserById(user.Id))

	var reloaded User
	err := DB.Unscoped().First(&reloaded, user.Id).Error
	require.Error(t, err, "hard delete removes the record entirely")
	assert.Equal(t, 0, reloaded.Id)
}
