package operation_setting

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

// 额度展示类型
const (
	QuotaDisplayTypeUSD    = "USD"
	QuotaDisplayTypeCNY    = "CNY"
	QuotaDisplayTypeTokens = "TOKENS"
	QuotaDisplayTypeCustom = "CUSTOM"
)

// 站点基准货币：内部记账单位（quota/QuotaPerUnit）的实际计价币种
const (
	BaseCurrencyUSD = "USD"
	BaseCurrencyCNY = "CNY"
)

// 额度清零策略（作为新用户默认；用户可单独覆盖为其一）
const (
	QuotaResetModeMonthly = "monthly" // 每月固定日清零
	QuotaResetModeCycle   = "cycle"   // 开通周期清零
	QuotaResetModeNever   = "never"   // 永不清零，用完为止
)

const defaultQuotaResetCycleDays = 30

type GeneralSetting struct {
	DocsLink            string `json:"docs_link"`
	PingIntervalEnabled bool   `json:"ping_interval_enabled"`
	PingIntervalSeconds int    `json:"ping_interval_seconds"`
	// 当前站点额度展示类型：USD / CNY / TOKENS
	QuotaDisplayType string `json:"quota_display_type"`
	// 站点基准货币：USD / CNY。基准为 CNY 时，系统数据视为人民币口径，
	// 人民币展示不换算、美元展示除以汇率；基准为 USD 时保持原逻辑
	BaseCurrency string `json:"base_currency"`
	// 自定义货币符号，用于 CUSTOM 展示类型
	CustomCurrencySymbol string `json:"custom_currency_symbol"`
	// 自定义货币与美元汇率（1 USD = X Custom）
	CustomCurrencyExchangeRate float64 `json:"custom_currency_exchange_rate"`
	// 月度额度重置日（每月第几天，1-28）与执行小时（0-23），仅 monthly 策略使用
	QuotaResetDay  int `json:"quota_reset_day"`
	QuotaResetHour int `json:"quota_reset_hour"`
	// cycle 策略的周期天数
	QuotaResetCycleDays int `json:"quota_reset_cycle_days"`
}

// 默认配置
var generalSetting = GeneralSetting{
	DocsLink:                   "https://docs.newapi.pro",
	PingIntervalEnabled:        false,
	PingIntervalSeconds:        60,
	QuotaDisplayType:           QuotaDisplayTypeCNY,
	BaseCurrency:               BaseCurrencyCNY,
	CustomCurrencySymbol:       "¤",
	CustomCurrencyExchangeRate: 1.0,
	QuotaResetCycleDays:        defaultQuotaResetCycleDays,
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("general_setting", &generalSetting)
}

func GetGeneralSetting() *GeneralSetting {
	return &generalSetting
}

// IsCurrencyDisplay 是否以货币形式展示（美元或人民币）
func IsCurrencyDisplay() bool {
	return generalSetting.QuotaDisplayType != QuotaDisplayTypeTokens
}

// IsValidQuotaResetMode 校验清零策略取值（每个用户必须有明确策略，无"跟随默认"）
func IsValidQuotaResetMode(mode string) bool {
	switch mode {
	case QuotaResetModeMonthly, QuotaResetModeCycle, QuotaResetModeNever:
		return true
	}
	return false
}

// GetQuotaResetDayHour 返回月度清零的重置日（1-28）与执行小时（0-23），非法配置回退每月 1 日 0 点
func GetQuotaResetDayHour() (day, hour int) {
	day, hour = 1, 0
	if d := generalSetting.QuotaResetDay; d >= 1 && d <= 28 {
		day = d
	}
	if h := generalSetting.QuotaResetHour; h >= 0 && h <= 23 {
		hour = h
	}
	return
}

// MonthlyResetWindow 返回 now 所处的月度清零周期窗口：
// effective 为本周期起算（上一次月度清零）时刻，next 为下一次月度清零时刻
func MonthlyResetWindow(now time.Time) (effective, next time.Time) {
	day, hour := GetQuotaResetDayHour()
	loc := now.Location()
	candidate := time.Date(now.Year(), now.Month(), day, hour, 0, 0, 0, loc)
	if candidate.After(now) {
		return time.Date(now.Year(), now.Month()-1, day, hour, 0, 0, 0, loc), candidate
	}
	return candidate, time.Date(now.Year(), now.Month()+1, day, hour, 0, 0, 0, loc)
}

// GetQuotaResetCycleSeconds 返回 cycle 策略的周期秒数，非法配置回退 30 天
func GetQuotaResetCycleSeconds() int64 {
	days := generalSetting.QuotaResetCycleDays
	if days <= 0 {
		days = defaultQuotaResetCycleDays
	}
	return int64(days) * 86400
}

// IsBaseCNY 站点基准货币是否为人民币（内部记账单位实际为人民币口径）
func IsBaseCNY() bool {
	return generalSetting.BaseCurrency == BaseCurrencyCNY
}

// GetQuotaDisplayType 返回额度展示类型
func GetQuotaDisplayType() string {
	return generalSetting.QuotaDisplayType
}

// SafeUSDExchangeRate 返回兜底为 1 的美元汇率，避免错误配置导致乘零或除零
func SafeUSDExchangeRate() float64 {
	if USDExchangeRate > 0 {
		return USDExchangeRate
	}
	return 1
}

// QuotaToDisplayAmount 将内部额度统一换算为当前展示类型的币种数值（不含符号与格式化）。
// 该函数是所有服务端金额展示换算的唯一出口：
//   - 基准 USD：CNY 展示 = internal × 汇率（原逻辑）
//   - 基准 CNY：CNY 展示 = internal（不换算），USD 展示 = internal ÷ 汇率，
//     CUSTOM 展示 = internal × 自定义汇率 ÷ 汇率
func QuotaToDisplayAmount(quota float64) float64 {
	switch generalSetting.QuotaDisplayType {
	case QuotaDisplayTypeTokens:
		return quota
	case QuotaDisplayTypeCNY:
		internal := quota / common.QuotaPerUnit
		if IsBaseCNY() {
			return internal
		}
		return internal * SafeUSDExchangeRate()
	case QuotaDisplayTypeCustom:
		internal := quota / common.QuotaPerUnit
		rate := generalSetting.CustomCurrencyExchangeRate
		if rate <= 0 {
			rate = 1
		}
		if IsBaseCNY() {
			return internal * rate / SafeUSDExchangeRate()
		}
		return internal * rate
	default: // USD
		internal := quota / common.QuotaPerUnit
		if IsBaseCNY() {
			return internal / SafeUSDExchangeRate()
		}
		return internal
	}
}
