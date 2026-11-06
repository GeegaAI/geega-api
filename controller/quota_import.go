package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// QuotaImportTemplate 下载标准额度申请模版（运行时由 excelize 生成）
func QuotaImportTemplate(c *gin.Context) {
	data, err := service.BuildQuotaImportTemplate()
	if err != nil {
		common.ApiErrorMsg(c, "模版生成失败")
		return
	}
	c.Header("Content-Disposition", `attachment; filename="quota-import-template.xlsx"; filename*=UTF-8''%E9%A2%9D%E5%BA%A6%E7%94%B3%E8%AF%B7%E6%A8%A1%E7%89%88.xlsx`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// parseUploadExcel 读取 multipart 中的 Excel 并解析；失败时直接写错误响应
func parseUploadExcel(c *gin.Context) ([]model.QuotaImportPreviewRow, string, error) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		common.ApiError(c, fmt.Errorf("请上传 Excel 文件"))
		return nil, "", err
	}
	f, err := fileHeader.Open()
	if err != nil {
		common.ApiError(c, err)
		return nil, "", err
	}
	defer f.Close()

	rows, err := service.ParseQuotaImportExcel(f)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return nil, "", err
	}
	return rows, fileHeader.Filename, nil
}

// QuotaImportPreview 上传 Excel 并返回解析+用户匹配预览（不落库、不建任务）
func QuotaImportPreview(c *gin.Context) {
	rows, filename, err := parseUploadExcel(c)
	if err != nil {
		return
	}
	if err := service.FillQuotaPreviewUserState(rows); err != nil {
		common.ApiError(c, err)
		return
	}
	summary := buildPreviewSummary(rows, rows)
	summary["filename"] = filename
	summary["content_hash"] = service.ComputeQuotaImportHash(rows)
	common.ApiSuccess(c, summary)
}

// QuotaImportUpload 上传 Excel 创建待执行任务（不执行）。
// 可选 form 字段 scheduled_time：计划执行时间（Unix 秒），0/缺省=仅创建不定时
func QuotaImportUpload(c *gin.Context) {
	rows, filename, err := parseUploadExcel(c)
	if err != nil {
		return
	}
	if len(rows) == 0 {
		common.ApiErrorMsg(c, "Excel 中没有有效数据行")
		return
	}
	if err := service.FillQuotaPreviewUserState(rows); err != nil {
		common.ApiError(c, err)
		return
	}
	expectedHash := c.PostForm("content_hash")
	if expectedHash == "" || expectedHash != service.ComputeQuotaImportHash(rows) {
		common.ApiErrorMsg(c, "文件内容与预览不一致，请重新预览后再上传")
		return
	}

	now := common.GetTimestamp()
	record := &model.QuotaImportRecord{
		Type:         model.QuotaImportTypeUpload,
		Filename:     filename,
		OperatorId:   c.GetInt("id"),
		OperatorName: c.GetString("username"),
		Status:       model.QuotaImportStatusPending,
		TotalRows:    len(rows),
		ScheduledTime: func() int64 {
			st, _ := strconv.ParseInt(c.PostForm("scheduled_time"), 10, 64)
			return st
		}(),
		CreatedTime: now,
	}
	details := make([]model.QuotaImportDetail, 0, len(rows))
	for _, row := range rows {
		result := model.QuotaImportRowSuccess
		errMsg := ""
		if !row.Matched {
			result = model.QuotaImportRowUnmatched
			errMsg = "user not found"
		} else if row.NonMonthly {
			result = model.QuotaImportRowSkipped
			errMsg = "非月度清零策略，Excel 不处理"
		}
		details = append(details, model.QuotaImportDetail{
			Username:     row.Username,
			AmountYuan:   row.AmountYuan,
			Quota:        row.Quota,
			UserId:       row.UserId,
			Result:       result,
			ErrorMessage: errMsg,
			CreatedTime:  now,
		})
	}
	if err := model.CreateQuotaImportRecord(record, details); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, 0, "quota_import.upload", map[string]interface{}{
		"record_id":  record.Id,
		"filename":   record.Filename,
		"total_rows": record.TotalRows,
		"scheduled":  record.ScheduledTime,
	})
	common.ApiSuccess(c, record)
}

// QuotaImportExecute 立即执行一个待执行任务
func QuotaImportExecute(c *gin.Context) {
	recordId, err := strconv.Atoi(c.Param("id"))
	if err != nil || recordId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	record, err := service.ExecuteQuotaImportTask(recordId)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	recordManageAuditFor(c, 0, "quota_import.execute", map[string]interface{}{
		"record_id":    record.Id,
		"filename":     record.Filename,
		"total_rows":   record.TotalRows,
		"success_rows": record.SuccessRows,
		"reset_count":  record.ResetCount,
		"trigger":      "manual",
	})
	common.ApiSuccess(c, record)
}

// QuotaImportSchedule 设置/修改待执行任务的计划执行时间
func QuotaImportSchedule(c *gin.Context) {
	type scheduleRequest struct {
		ScheduledTime int64 `json:"scheduled_time"`
	}
	var req scheduleRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	recordId, err := strconv.Atoi(c.Param("id"))
	if err != nil || recordId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if req.ScheduledTime < 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.UpdateQuotaImportSchedule(recordId, req.ScheduledTime); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	recordManageAuditFor(c, 0, "quota_import.schedule", map[string]interface{}{
		"record_id":      recordId,
		"scheduled_time": req.ScheduledTime,
	})
	common.ApiSuccess(c, nil)
}

// QuotaImportDelete 删除待执行任务
func QuotaImportDelete(c *gin.Context) {
	recordId, err := strconv.Atoi(c.Param("id"))
	if err != nil || recordId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.DeleteQuotaImportRecord(recordId); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	recordManageAuditFor(c, 0, "quota_import.delete", map[string]interface{}{
		"record_id": recordId,
	})
	common.ApiSuccess(c, nil)
}

func buildPreviewSummary(all, rows []model.QuotaImportPreviewRow) gin.H {
	matched := 0
	unmatched := 0
	issue := 0
	var totalAmount float64
	for _, row := range rows {
		if row.Matched {
			matched++
		} else {
			unmatched++
		}
		if row.Issue != "" {
			issue++
		}
		totalAmount += row.AmountYuan
	}
	return gin.H{
		"rows":         all,
		"total":        len(rows),
		"matched":      matched,
		"unmatched":    unmatched,
		"issue":        issue,
		"total_amount": totalAmount,
	}
}

// GetQuotaImportRecords 分页返回任务/执行记录列表
func GetQuotaImportRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	records, total, err := model.GetQuotaImportRecords(page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"records":   records,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetQuotaImportRecordDetail 返回某次导入的明细
func GetQuotaImportRecordDetail(c *gin.Context) {
	recordId, err := strconv.Atoi(c.Param("id"))
	if err != nil || recordId <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	details, err := model.GetQuotaImportDetails(recordId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, details)
}
