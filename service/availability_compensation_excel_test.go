package service

import (
	"bytes"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestBuildAvailabilityCompensationWorkbook(t *testing.T) {
	result := &model.AvailabilityCompensationResult{
		StartTimestamp:     1789228800,
		EndTimestamp:       1789228800 + 16*24*3600,
		TotalMinutes:       16 * 24 * 60,
		DataStartTimestamp: 1789228800,
		DataEndTimestamp:   1789228800 + 16*24*60*60 - 1,
		ChannelCount:       1,
		ModelCount:         1,
		SupplierCount:      1,
		Rows: []model.AvailabilityCompensationRow{
			{
				ModelName:               "model-a",
				ChannelId:               101,
				ChannelName:             "Acme_model-a",
				Supplier:                "Acme",
				OkRequests:              160,
				FailAllRequests:         40,
				FailServiceRequests:     30,
				ErrorRateAll:            20,
				ErrorRateService:        15.789,
				ErrorMinutesGe50All:     20,
				ErrorMinutesAnyAll:      30,
				ErrorMinutesGe50Service: 10,
				ErrorMinutesAnyService:  20,
				AvailabilityGe50All:     99.913,
				AvailabilityAnyAll:      99.87,
				AvailabilityGe50Service: 99.957,
				AvailabilityAnyService:  99.913,
				TierAll:                 model.AvailabilityCompensationTierNone,
				TierService:             model.AvailabilityCompensationTierNone,
			},
		},
		Suppliers: []model.AvailabilityCompensationSupplierRow{
			{
				Supplier:                "Acme",
				ChannelCount:            1,
				OkRequests:              160,
				FailAllRequests:         40,
				FailServiceRequests:     30,
				ErrorRateAll:            20,
				ErrorRateService:        15.789,
				ErrorMinutesGe50All:     20,
				ErrorMinutesAnyAll:      30,
				ErrorMinutesGe50Service: 10,
				ErrorMinutesAnyService:  20,
				AvailabilityGe50All:     99.913,
				AvailabilityAnyAll:      99.87,
				AvailabilityGe50Service: 99.957,
				AvailabilityAnyService:  99.913,
				TierAll:                 model.AvailabilityCompensationTierNone,
				TierService:             model.AvailabilityCompensationTierNone,
			},
		},
		Channels: []model.AvailabilityCompensationChannel{
			{Id: 101, Name: "Acme_model-a", Status: common.ChannelStatusEnabled, Supplier: "Acme"},
		},
	}

	data, err := BuildAvailabilityCompensationWorkbook(result)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()

	assert.Equal(t, []string{"说明", "业界对标", "版本1-全部错误归供应商", "版本2-剔除系统自身错误", "供应商汇总", "渠道对照表"}, f.GetSheetList())

	version1, err := f.GetRows("版本1-全部错误归供应商")
	require.NoError(t, err)
	require.Len(t, version1, 2)
	assert.Equal(t, "模型", version1[0][0])
	assert.Equal(t, "错误请求(全部)", version1[0][5])
	assert.Equal(t, "model-a", version1[1][0])
	assert.Equal(t, "40", version1[1][5])
	assert.Equal(t, "无赔偿", version1[1][11])

	version2, err := f.GetRows("版本2-剔除系统自身错误")
	require.NoError(t, err)
	require.Len(t, version2, 2)
	assert.Equal(t, "服务端错误", version2[0][5])
	assert.Equal(t, "30", version2[1][5])

	suppliers, err := f.GetRows("供应商汇总")
	require.NoError(t, err)
	require.Len(t, suppliers, 2)
	assert.Equal(t, "Acme", suppliers[1][0])
	assert.Contains(t, suppliers[1][4], "99.913")

	channels, err := f.GetRows("渠道对照表")
	require.NoError(t, err)
	require.Len(t, channels, 2)
	assert.Equal(t, "启用", channels[1][3])
}
