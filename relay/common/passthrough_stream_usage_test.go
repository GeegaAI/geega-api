package common

import (
	"testing"

	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func streamRelayInfo(stream bool, supportStreamOptions bool) *RelayInfo {
	return &RelayInfo{
		IsStream:  stream,
		RelayMode: relayconstant.RelayModeChatCompletions,
		ChannelMeta: &ChannelMeta{
			SupportStreamOptions: supportStreamOptions,
		},
	}
}

func TestInjectPassThroughStreamUsageAddsMissingOption(t *testing.T) {
	body := []byte(`{"model":"gpt-4o","stream":true,"messages":[{"role":"user","content":"hi"}]}`)

	patched, injected, err := InjectPassThroughStreamUsage(body, streamRelayInfo(true, true))
	require.NoError(t, err)
	require.True(t, injected)
	assert.True(t, gjson.GetBytes(patched, "stream_options.include_usage").Bool())
	// 其余字段保持不变
	assert.Equal(t, "gpt-4o", gjson.GetBytes(patched, "model").String())
	assert.Equal(t, "hi", gjson.GetBytes(patched, "messages.0.content").String())
}

func TestInjectPassThroughStreamUsageOverridesFalse(t *testing.T) {
	body := []byte(`{"model":"gpt-4o","stream":true,"stream_options":{"include_usage":false}}`)

	patched, injected, err := InjectPassThroughStreamUsage(body, streamRelayInfo(true, true))
	require.NoError(t, err)
	require.True(t, injected)
	assert.True(t, gjson.GetBytes(patched, "stream_options.include_usage").Bool())
}

func TestInjectPassThroughStreamUsageKeepsSiblingOptions(t *testing.T) {
	body := []byte(`{"stream":true,"stream_options":{"include_usage":false,"continuous_usage_stats":true}}`)

	patched, injected, err := InjectPassThroughStreamUsage(body, streamRelayInfo(true, true))
	require.NoError(t, err)
	require.True(t, injected)
	assert.True(t, gjson.GetBytes(patched, "stream_options.include_usage").Bool())
	assert.True(t, gjson.GetBytes(patched, "stream_options.continuous_usage_stats").Bool())
}

func TestInjectPassThroughStreamUsageSkipsWhenAlreadyEnabled(t *testing.T) {
	body := []byte(`{"stream":true,"stream_options":{"include_usage":true}}`)

	patched, injected, err := InjectPassThroughStreamUsage(body, streamRelayInfo(true, true))
	require.NoError(t, err)
	assert.False(t, injected)
	assert.Equal(t, body, patched)
}

func TestInjectPassThroughStreamUsageSkipsNonStream(t *testing.T) {
	body := []byte(`{"stream":false}`)

	patched, injected, err := InjectPassThroughStreamUsage(body, streamRelayInfo(false, true))
	require.NoError(t, err)
	assert.False(t, injected)
	assert.Equal(t, body, patched)
}

func TestInjectPassThroughStreamUsageSkipsBodyStreamFalse(t *testing.T) {
	body := []byte(`{"stream":false}`)

	patched, injected, err := InjectPassThroughStreamUsage(body, streamRelayInfo(true, true))
	require.NoError(t, err)
	assert.False(t, injected)
	assert.Equal(t, body, patched)
}

func TestInjectPassThroughStreamUsageSkipsUnsupportedChannel(t *testing.T) {
	body := []byte(`{"stream":true}`)

	patched, injected, err := InjectPassThroughStreamUsage(body, streamRelayInfo(true, false))
	require.NoError(t, err)
	assert.False(t, injected)
	assert.Equal(t, body, patched)
}

func TestInjectPassThroughStreamUsageSkipsNonChatMode(t *testing.T) {
	body := []byte(`{"stream":true}`)
	info := streamRelayInfo(true, true)
	info.RelayMode = relayconstant.RelayModeCompletions

	patched, injected, err := InjectPassThroughStreamUsage(body, info)
	require.NoError(t, err)
	assert.False(t, injected)
	assert.Equal(t, body, patched)
}

func TestInjectPassThroughStreamUsageSkipsInvalidOrWeirdBodies(t *testing.T) {
	info := streamRelayInfo(true, true)

	for _, body := range [][]byte{
		[]byte(`not-json`),
		[]byte(`[]`),
		[]byte(``),
		[]byte(`{"stream":true,"stream_options":null}`),
		[]byte(`{"stream":true,"stream_options":"weird"}`),
	} {
		patched, injected, err := InjectPassThroughStreamUsage(body, info)
		require.NoError(t, err)
		assert.False(t, injected, "body=%q", body)
		assert.Equal(t, body, patched, "body=%q", body)
	}
}

func TestInjectPassThroughStreamUsageNilInfo(t *testing.T) {
	body := []byte(`{"stream":true}`)

	patched, injected, err := InjectPassThroughStreamUsage(body, nil)
	require.NoError(t, err)
	assert.False(t, injected)
	assert.Equal(t, body, patched)
}
