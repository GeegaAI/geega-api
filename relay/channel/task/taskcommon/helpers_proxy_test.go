package taskcommon

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/assert"
)

func TestBuildNormalizedProxyURLShouldNormalizeServerAddress(t *testing.T) {
	original := system_setting.ServerAddress
	t.Cleanup(func() { system_setting.ServerAddress = original })

	cases := []struct {
		name    string
		address string
		want    string
	}{
		{name: "无后缀", address: "https://example.com", want: "https://example.com/v1/videos/t1/content"},
		{name: "已带 v1", address: "https://example.com/v1", want: "https://example.com/v1/videos/t1/content"},
		{name: "带尾斜杠", address: "https://example.com/v1/", want: "https://example.com/v1/videos/t1/content"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			system_setting.ServerAddress = tc.address
			assert.Equal(t, tc.want, BuildNormalizedProxyURL("t1"))
		})
	}
}
