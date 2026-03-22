package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func int64Ptr(v int64) *int64 {
	return &v
}

func uintPtr(v uint) *uint {
	return &v
}

func TestGetRandomSatisfiedChannelWithExcludeSkipsExcludedChannel(t *testing.T) {
	oldMemoryCache := common.MemoryCacheEnabled
	oldGroupMap := group2model2channels
	oldChannels := channelsIDM
	t.Cleanup(func() {
		common.MemoryCacheEnabled = oldMemoryCache
		group2model2channels = oldGroupMap
		channelsIDM = oldChannels
	})

	common.MemoryCacheEnabled = true
	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-5.4": {1, 2},
		},
	}
	channelsIDM = map[int]*Channel{
		1: {Id: 1, Priority: int64Ptr(0), Weight: uintPtr(10)},
		2: {Id: 2, Priority: int64Ptr(0), Weight: uintPtr(10)},
	}

	channel, err := GetRandomSatisfiedChannelWithExclude("default", "gpt-5.4", 0, []int{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if channel == nil || channel.Id != 2 {
		t.Fatalf("expected channel 2, got %#v", channel)
	}
}

func TestGetRandomSatisfiedChannelWithExcludeFallsBackToRemainingPriority(t *testing.T) {
	oldMemoryCache := common.MemoryCacheEnabled
	oldGroupMap := group2model2channels
	oldChannels := channelsIDM
	t.Cleanup(func() {
		common.MemoryCacheEnabled = oldMemoryCache
		group2model2channels = oldGroupMap
		channelsIDM = oldChannels
	})

	common.MemoryCacheEnabled = true
	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-5.4": {1, 2},
		},
	}
	channelsIDM = map[int]*Channel{
		1: {Id: 1, Priority: int64Ptr(10), Weight: uintPtr(10)},
		2: {Id: 2, Priority: int64Ptr(5), Weight: uintPtr(10)},
	}

	channel, err := GetRandomSatisfiedChannelWithExclude("default", "gpt-5.4", 0, []int{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if channel == nil || channel.Id != 2 {
		t.Fatalf("expected channel 2 after excluding higher priority channel, got %#v", channel)
	}
}

func TestGetRandomSatisfiedChannelWithExcludeReturnsNilWhenAllExcluded(t *testing.T) {
	oldMemoryCache := common.MemoryCacheEnabled
	oldGroupMap := group2model2channels
	oldChannels := channelsIDM
	t.Cleanup(func() {
		common.MemoryCacheEnabled = oldMemoryCache
		group2model2channels = oldGroupMap
		channelsIDM = oldChannels
	})

	common.MemoryCacheEnabled = true
	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-5.4": {1},
		},
	}
	channelsIDM = map[int]*Channel{
		1: {Id: 1, Priority: int64Ptr(0), Weight: uintPtr(10)},
	}

	channel, err := GetRandomSatisfiedChannelWithExclude("default", "gpt-5.4", 0, []int{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if channel != nil {
		t.Fatalf("expected no channel after excluding all candidates, got %#v", channel)
	}
}
