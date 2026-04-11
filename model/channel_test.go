package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestNormalizeJSONFieldsForCreate_EmptyJSONFieldsBecomeValidJSON(t *testing.T) {
	modelMapping := "   "
	statusCodeMapping := ""
	setting := " \n "
	paramOverride := ""
	headerOverride := "\t"

	channel := &Channel{
		ModelMapping:      &modelMapping,
		StatusCodeMapping: &statusCodeMapping,
		Setting:           &setting,
		ParamOverride:     &paramOverride,
		HeaderOverride:    &headerOverride,
	}

	require.NoError(t, channel.NormalizeJSONFieldsForCreate())
	require.NotNil(t, channel.ModelMapping)
	require.NotNil(t, channel.StatusCodeMapping)
	require.NotNil(t, channel.Setting)
	require.NotNil(t, channel.ParamOverride)
	require.NotNil(t, channel.HeaderOverride)
	require.Equal(t, "{}", *channel.ModelMapping)
	require.Equal(t, "{}", *channel.StatusCodeMapping)
	require.Equal(t, "{}", *channel.Setting)
	require.Equal(t, "{}", *channel.ParamOverride)
	require.Equal(t, "{}", *channel.HeaderOverride)
	require.Equal(t, "{}", channel.OtherSettings)
	require.Equal(t, "{}", channel.OtherInfo)
}

func TestNormalizeJSONFieldsForCreate_InvalidJSONReturnsError(t *testing.T) {
	paramOverride := "{invalid"
	channel := &Channel{
		ParamOverride: &paramOverride,
	}

	err := channel.NormalizeJSONFieldsForCreate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "参数覆盖")
}

func TestNormalizeJSONFieldsForCreate_CompactsValidJSON(t *testing.T) {
	modelMapping := "{\n  \"gpt-4o\": \" upstream-model \"\n}"
	paramOverride := "{\n  \"temperature\": 0\n}"
	headerOverride := "{\n  \"X-Test\": \"1\"\n}"

	channel := &Channel{
		ModelMapping:   &modelMapping,
		ParamOverride:  &paramOverride,
		HeaderOverride: &headerOverride,
		OtherSettings:  "{\n  \"vertex_key_type\": \"json\"\n}",
		OtherInfo:      "{\n  \"status_reason\": \"ok\"\n}",
	}

	require.NoError(t, channel.NormalizeJSONFieldsForCreate())
	require.Equal(t, "{\"gpt-4o\":\" upstream-model \"}", *channel.ModelMapping)
	require.Equal(t, "{\"temperature\":0}", *channel.ParamOverride)
	require.Equal(t, "{\"X-Test\":\"1\"}", *channel.HeaderOverride)
	require.Equal(t, "{\"vertex_key_type\":\"json\"}", channel.OtherSettings)
	require.Equal(t, "{\"status_reason\":\"ok\"}", channel.OtherInfo)
}

func TestHandlerMultiKeyUpdate_ReenableAutoDisabledChannel(t *testing.T) {
	channel := &Channel{
		Status: common.ChannelStatusAutoDisabled,
		Key:    "key-1\nkey-2",
		ChannelInfo: ChannelInfo{
			IsMultiKey:         true,
			MultiKeySize:       2,
			MultiKeyStatusList: map[int]int{0: common.ChannelStatusAutoDisabled, 1: common.ChannelStatusAutoDisabled},
			MultiKeyDisabledReason: map[int]string{
				0: "quota exceeded",
				1: "quota exceeded",
			},
			MultiKeyDisabledTime: map[int]int64{
				0: 100,
				1: 200,
			},
		},
	}

	handlerMultiKeyUpdate(channel, "key-1", common.ChannelStatusEnabled, "")

	require.Equal(t, common.ChannelStatusEnabled, channel.Status)
	require.Equal(t, map[int]int{1: common.ChannelStatusAutoDisabled}, channel.ChannelInfo.MultiKeyStatusList)
	require.NotContains(t, channel.ChannelInfo.MultiKeyDisabledReason, 0)
	require.NotContains(t, channel.ChannelInfo.MultiKeyDisabledTime, 0)

	info := channel.GetOtherInfo()
	require.Equal(t, "", info["status_reason"])
	require.NotNil(t, info["status_time"])
}
