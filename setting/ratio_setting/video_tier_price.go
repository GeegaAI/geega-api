package ratio_setting

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

// Video tier pricing: per-model prices differentiated by output resolution and
// whether the request carries video input. Stored as an admin-editable option
// (VideoTierPrice) and hot-reloaded through SyncOptions.
//
// Tier keys: "<resolution>" (no video input) or "<resolution>_video" (with
// video input). Values are entered in CNY per 1M tokens (the Ark price-list
// unit); only their ratios matter, so the tier coefficient sent to task
// billing is tierPrice / baseTierPrice, where the base tier is 480p or 720p
// without video input (the base anchor).
// No built-in defaults: models missing from this option are billed at the
// model ratio without a tier coefficient.
var videoTierResolutions = []string{"480p", "720p", "768p", "1080p", "2k", "4k"}

var videoTierPriceMap = types.NewRWMap[string, map[string]float64]()

func isVideoTierKey(key string) bool {
	for _, resolution := range videoTierResolutions {
		if key == resolution || key == resolution+"_video" {
			return true
		}
	}
	return false
}

func videoTierBasePrice(tiers map[string]float64) (float64, bool) {
	if price, ok := tiers["720p"]; ok && price > 0 {
		return price, true
	}
	if price, ok := tiers["480p"]; ok && price > 0 {
		return price, true
	}
	return 0, false
}

// GetTaskTierRatio 返回视频任务在指定档位下相对基准档的计费系数。
// 配置优先：模型在 VideoTierPrice 中存在时按配置计算（含视频输入/分辨率档）；
// 模型存在但该档位未配置时返回 1.0（按模型倍率基准计费）；
// 模型不存在时返回 false，由适配器回退内置价格表。
func GetTaskTierRatio(modelName, resolution string, hasVideo bool) (float64, bool) {
	tiers, ok := videoTierPriceMap.Get(strings.TrimSpace(modelName))
	if !ok || len(tiers) == 0 {
		return 0, false
	}

	base, ok := videoTierBasePrice(tiers)
	if !ok {
		return 0, false
	}

	key := strings.ToLower(strings.TrimSpace(resolution))
	if hasVideo {
		key += "_video"
	}
	tierPrice, ok := tiers[key]
	if !ok || tierPrice <= 0 {
		return 1.0, true
	}
	return tierPrice / base, true
}

func VideoTierPrice2JSONString() string {
	return videoTierPriceMap.MarshalJSONString()
}

// UpdateVideoTierPriceByJSONString 校验并热更新视频档位价格配置。
// 校验规则：档位键必须在枚举内、价格必须为正数、模型条目需包含基准档
// （480p 或 720p 不含视频）；未配置任何档位的模型条目视为清空，直接忽略。
func UpdateVideoTierPriceByJSONString(jsonStr string) error {
	if strings.TrimSpace(jsonStr) == "" {
		jsonStr = "{}"
	}

	var parsed map[string]map[string]float64
	if err := common.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return fmt.Errorf("invalid video tier price json: %w", err)
	}

	normalized := make(map[string]map[string]float64, len(parsed))
	for model, tiers := range parsed {
		name := strings.TrimSpace(model)
		if name == "" {
			return fmt.Errorf("video tier price: model name cannot be empty")
		}
		normalizedTiers := make(map[string]float64, len(tiers))
		for key, price := range tiers {
			normalizedKey := strings.ToLower(strings.TrimSpace(key))
			if !isVideoTierKey(normalizedKey) {
				return fmt.Errorf("video tier price: unknown tier key %q for model %s", key, name)
			}
			if price <= 0 {
				return fmt.Errorf("video tier price: price for %s %s must be positive", name, normalizedKey)
			}
			normalizedTiers[normalizedKey] = price
		}
		if len(normalizedTiers) == 0 {
			continue
		}
		if _, ok := videoTierBasePrice(normalizedTiers); !ok {
			return fmt.Errorf("video tier price: model %s requires base tier 480p or 720p", name)
		}
		normalized[name] = normalizedTiers
	}

	jsonBytes, err := common.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("video tier price: marshal normalized config failed: %w", err)
	}
	return types.LoadFromJsonStringWithCallback(videoTierPriceMap, string(jsonBytes), nil)
}
