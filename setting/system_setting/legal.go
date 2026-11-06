package system_setting

import (
	"encoding/json"
	"strings"
)

// LegalFallbackLanguage 是“默认”内容的存储键，用于所有语言都未单独配置时的兜底。
const LegalFallbackLanguage = ""

// LegalLanguageOrder 决定所有语言都缺失时挑选内容的顺序，需与前端保持一致。
var LegalLanguageOrder = []string{"zhCN", "en", "fr", "ru", "ja", "vi", "zhTW"}

var legalResolutionOrder = append([]string{LegalFallbackLanguage, "en"}, LegalLanguageOrder...)

// 协议内容的存储与解析工具：内容/名称均为 JSON 对象（语言 -> 内容）字符串，
// 历史数据是纯文本，读取时视为默认语言内容。
// 存储本体在 legal_doc_versions 表（最新行 = 当前内容），不再走 option 配置。

// ParseLegalLanguageMap 将按语言存储的字段解析为 map；纯文本历史数据归入默认语言。
func ParseLegalLanguageMap(raw string) map[string]string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return map[string]string{}
	}
	if strings.HasPrefix(trimmed, "{") {
		var parsed map[string]string
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil && parsed != nil {
			return parsed
		}
	}
	return map[string]string{LegalFallbackLanguage: raw}
}

// NormalizeLegalLanguage 把 BCP-47 或界面语言码归一为配置中使用的语言键。
func NormalizeLegalLanguage(lang string) string {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(lang), "_", "-"))
	switch normalized {
	case "":
		return ""
	case "zh", "zh-cn", "zh-hans", "zhcn":
		return "zhCN"
	case "zh-tw", "zh-hk", "zh-mo", "zh-hant", "zhtw":
		return "zhTW"
	}
	if index := strings.Index(normalized, "-"); index > 0 {
		return normalized[:index]
	}
	return normalized
}

// ResolveLegalLanguageValue 按“指定语言 -> 默认 -> 英文 -> 其余语言”的顺序取内容。
func ResolveLegalLanguageValue(raw string, lang string) string {
	values := ParseLegalLanguageMap(raw)
	normalized := NormalizeLegalLanguage(lang)
	if normalized != "" {
		if value, ok := values[normalized]; ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	for _, candidate := range legalResolutionOrder {
		if value, ok := values[candidate]; ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// IsLegalValueConfigured 判断字段是否在任一语言下配置了内容。
func IsLegalValueConfigured(raw string) bool {
	for _, value := range ParseLegalLanguageMap(raw) {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}
