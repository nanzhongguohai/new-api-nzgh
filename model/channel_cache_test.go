package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestGetRandomSatisfiedChannelWithExclude_ExhaustsSamePriorityBeforeFallback(t *testing.T) {
	originalCacheEnabled := common.MemoryCacheEnabled
	originalGroup2Model2Channels := group2model2channels
	originalChannelsIDM := channelsIDM
	defer func() {
		common.MemoryCacheEnabled = originalCacheEnabled
		group2model2channels = originalGroup2Model2Channels
		channelsIDM = originalChannelsIDM
	}()

	common.MemoryCacheEnabled = true

	highPriority := int64(10)
	lowPriority := int64(5)
	zeroWeight := uint(0)

	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-4o": {1, 2, 3},
		},
	}
	channelsIDM = map[int]*Channel{
		1: {Id: 1, Priority: &highPriority, Weight: &zeroWeight},
		2: {Id: 2, Priority: &highPriority, Weight: &zeroWeight},
		3: {Id: 3, Priority: &lowPriority, Weight: &zeroWeight},
	}

	first, err := GetRandomSatisfiedChannelWithExclude("default", "gpt-4o", nil)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.Contains(t, []int{1, 2}, first.Id)

	excluded := map[int]struct{}{first.Id: {}}
	second, err := GetRandomSatisfiedChannelWithExclude("default", "gpt-4o", excluded)
	require.NoError(t, err)
	require.NotNil(t, second)
	require.Contains(t, []int{1, 2}, second.Id)
	require.NotEqual(t, first.Id, second.Id)

	excluded[second.Id] = struct{}{}
	third, err := GetRandomSatisfiedChannelWithExclude("default", "gpt-4o", excluded)
	require.NoError(t, err)
	require.NotNil(t, third)
	require.Equal(t, 3, third.Id)
}

func TestGetHighestPrioritySatisfiedChannel_KeepsTopPriority(t *testing.T) {
	originalCacheEnabled := common.MemoryCacheEnabled
	originalGroup2Model2Channels := group2model2channels
	originalChannelsIDM := channelsIDM
	defer func() {
		common.MemoryCacheEnabled = originalCacheEnabled
		group2model2channels = originalGroup2Model2Channels
		channelsIDM = originalChannelsIDM
	}()

	common.MemoryCacheEnabled = true

	highPriority := int64(20)
	lowPriority := int64(10)
	zeroWeight := uint(0)

	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-5": {2, 1},
		},
	}
	channelsIDM = map[int]*Channel{
		1: {Id: 1, Priority: &lowPriority, Weight: &zeroWeight},
		2: {Id: 2, Priority: &highPriority, Weight: &zeroWeight},
	}

	channel, err := GetHighestPrioritySatisfiedChannel("default", "gpt-5")
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 2, channel.Id)
}
