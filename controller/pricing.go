package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// userVisibleModelRatios 返回该用户在当前定价列表内、计费侧实际会生效的专属倍率：
// 普通按量模型替换模型倍率；tiered_expr 模型作为结算系数
// （表达式计费 × 分组倍率 × 专属系数）；按次计费模型作为价格系数
// （单价 × 分组倍率 × 专属系数）。三种模式均真实生效，展示口径与计费口径一致。
func userVisibleModelRatios(userId int, pricing []model.Pricing) map[string]float64 {
	visible := make(map[string]float64)
	userRatios := model.GetUserModelRatios(userId)
	if len(userRatios) == 0 {
		return visible
	}
	for _, item := range pricing {
		if ratio, ok := userRatios[item.ModelName]; ok {
			visible[item.ModelName] = ratio
		}
	}
	return visible
}

func filterPricingByUsableGroups(pricing []model.Pricing, usableGroup map[string]string) []model.Pricing {
	if len(pricing) == 0 {
		return pricing
	}
	if len(usableGroup) == 0 {
		return []model.Pricing{}
	}

	filtered := make([]model.Pricing, 0, len(pricing))
	for _, item := range pricing {
		if common.StringsContains(item.EnableGroup, "all") {
			filtered = append(filtered, item)
			continue
		}
		for _, group := range item.EnableGroup {
			if _, ok := usableGroup[group]; ok {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

// stripPricingToGroup 将定价列表收窄到单个分组：模型 enable_groups 只保留该分组，
// 且仅保留该分组可用的模型。
func stripPricingToGroup(pricing []model.Pricing, group string) []model.Pricing {
	filtered := make([]model.Pricing, 0, len(pricing))
	for _, item := range pricing {
		if !common.StringsContains(item.EnableGroup, group) {
			continue
		}
		item.EnableGroup = []string{group}
		filtered = append(filtered, item)
	}
	return filtered
}

func GetPricing(c *gin.Context) {
	pricing := model.GetPricing()
	userId, exists := c.Get("id")

	// 管理员看全量；非管理员与匿名访客按分组收窄模型列表与分组元数据
	if exists {
		user, err := model.GetUserCache(userId.(int))
		if err == nil && user.Role >= common.RoleAdminUser {
			c.JSON(200, gin.H{
				"success":            true,
				"data":               pricing,
				"vendors":            model.GetVendors(),
				"group_ratio":        ratio_setting.GetGroupRatioCopy(),
				"usable_group":       service.GetUserUsableGroups(user.Group),
				"supported_endpoint": model.GetSupportedEndpointMap(),
				"auto_groups":        service.GetUserAutoGroup(user.Group),
				"user_model_ratio":   userVisibleModelRatios(user.Id, pricing),
				"pricing_version":    "a42d372ccf0b5dd13ecf71203521f9d2",
			})
			return
		}
		// 非管理员：只展示自己分组下的模型与元数据
		if err == nil && user.Group != "" {
			group := user.Group
			pricing = stripPricingToGroup(pricing, group)
			c.JSON(200, gin.H{
				"success":            true,
				"data":               pricing,
				"vendors":            model.GetVendors(),
				"group_ratio":        map[string]float64{group: service.GetUserGroupRatio(group, group)},
				"usable_group":       map[string]string{group: setting.GetUsableGroupDescription(group)},
				"supported_endpoint": model.GetSupportedEndpointMap(),
				"auto_groups":        []string{},
				"user_model_ratio":   userVisibleModelRatios(user.Id, pricing),
				"pricing_version":    "a42d372ccf0b5dd13ecf71203521f9d2",
			})
			return
		}
	}

	// 匿名访客：按默认分组展示
	defaultGroup := "default"
	pricing = stripPricingToGroup(pricing, defaultGroup)
	usableGroup := map[string]string{defaultGroup: setting.GetUsableGroupDescription(defaultGroup)}
	groupRatio := map[string]float64{defaultGroup: service.GetUserGroupRatio(defaultGroup, defaultGroup)}
	c.JSON(200, gin.H{
		"success":            true,
		"data":               pricing,
		"vendors":            model.GetVendors(),
		"group_ratio":        groupRatio,
		"usable_group":       usableGroup,
		"supported_endpoint": model.GetSupportedEndpointMap(),
		"auto_groups":        []string{},
		"user_model_ratio":   map[string]float64{},
		"pricing_version":    "a42d372ccf0b5dd13ecf71203521f9d2",
	})
}

func ResetModelRatio(c *gin.Context) {
	defaultStr := ratio_setting.DefaultModelRatio2JSONString()
	err := model.UpdateOption("ModelRatio", defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = ratio_setting.UpdateModelRatioByJSONString(defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "重置模型倍率成功",
	})
}
