package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEndpointTypesByChannelTypeShouldExposeVideoForDoubaoVideo(t *testing.T) {
	types := GetEndpointTypesByChannelType(constant.ChannelTypeDoubaoVideo, "doubao-seedance-2-0-260128")

	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAIVideo}, types)

	info, ok := GetDefaultEndpointInfo(constant.EndpointTypeOpenAIVideo)
	require.True(t, ok)
	assert.Equal(t, "/v1/video/generations", info.Path)
	assert.Equal(t, "POST", info.Method)
}

func TestGetEndpointTypesByChannelTypeShouldKeepVolcEngineAsOpenAI(t *testing.T) {
	types := GetEndpointTypesByChannelType(constant.ChannelTypeVolcEngine, "doubao-seedream-3-0")

	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAI}, types)
}
