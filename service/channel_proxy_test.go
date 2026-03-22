package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

func TestResolveChannelProxyValuePrefersChannelProxy(t *testing.T) {
	setting := system_setting.GetChannelProxySetting()
	original := setting.TypeProxies
	setting.TypeProxies = map[string]string{"57": "http://type-proxy:7890"}
	t.Cleanup(func() {
		setting.TypeProxies = original
	})

	got := ResolveChannelProxyValue(57, "http://channel-proxy:7890")
	if got != "http://channel-proxy:7890" {
		t.Fatalf("expected channel proxy, got %q", got)
	}
}

func TestResolveChannelProxyValueFallsBackToTypeProxy(t *testing.T) {
	setting := system_setting.GetChannelProxySetting()
	original := setting.TypeProxies
	setting.TypeProxies = map[string]string{"57": "http://type-proxy:7890"}
	t.Cleanup(func() {
		setting.TypeProxies = original
	})

	got := ResolveChannelProxyValue(57, "")
	if got != "http://type-proxy:7890" {
		t.Fatalf("expected type proxy, got %q", got)
	}
}

func TestResolveChannelProxyReturnsEmptyWithoutOverrides(t *testing.T) {
	setting := system_setting.GetChannelProxySetting()
	original := setting.TypeProxies
	setting.TypeProxies = map[string]string{}
	t.Cleanup(func() {
		setting.TypeProxies = original
	})

	got := ResolveChannelProxyValue(57, "")
	if got != "" {
		t.Fatalf("expected empty proxy, got %q", got)
	}
}

func TestResolveChannelProxyUsesChannelSettings(t *testing.T) {
	setting := system_setting.GetChannelProxySetting()
	original := setting.TypeProxies
	setting.TypeProxies = map[string]string{"57": "http://type-proxy:7890"}
	t.Cleanup(func() {
		setting.TypeProxies = original
	})

	channelSetting := dto.ChannelSettings{Proxy: "http://channel-proxy:7890"}
	raw, err := common.Marshal(channelSetting)
	if err != nil {
		t.Fatalf("marshal settings failed: %v", err)
	}

	channel := &model.Channel{
		Type:    57,
		Setting: common.GetPointer(string(raw)),
	}

	got := ResolveChannelProxy(channel)
	if got != "http://channel-proxy:7890" {
		t.Fatalf("expected channel proxy, got %q", got)
	}
}
