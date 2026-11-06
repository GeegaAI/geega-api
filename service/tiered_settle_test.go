package service

import (
	"testing"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestRefreshTieredBillingGroup_applies_user_ratio(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{
		PriceData: hosttypes.PriceData{
			GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 2},
		},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:               "tiered_expr",
			GroupRatio:                1,
			UserRatio:                 0.1,
			HasUserRatio:              true,
			EstimatedQuotaBeforeGroup: 750,
			EstimatedQuotaAfterGroup:  750,
		},
	}

	snap, err := refreshTieredBillingGroup(relayInfo)
	require.NoError(t, err)
	require.NotNil(t, snap)
	// 750 × 2.0 × 0.1 = 150
	require.Equal(t, 150, snap.EstimatedQuotaAfterGroup)
	require.Equal(t, 2.0, snap.GroupRatio)
}

func TestRefreshTieredBillingGroup_legacy_zero_user_ratio_falls_back_to_one(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{
		PriceData: hosttypes.PriceData{
			GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 2},
		},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:               "tiered_expr",
			GroupRatio:                1,
			UserRatio:                 0,
			EstimatedQuotaBeforeGroup: 750,
			EstimatedQuotaAfterGroup:  750,
		},
	}

	snap, err := refreshTieredBillingGroup(relayInfo)
	require.NoError(t, err)
	require.NotNil(t, snap)
	// 旧快照无 UserRatio：750 × 2.0 × 1 = 1500
	require.Equal(t, 1500, snap.EstimatedQuotaAfterGroup)
}

func TestRefreshTieredBillingGroup_explicit_zero_user_ratio_means_free(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{
		PriceData: hosttypes.PriceData{
			GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 2},
		},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:               "tiered_expr",
			GroupRatio:                1,
			UserRatio:                 0,
			HasUserRatio:              true,
			EstimatedQuotaBeforeGroup: 750,
			EstimatedQuotaAfterGroup:  0,
		},
	}

	snap, err := refreshTieredBillingGroup(relayInfo)
	require.NoError(t, err)
	require.NotNil(t, snap)
	// 显式 0 系数即免费：换组重估后仍为 0，不能被当成未配置
	require.Equal(t, 0, snap.EstimatedQuotaAfterGroup)
}

func TestRefreshTieredBillingGroup_unchanged_group_returns_snapshot_as_is(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{
		PriceData: hosttypes.PriceData{
			GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1},
		},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:               "tiered_expr",
			GroupRatio:                1,
			UserRatio:                 0.5,
			EstimatedQuotaBeforeGroup: 750,
			EstimatedQuotaAfterGroup:  375,
		},
	}

	snap, err := refreshTieredBillingGroup(relayInfo)
	require.NoError(t, err)
	require.Same(t, relayInfo.TieredBillingSnapshot, snap)
	require.Equal(t, 375, snap.EstimatedQuotaAfterGroup)
}
