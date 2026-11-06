package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	// 到期任务扫描与月度清零检查共用 1 分钟 tick：到期任务最多延迟 1 分钟执行
	quotaImportTickInterval = 1 * time.Minute
)

var (
	quotaImportOnce   sync.Once
	quotaImportLocked atomic.Bool
)

// QuotaImportLockTry 尝试获取执行锁（上传执行与定时清零互斥），返回释放函数
func QuotaImportLockTry() (func(), bool) {
	if !quotaImportLocked.CompareAndSwap(false, true) {
		return nil, false
	}
	return func() { quotaImportLocked.Store(false) }, true
}

func StartQuotaImportResetTask() {
	quotaImportOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("quota import reset task started: tick=%s", quotaImportTickInterval))
			ticker := time.NewTicker(quotaImportTickInterval)
			defer ticker.Stop()

			for range ticker.C {
				runQuotaResetOnce()
			}
		})
	})
}

// runQuotaResetOnce 每 tick 调度：到期上传任务 + 月度清零（monthly 群体）+ 周期清零（cycle 群体）。
// 月度清零以留痕月份为守卫（服务重启/多 tick 不重复执行）；周期清零依赖用户锚点天然幂等
func runQuotaResetOnce() {
	runDueQuotaImports()

	release, ok := QuotaImportLockTry()
	if !ok {
		return
	}
	defer release()

	runMonthlyResetOnce()
	runCycleResetOnce()
}

// runMonthlyResetOnce 每月到达配置的重置日后首个 tick，将生效策略为 monthly 的用户额度清零并留痕
func runMonthlyResetOnce() {
	day, hour := operation_setting.GetQuotaResetDayHour()
	now := time.Now()
	// 未到重置日当天配置时刻
	if now.Day() != day || now.Hour() < hour {
		return
	}

	month := now.Format("2006-01")
	done, err := model.HasResetRecordForMonth(month)
	if err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("quota monthly reset: query last record failed: %v", err))
		return
	}
	if done {
		return
	}

	count, err := model.ResetMonthlyQuotaUsers()
	if err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("quota monthly reset failed: %v", err))
		return
	}
	if _, err := model.CreateAutoResetRecord(count, month, common.GetTimestamp()); err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("quota monthly reset: create record failed: %v", err))
		return
	}
	// 独立标记与留痕并存：留痕记录被删除后仍能阻止当月重复清零
	if err := model.MarkMonthlyResetDone(month); err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("quota monthly reset: mark month failed: %v", err))
	}
	logger.LogInfo(context.Background(), fmt.Sprintf("quota monthly reset done: month=%s reset_count=%d", month, count))
}

// runCycleResetOnce 扫描到期的周期清零用户批量清零；按天聚合留痕
func runCycleResetOnce() {
	count, err := model.ResetCycleQuotaUsers()
	if err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("quota cycle reset failed: %v", err))
		return
	}
	if count == 0 {
		return
	}
	remark := "cycle:" + time.Now().Format("2006-01-02")
	if _, err := model.CreateAutoResetRecord(count, remark, common.GetTimestamp()); err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("quota cycle reset: create record failed: %v", err))
		return
	}
	logger.LogInfo(context.Background(), fmt.Sprintf("quota cycle reset done: date=%s reset_count=%d", remark, count))
}

// runDueQuotaImports 执行所有到达计划时间的待执行任务；
// DB 抢占（pending→executing）防多节点/手动触发并发重复执行
func runDueQuotaImports() {
	due, err := model.GetDuePendingQuotaImports(common.GetTimestamp())
	if err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("quota import scheduler: query due tasks failed: %v", err))
		return
	}
	for _, task := range due {
		release, ok := QuotaImportLockTry()
		if !ok {
			return
		}
		claimed, err := model.ClaimQuotaImportExecution(task.Id)
		if err != nil || !claimed {
			release()
			continue
		}
		record, err := model.ExecuteQuotaImportRecord(task.Id)
		release()
		if err != nil {
			logger.LogWarn(context.Background(), fmt.Sprintf("quota import task %d execute failed: %v", task.Id, err))
			continue
		}
		logger.LogInfo(context.Background(), fmt.Sprintf("quota import task %d executed: status=%s ok=%d reset=%d",
			task.Id, record.Status, record.SuccessRows, record.ResetCount))
	}
}

// ExecuteQuotaImportTask 手动立即执行一个任务（preview+execute 接口共用路径）
func ExecuteQuotaImportTask(recordId int) (*model.QuotaImportRecord, error) {
	release, ok := QuotaImportLockTry()
	if !ok {
		return nil, fmt.Errorf("当前有导入/清零任务正在执行，请稍后重试")
	}
	defer release()

	claimed, err := model.ClaimQuotaImportExecution(recordId)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, fmt.Errorf("任务不存在或不是待执行状态")
	}
	record, err := model.ExecuteQuotaImportRecord(recordId)
	if err != nil {
		return nil, err
	}
	logger.LogInfo(context.Background(), fmt.Sprintf("quota import task %d executed manually: status=%s ok=%d reset=%d",
		recordId, record.Status, record.SuccessRows, record.ResetCount))
	return record, nil
}
