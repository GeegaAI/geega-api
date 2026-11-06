package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const errorAnalysisMaxRangeSeconds = 31 * 24 * 60 * 60

func GetErrorAnalysis(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if endTimestamp == 0 {
		endTimestamp = time.Now().Unix()
	}
	if startTimestamp == 0 {
		startTimestamp = endTimestamp - 24*60*60
	}
	if startTimestamp >= endTimestamp {
		common.ApiError(c, errors.New("start_timestamp 必须小于 end_timestamp"))
		return
	}
	if endTimestamp-startTimestamp > errorAnalysisMaxRangeSeconds {
		common.ApiError(c, errors.New("时间范围不能超过 31 天"))
		return
	}

	var timezoneOffset *int
	if raw := strings.TrimSpace(c.Query("timezone_offset")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < -14*60 || value > 12*60 {
			common.ApiError(c, errors.New("timezone_offset 参数无效"))
			return
		}
		timezoneOffset = &value
	}

	channelId, _ := strconv.Atoi(c.Query("channel_id"))
	modelName := strings.TrimSpace(c.Query("model_name"))
	group := strings.TrimSpace(c.Query("group"))

	result, err := model.GetErrorAnalysis(startTimestamp, endTimestamp, channelId, modelName, group, timezoneOffset)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}
