package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelInfoValueReturnsString(t *testing.T) {
	value, err := (ChannelInfo{
		IsMultiKey:         true,
		MultiKeySize:       2,
		MultiKeyStatusList: map[int]int{0: 1},
	}).Value()

	require.NoError(t, err)
	valueStr, ok := value.(string)
	require.True(t, ok)
	require.Contains(t, valueStr, "\"is_multi_key\":true")
}

func TestChannelInfoScanSupportsString(t *testing.T) {
	var info ChannelInfo

	err := info.Scan(`{"is_multi_key":true,"multi_key_size":3,"multi_key_status_list":{"1":2},"multi_key_polling_index":1,"multi_key_mode":"random"}`)
	require.NoError(t, err)
	require.True(t, info.IsMultiKey)
	require.Equal(t, 3, info.MultiKeySize)
	require.Equal(t, 2, info.MultiKeyStatusList[1])
	require.Equal(t, 1, info.MultiKeyPollingIndex)
}
