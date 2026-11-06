package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withVideoTierPriceConfig(t *testing.T, jsonStr string) {
	t.Helper()
	snapshot := videoTierPriceMap.ReadAll()
	t.Cleanup(func() {
		videoTierPriceMap.Clear()
		videoTierPriceMap.AddAll(snapshot)
	})
	require.NoError(t, UpdateVideoTierPriceByJSONString(jsonStr))
}

func TestGetTaskTierRatio_ConfiguredTierShouldReturnRelativePrice(t *testing.T) {
	withVideoTierPriceConfig(t, `{"m1":{"720p":46,"1080p":51}}`)

	ratio, ok := GetTaskTierRatio("m1", "1080p", false)
	assert.True(t, ok)
	assert.InDelta(t, 51.0/46.0, ratio, 1e-9)

	ratio, ok = GetTaskTierRatio("m1", "720p", false)
	assert.True(t, ok)
	assert.InDelta(t, 1.0, ratio, 1e-9)
}

func TestGetTaskTierRatio_MissingTierShouldFallBackToBase(t *testing.T) {
	withVideoTierPriceConfig(t, `{"m1":{"720p":46,"1080p":51}}`)

	ratio, ok := GetTaskTierRatio("m1", "4k", false)
	assert.True(t, ok)
	assert.InDelta(t, 1.0, ratio, 1e-9)
}

func TestGetTaskTierRatio_VideoVariantShouldUseVideoKey(t *testing.T) {
	withVideoTierPriceConfig(t, `{"m1":{"720p":46,"720p_video":28}}`)

	ratio, ok := GetTaskTierRatio("m1", "720p", true)
	assert.True(t, ok)
	assert.InDelta(t, 28.0/46.0, ratio, 1e-9)
}

func TestGetTaskTierRatio_UnknownModelShouldNotClaimConfig(t *testing.T) {
	withVideoTierPriceConfig(t, `{"m1":{"720p":46}}`)

	_, ok := GetTaskTierRatio("m2", "720p", false)
	assert.False(t, ok)
}

func TestGetTaskTierRatio_ResolutionShouldNormalizeCase(t *testing.T) {
	withVideoTierPriceConfig(t, `{"m1":{"720p":46,"1080p":51}}`)

	ratio, ok := GetTaskTierRatio("m1", " 1080P ", false)
	assert.True(t, ok)
	assert.InDelta(t, 51.0/46.0, ratio, 1e-9)
}

func TestUpdateVideoTierPriceByJSONString_ShouldRejectUnknownTierKey(t *testing.T) {
	err := UpdateVideoTierPriceByJSONString(`{"m1":{"144p":1,"720p":46}}`)
	assert.Error(t, err)
}

func TestUpdateVideoTierPriceByJSONString_ShouldRejectNonPositivePrice(t *testing.T) {
	err := UpdateVideoTierPriceByJSONString(`{"m1":{"720p":0}}`)
	assert.Error(t, err)
}

func TestUpdateVideoTierPriceByJSONString_ShouldRejectMissingBaseTier(t *testing.T) {
	err := UpdateVideoTierPriceByJSONString(`{"m1":{"1080p":51}}`)
	assert.Error(t, err)
}

func TestUpdateVideoTierPriceByJSONString_ShouldIgnoreEmptyModelEntry(t *testing.T) {
	withVideoTierPriceConfig(t, `{"m1":{"720p":46},"empty":{}}`)

	_, ok := GetTaskTierRatio("empty", "720p", false)
	assert.False(t, ok)

	_, ok = GetTaskTierRatio("m1", "720p", false)
	assert.True(t, ok)
}
