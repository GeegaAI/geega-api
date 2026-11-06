package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"gorm.io/gorm"
)

const (
	QuotaImportTypeUpload    = "upload"     // 上传 Excel 建任务（按名单赋值）
	QuotaImportTypeAutoReset = "auto_reset" // 定时任务月度清零留痕

	QuotaImportStatusPending   = "pending"   // 待执行
	QuotaImportStatusExecuting = "executing" // 执行中（瞬态，DB 抢占标记）
	QuotaImportStatusSuccess   = "success"   // 全部成功
	QuotaImportStatusPartial   = "partial"   // 部分行失败/未匹配
	QuotaImportStatusFailed    = "failed"    // 执行失败

	QuotaImportRowSuccess   = "success"
	QuotaImportRowUnmatched = "unmatched"
	QuotaImportRowSkipped   = "skipped" // 非月度清零策略用户，Excel 不处理
	QuotaImportRowFailed    = "failed"
)

type QuotaImportRecord struct {
	Id            int    `json:"id"`
	Type          string `json:"type" gorm:"type:varchar(16);index"`
	Filename      string `json:"filename" gorm:"type:varchar(255)"`
	OperatorId    int    `json:"operator_id" gorm:"index"`
	OperatorName  string `json:"operator_name" gorm:"type:varchar(64)"`
	Status        string `json:"status" gorm:"type:varchar(16);index"`
	TotalRows     int    `json:"total_rows"`
	SuccessRows   int    `json:"success_rows"`
	UnmatchedRows int    `json:"unmatched_rows"`
	SkippedRows   int    `json:"skipped_rows"`
	FailedRows    int    `json:"failed_rows"`
	ResetCount    int    `json:"reset_count"`
	ScheduledTime int64  `json:"scheduled_time"` // 计划执行时间，0=未定时
	Remark        string `json:"remark" gorm:"type:varchar(512)"`
	CreatedTime   int64  `json:"created_time"`
	ExecutedTime  int64  `json:"executed_time"`
}

func (QuotaImportRecord) TableName() string { return "quota_import_records" }

type QuotaImportDetail struct {
	Id           int     `json:"id"`
	RecordId     int     `json:"record_id" gorm:"index"`
	Username     string  `json:"username" gorm:"type:varchar(64)"`
	AmountYuan   float64 `json:"amount_yuan"`
	Quota        int     `json:"quota"`
	UserId       int     `json:"user_id"`
	Result       string  `json:"result" gorm:"type:varchar(16)"`
	ErrorMessage string  `json:"error_message" gorm:"type:varchar(512)"`
	CreatedTime  int64   `json:"created_time"`
}

func (QuotaImportDetail) TableName() string { return "quota_import_details" }

// 上传解析预览行（不落库）
type QuotaImportPreviewRow struct {
	RowNumber    int     `json:"row_number"`
	Username     string  `json:"username"`
	AmountYuan   float64 `json:"amount_yuan"`
	Quota        int     `json:"quota"`
	UserId       int     `json:"user_id"`
	Matched      bool    `json:"matched"`
	CurrentQuota int     `json:"current_quota"`
	NonMonthly   bool    `json:"non_monthly"` // 非月度清零策略：Excel 不处理
	Duplicate    bool    `json:"duplicate"`
	Issue        string  `json:"issue"`
}

// quotaMonthlyResetDoneKey 单独保存"最近一次已执行月度清零"的月份。
// 留痕记录可被运营删除，标记独立保存可避免删记录后当月被重复清零
const quotaMonthlyResetDoneKey = "QuotaMonthlyResetDoneMonth"

// MarkMonthlyResetDone 记录月度清零已完成（月份格式 2006-01）。
// 只落库不写配置缓存：该键仅供本流程内部判断，避免污染全局配置
func MarkMonthlyResetDone(month string) error {
	option := Option{Key: quotaMonthlyResetDoneKey}
	if err := DB.FirstOrCreate(&option, Option{Key: quotaMonthlyResetDoneKey}).Error; err != nil {
		return err
	}
	option.Value = month
	return DB.Save(&option).Error
}

func markedMonthlyResetMonth() string {
	var opt Option
	if err := DB.Select("value").Where(&Option{Key: quotaMonthlyResetDoneKey}).First(&opt).Error; err != nil {
		return ""
	}
	return opt.Value
}

// HasResetRecordForMonth 判断指定月份是否已完成月度清零，用于定时任务防重复执行。
// 以独立标记为准；历史数据没有标记时退回查留痕并补写标记。周期清零 remark 为 "cycle:日期"，不会冲突
func HasResetRecordForMonth(month string) (bool, error) {
	if markedMonthlyResetMonth() == month {
		return true, nil
	}
	var count int64
	err := DB.Model(&QuotaImportRecord{}).
		Where("type = ? AND remark = ?", QuotaImportTypeAutoReset, month).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil
	}
	if err := MarkMonthlyResetDone(month); err != nil {
		return false, err
	}
	return true, nil
}

