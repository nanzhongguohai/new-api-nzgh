package service

import (
	"slices"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

func GetRetryExcludedChannelIDs(ctx *gin.Context) []int {
	if ctx == nil {
		return nil
	}
	excluded, ok := common.GetContextKeyType[[]int](ctx, constant.ContextKeyRetryExcludedChannelIDs)
	if !ok {
		return nil
	}
	return slices.Clone(excluded)
}

func MarkChannelExcludedForRetry(ctx *gin.Context, channelID int) {
	if ctx == nil || channelID <= 0 {
		return
	}
	excluded := GetRetryExcludedChannelIDs(ctx)
	if slices.Contains(excluded, channelID) {
		return
	}
	excluded = append(excluded, channelID)
	common.SetContextKey(ctx, constant.ContextKeyRetryExcludedChannelIDs, excluded)
}
