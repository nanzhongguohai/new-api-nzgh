package service

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

type RetryParam struct {
	Ctx          *gin.Context
	TokenGroup   string
	ModelName    string
	Retry        *int
	resetNextTry bool
}

func (p *RetryParam) GetRetry() int {
	if p.Retry == nil {
		return 0
	}
	return *p.Retry
}

func (p *RetryParam) SetRetry(retry int) {
	p.Retry = &retry
}

func (p *RetryParam) IncreaseRetry() {
	if p.resetNextTry {
		p.resetNextTry = false
		return
	}
	if p.Retry == nil {
		p.Retry = new(int)
	}
	*p.Retry++
}

func (p *RetryParam) ResetRetryNextTry() {
	p.resetNextTry = true
}

func getExcludedChannelIDs(ctx *gin.Context) map[int]struct{} {
	excluded := make(map[int]struct{})
	if ctx == nil {
		return excluded
	}
	for _, rawID := range ctx.GetStringSlice("use_channel") {
		channelID, err := strconv.Atoi(rawID)
		if err != nil || channelID <= 0 {
			continue
		}
		excluded[channelID] = struct{}{}
	}
	return excluded
}

// CacheGetRandomSatisfiedChannel selects the highest-priority remaining channel for this request.
// 同一请求内会优先用尽同优先级的其它渠道，再下沉到更低优先级。
func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	var channel *model.Channel
	var err error
	selectGroup := param.TokenGroup
	userGroup := common.GetContextKeyString(param.Ctx, constant.ContextKeyUserGroup)
	excludedChannelIDs := getExcludedChannelIDs(param.Ctx)

	if param.TokenGroup == "auto" {
		if len(setting.GetAutoGroups()) == 0 {
			return nil, selectGroup, errors.New("auto groups is not enabled")
		}
		autoGroups := GetUserAutoGroup(userGroup)

		// startGroupIndex: the group index to start searching from
		// startGroupIndex: 开始搜索的分组索引
		startGroupIndex := 0
		crossGroupRetry := common.GetContextKeyBool(param.Ctx, constant.ContextKeyTokenCrossGroupRetry)

		if lastGroupIndex, exists := common.GetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex); exists {
			if idx, ok := lastGroupIndex.(int); ok {
				startGroupIndex = idx
			}
		}

		for i := startGroupIndex; i < len(autoGroups); i++ {
			autoGroup := autoGroups[i]
			logger.LogDebug(param.Ctx, "Auto selecting group: %s, retry: %d", autoGroup, param.GetRetry())

			channel, _ = model.GetRandomSatisfiedChannelWithExclude(autoGroup, param.ModelName, excludedChannelIDs)
			if channel == nil {
				logger.LogDebug(param.Ctx, "No remaining channel in group %s for model %s, trying next group", autoGroup, param.ModelName)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				continue
			}
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, autoGroup)
			selectGroup = autoGroup
			logger.LogDebug(param.Ctx, "Auto selected group: %s", autoGroup)
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i)
			if crossGroupRetry {
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupRetryIndex, param.GetRetry())
			}
			break
		}
	} else {
		channel, err = model.GetRandomSatisfiedChannelWithExclude(param.TokenGroup, param.ModelName, excludedChannelIDs)
		if err != nil {
			return nil, param.TokenGroup, err
		}
	}
	return channel, selectGroup, nil
}