func CreateAutoResetRecord(resetCount int64, month string, executedTime int64) (*QuotaImportRecord, error) {
	record := QuotaImportRecord{
		Type:         QuotaImportTypeAutoReset,
		Filename:     "-",
		Status:       QuotaImportStatusSuccess,
		ResetCount:   int(resetCount),
		Remark:       month,
		CreatedTime:  executedTime,
		ExecutedTime: executedTime,
	}
	if err := DB.Create(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func CreateQuotaImportRecord(record *QuotaImportRecord, details []QuotaImportDetail) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		if len(details) > 0 {
			for i := range details {
				details[i].RecordId = record.Id
			}
			return tx.CreateInBatches(details, 500).Error
		}
		return nil
	})
}

// GetQuotaImportRecord 按 id 加载记录
func GetQuotaImportRecord(id int) (*QuotaImportRecord, error) {
	var record QuotaImportRecord
	if err := DB.First(&record, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// GetDuePendingQuotaImports 返回到达计划执行时间的待执行任务（升序）
func GetDuePendingQuotaImports(now int64) ([]QuotaImportRecord, error) {
	var records []QuotaImportRecord
	err := DB.Where("type = ? AND status = ? AND scheduled_time > 0 AND scheduled_time <= ?",
		QuotaImportTypeUpload, QuotaImportStatusPending, now).
		Order("scheduled_time asc").Find(&records).Error
	return records, err
}

// ClaimQuotaImportExecution DB 级抢占：把任务置为 executing，防止多节点/手动与定时并发重复执行
func ClaimQuotaImportExecution(id int) (bool, error) {
	result := DB.Model(&QuotaImportRecord{}).
		Where("id = ? AND type = ? AND status IN ?", id, QuotaImportTypeUpload,
			[]string{QuotaImportStatusPending, QuotaImportStatusPartial, QuotaImportStatusFailed}).
		Update("status", QuotaImportStatusExecuting)
	return result.RowsAffected == 1, result.Error
}

// UpdateQuotaImportSchedule 设置/清除任务的计划执行时间（仅待执行任务）
func UpdateQuotaImportSchedule(id int, scheduledTime int64) error {
	result := DB.Model(&QuotaImportRecord{}).
		Where("id = ? AND type = ? AND status = ?", id, QuotaImportTypeUpload, QuotaImportStatusPending).
		Update("scheduled_time", scheduledTime)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("任务不存在或不是待执行状态")
	}
	return nil
}

// DeleteQuotaImportRecord 删除一条执行记录及其明细（导入任务与自动清零留痕均可删）。
// 删除当月月度清零留痕前先固化独立标记，避免系统误判当月未清零
func DeleteQuotaImportRecord(id int) error {
	var record QuotaImportRecord
	if err := DB.First(&record, id).Error; err != nil {
		return err
	}
	if record.Status == QuotaImportStatusExecuting {
		return errors.New("记录正在执行，请稍后再删除")
	}
	if record.Type == QuotaImportTypeAutoReset && record.Remark == time.Now().Format("2006-01") {
		if err := MarkMonthlyResetDone(record.Remark); err != nil {
			return err
		}
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("record_id = ?", id).Delete(&QuotaImportDetail{}).Error; err != nil {
			return err
		}
		return tx.Delete(&QuotaImportRecord{}, id).Error
	})
}

// UserMapByUsername 返回 username 小写 → user 映射，供导入预览匹配
func UserMapByUsername() (map[string]*User, error) {
	return userMapByUsername()
}

// userMapByUsername 返回 username 小写 → user 映射（含当前额度与清零策略），用于 Excel 行匹配
func userMapByUsername() (map[string]*User, error) {
	var users []User
	if err := DB.Select("id, username, quota, role, status, quota_reset_mode, quota_next_reset_at").Find(&users).Error; err != nil {
		return nil, err
	}
	m := make(map[string]*User, len(users))
	for i := range users {
		m[strings.ToLower(users[i].Username)] = &users[i]
	}
	return m, nil
}

