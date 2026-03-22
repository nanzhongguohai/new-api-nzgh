package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func seedChannelForSelectionTest(t *testing.T, id int, group string, modelName string, priority int64) {
	t.Helper()

	weight := uint(10)
	channel := &model.Channel{
		Id:       id,
		Name:     group + "-" + modelName,
		Key:      "sk-test",
		Status:   common.ChannelStatusEnabled,
		Group:    group,
		Models:   modelName,
		Priority: &priority,
		Weight:   &weight,
	}
	require.NoError(t, model.DB.Create(channel).Error)

	ability := &model.Ability{
		Group:     group,
		Model:     modelName,
		ChannelId: id,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
	}
	require.NoError(t, model.DB.Create(ability).Error)
}

func prepareChannelSelectionTest(t *testing.T) {
	t.Helper()

	require.NoError(t, model.DB.AutoMigrate(&model.Channel{}, &model.Ability{}))
	model.DB.Exec("DELETE FROM abilities")
	model.DB.Exec("DELETE FROM channels")
	model.InitChannelCache()

	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM abilities")
		model.DB.Exec("DELETE FROM channels")
		model.InitChannelCache()
	})
}

func TestCacheGetRandomSatisfiedChannel_SwitchesChannelAfterFailure(t *testing.T) {
	prepareChannelSelectionTest(t)

	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
	})
	common.MemoryCacheEnabled = true

	seedChannelForSelectionTest(t, 101, "default", "gpt-5.4", 10)
	seedChannelForSelectionTest(t, 102, "default", "gpt-5.4", 5)
	model.InitChannelCache()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	retry := 0
	param := &RetryParam{
		Ctx:        ctx,
		TokenGroup: "default",
		ModelName:  "gpt-5.4",
		Retry:      &retry,
	}

	first, group, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.Equal(t, "default", group)
	require.NotNil(t, first)
	require.Equal(t, 101, first.Id)

	MarkChannelExcludedForRetry(ctx, first.Id)
	param.IncreaseRetry()

	second, group, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.Equal(t, "default", group)
	require.NotNil(t, second)
	require.Equal(t, 102, second.Id)
}

func TestCacheGetRandomSatisfiedChannel_AutoGroupSwitchesAfterFailure(t *testing.T) {
	prepareChannelSelectionTest(t)

	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	oldRetryTimes := common.RetryTimes
	oldAutoGroupsJSON := setting.AutoGroups2JsonString()
	oldUserGroupsJSON := setting.UserUsableGroups2JSONString()
	t.Cleanup(func() {
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
		common.RetryTimes = oldRetryTimes
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(oldAutoGroupsJSON))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(oldUserGroupsJSON))
	})

	common.MemoryCacheEnabled = true
	common.RetryTimes = 1
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"group-a":"A","group-b":"B"}`))
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["group-a","group-b"]`))

	seedChannelForSelectionTest(t, 201, "group-a", "gpt-5.4", 10)
	seedChannelForSelectionTest(t, 202, "group-b", "gpt-5.4", 10)
	model.InitChannelCache()

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	common.SetContextKey(ctx, constant.ContextKeyTokenCrossGroupRetry, true)

	retry := 0
	param := &RetryParam{
		Ctx:        ctx,
		TokenGroup: "auto",
		ModelName:  "gpt-5.4",
		Retry:      &retry,
	}

	first, group, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.Equal(t, "group-a", group)
	require.NotNil(t, first)
	require.Equal(t, 201, first.Id)

	MarkChannelExcludedForRetry(ctx, first.Id)
	param.IncreaseRetry()

	second, group, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.Equal(t, "group-b", group)
	require.NotNil(t, second)
	require.Equal(t, 202, second.Id)
}
