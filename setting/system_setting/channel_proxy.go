package system_setting

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

type ChannelProxySetting struct {
	TypeProxies map[string]string `json:"type_proxies"`
}

var defaultChannelProxySetting = ChannelProxySetting{
	TypeProxies: map[string]string{},
}

func init() {
	config.GlobalConfig.Register("channel_proxy_setting", &defaultChannelProxySetting)
}

func GetChannelProxySetting() *ChannelProxySetting {
	if defaultChannelProxySetting.TypeProxies == nil {
		defaultChannelProxySetting.TypeProxies = map[string]string{}
	}
	return &defaultChannelProxySetting
}

func (s *ChannelProxySetting) GetTypeProxy(channelType int) string {
	if s == nil || len(s.TypeProxies) == 0 {
		return ""
	}
	return strings.TrimSpace(s.TypeProxies[strconv.Itoa(channelType)])
}