// executeQuotaRows 执行"按名单赋值"核心逻辑：只把文件额度写给生效策略为 monthly 的用户，
// 不做任何清零（清零由月度/周期清零任务独立处理）。
// 判定只有两类：用户是否存在、生效策略是否为 monthly；其余行跳过并记录原因
func executeQuotaRows(rows []QuotaImportPreviewRow, record *QuotaImportRecord) []QuotaImportDetail {
	userMap, err := userMapByUsername()
	record.FailedRows = 0
	record.UnmatchedRows = 0
	record.SkippedRows = 0
	record.SuccessRows = 0
	if err != nil {
		record.Status = QuotaImportStatusFailed
		record.Remark = err.Error()
		return nil
	}

	now := common.GetTimestamp()
	// 1. 逐行赋值：同账号重复取最后一次出现的值（rows 已按最终生效去重后传入）
	details := make([]QuotaImportDetail, 0, len(rows))
	invalidated := map[int]bool{}
	for _, row := range rows {
		detail := QuotaImportDetail{
			Username:    row.Username,
			AmountYuan:  row.AmountYuan,
			Quota:       row.Quota,
			CreatedTime: now,
		}
		user, ok := userMap[strings.ToLower(row.Username)]
		if !ok || user.Id == 0 {
			detail.Result = QuotaImportRowUnmatched
			detail.ErrorMessage = "user not found"
			record.UnmatchedRows++
		} else {
			detail.UserId = user.Id
			switch {
			case user.EffectiveQuotaResetMode() != operation_setting.QuotaResetModeMonthly:
				// Excel 只服务月度清零用户；周期/不清零用户的额度请在用户编辑弹窗分配
				detail.Result = QuotaImportRowSkipped
				detail.ErrorMessage = "非月度清零策略，Excel 不处理"
				record.SkippedRows++
			default:
				if err := DB.Model(&User{}).Where("id = ?", user.Id).Update("quota", row.Quota).Error; err != nil {
					detail.Result = QuotaImportRowFailed
					detail.ErrorMessage = err.Error()
					record.FailedRows++
				} else {
					detail.Result = QuotaImportRowSuccess
					record.SuccessRows++
					invalidated[user.Id] = true
				}
			}
		}
		details = append(details, detail)
	}

	// 2. 失效受影响用户缓存（赋值涉及的用户）
	for userId := range invalidated {
		if err := invalidateUserCache(userId); err != nil {
			common.SysLog(fmt.Sprintf("quota import: failed to invalidate cache for user %d: %s", userId, err.Error()))
		}
	}

	if record.FailedRows > 0 || record.UnmatchedRows > 0 || record.SkippedRows > 0 {
		record.Status = QuotaImportStatusPartial
	} else {
		record.Status = QuotaImportStatusSuccess
	}
	return details
}

// ExecuteQuotaImportRecord 执行一个已创建的导入任务：从落库明细取行，
// 执行时重新匹配用户（用户增删以后执行结果为准），执行后重写明细与记录
func ExecuteQuotaImportRecord(recordId int) (*QuotaImportRecord, error) {
	record, err := GetQuotaImportRecord(recordId)
	if err != nil {
		return nil, err
	}
	stored, err := GetQuotaImportDetails(recordId)
	if err != nil {
		return nil, err
	}
	rows := make([]QuotaImportPreviewRow, 0, len(stored))
	for _, d := range stored {
		rows = append(rows, QuotaImportPreviewRow{
			RowNumber:  d.Id,
			Username:   d.Username,
			AmountYuan: d.AmountYuan,
			Quota:      d.Quota,
		})
	}

	details := executeQuotaRows(rows, record)
	record.ExecutedTime = common.GetTimestamp()
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("record_id = ?", recordId).Delete(&QuotaImportDetail{}).Error; err != nil {
			return err
		}
		if len(details) > 0 {
			for i := range details {
				details[i].RecordId = recordId
			}
			if err := tx.CreateInBatches(details, 500).Error; err != nil {
				return err
			}
		}
		return tx.Model(&QuotaImportRecord{}).Where("id = ?", recordId).Updates(map[string]interface{}{
			"status":         record.Status,
			"total_rows":     record.TotalRows,
			"success_rows":   record.SuccessRows,
			"unmatched_rows": record.UnmatchedRows,
			"skipped_rows":   record.SkippedRows,
			"failed_rows":    record.FailedRows,
			"reset_count":    record.ResetCount,
			"remark":         record.Remark,
			"executed_time":  record.ExecutedTime,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return record, nil
}

// GetQuotaImportRecords 分页返回任务/执行记录列表
func GetQuotaImportRecords(page, pageSize int) ([]QuotaImportRecord, int64, error) {
	var records []QuotaImportRecord
	var total int64
	if err := DB.Model(&QuotaImportRecord{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := DB.Order("created_time desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error
	return records, total, err
}

// GetQuotaImportDetails 返回某次导入的明细
func GetQuotaImportDetails(recordId int) ([]QuotaImportDetail, error) {
	var details []QuotaImportDetail
	err := DB.Where("record_id = ?", recordId).Order("id asc").Find(&details).Error
	return details, err
}
