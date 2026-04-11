package middleware

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestIsTransientResponsesModelName(t *testing.T) {
	require.True(t, isTransientResponsesModelName("m_6b473fa8"))
	require.True(t, isTransientResponsesModelName("m_abc123XYZ"))
	require.False(t, isTransientResponsesModelName("gpt-5.4"))
	require.False(t, isTransientResponsesModelName("m_"))
	require.False(t, isTransientResponsesModelName("m_bad/value"))
}

func TestCanReuseAffinityChannelForTransientModel(t *testing.T) {
	channel := &model.Channel{
		Group: "default,vip",
	}

	require.True(t, canReuseAffinityChannelForTransientModel(channel, "default", "m_6b473fa8"))
	require.False(t, canReuseAffinityChannelForTransientModel(channel, "default", "gpt-5.4"))
	require.False(t, canReuseAffinityChannelForTransientModel(channel, "auto", "m_6b473fa8"))
	require.False(t, canReuseAffinityChannelForTransientModel(channel, "svip", "m_6b473fa8"))
}
