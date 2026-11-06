package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type updateUserModelRatiosRequest struct {
	Ratios map[string]float64 `json:"ratios"`
}

// GetUserModelRatiosByAdmin 查看指定用户的专属模型倍率（模型名→倍率）。
func GetUserModelRatiosByAdmin(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户 ID")
		return
	}
	ratios := model.GetUserModelRatios(userId)
	if ratios == nil {
		ratios = map[string]float64{}
	}
	common.ApiSuccess(c, ratios)
}

// UpdateUserModelRatiosByAdmin 全量替换指定用户的专属模型倍率；ratios 为空对象时清空。
// 覆盖值语义为替换全局模型倍率，分组倍率照常相乘。
func UpdateUserModelRatiosByAdmin(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户 ID")
		return
	}
	var req updateUserModelRatiosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "请求参数错误")
		return
	}
	if err := model.ReplaceUserModelRatios(userId, req.Ratios); err != nil {
		common.ApiError(c, err)
		return
	}
	ratios := model.GetUserModelRatios(userId)
	if ratios == nil {
		ratios = map[string]float64{}
	}
	common.ApiSuccess(c, ratios)
}
