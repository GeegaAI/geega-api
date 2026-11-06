package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

const availabilityCompensationMaxRangeSeconds = 62 * 24 * 60 * 60

var availabilityCompensationLocation = time.FixedZone("CST", 8*60*60)

func parseAvailabilityCompensationRange(c *gin.Context) (int64, int64, bool) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if startTimestamp <= 0 || endTimestamp <= 0 {
		common.ApiError(c, errors.New("start_timestamp 和 end_timestamp 不能为空"))
		return 0, 0, false
	}
	if startTimestamp >= endTimestamp {
		common.ApiError(c, errors.New("start_timestamp 必须小于 end_timestamp"))
		return 0, 0, false
	}
	if endTimestamp-startTimestamp > availabilityCompensationMaxRangeSeconds {
		common.ApiError(c, errors.New("统计周期不能超过 62 天"))
		return 0, 0, false
	}
	return startTimestamp, endTimestamp, true
}

// GetAvailabilityCompensation 供应商可用性赔偿测算（只读聚合）。
func GetAvailabilityCompensation(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseAvailabilityCompensationRange(c)
	if !ok {
		return
	}
	result, err := model.GetAvailabilityCompensation(startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

// ExportAvailabilityCompensation 下载与页面同一口径的 Excel 报告。
func ExportAvailabilityCompensation(c *gin.Context) {
	startTimestamp, endTimestamp, ok := parseAvailabilityCompensationRange(c)
	if !ok {
		return
	}
	result, err := model.GetAvailabilityCompensation(startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data, err := service.BuildAvailabilityCompensationWorkbook(result)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	startDate := time.Unix(startTimestamp, 0).In(availabilityCompensationLocation).Format("20060102")
	endDate := time.Unix(endTimestamp, 0).In(availabilityCompensationLocation).Add(-time.Second).Format("20060102")
	chineseName := fmt.Sprintf("供应商可用性赔偿测算-%s-%s.xlsx", startDate, endDate)
	asciiName := fmt.Sprintf("supplier-availability-compensation-%s-%s.xlsx", startDate, endDate)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, asciiName, url.PathEscape(chineseName)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
