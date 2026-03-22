package service

import (
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

func ResolveChannelProxyValue(channelType int, channelProxy string) string {
	channelProxy = strings.TrimSpace(channelProxy)
	if channelProxy != "" {
		return channelProxy
	}
	return system_setting.GetChannelProxySetting().GetTypeProxy(channelType)
}

func ResolveEffectiveChannelSettings(channelType int, channelSetting dto.ChannelSettings) dto.ChannelSettings {
	channelSetting.Proxy = ResolveChannelProxyValue(channelType, channelSetting.Proxy)
	return channelSetting
}

func ResolveChannelProxy(channel *model.Channel) string {
	if channel == nil {
		return ""
	}
	return ResolveChannelProxyValue(channel.Type, channel.GetSetting().Proxy)
}
