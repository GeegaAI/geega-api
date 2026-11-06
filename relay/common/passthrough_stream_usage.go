package common

import (
	"fmt"

	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// InjectPassThroughStreamUsage 在 pass-through（透传原始请求体）场景下，
// 为 OpenAI Chat Completions 流式请求补上 stream_options.include_usage=true。
//
// 背景：非透传路径会在 relay/compatible_handler.go 与各 adaptor 中强制
// include_usage=true；但透传路径发送的是客户端原始 body，客户端未开启时
// 上游（OpenAI / vLLM 等）不会在最后一条 SSE 返回 usage，结算只能退化为
// 本地估算。这里用 sjson 做最小侵入注入，保留 body 的其余字节与字段。
//
// 返回：
//   - body：注入后的 body；未注入时原样返回
//   - injected：是否真的发生了注入
//   - err：sjson 写入失败时返回；调用方应记录日志并回退到原始 body
func InjectPassThroughStreamUsage(body []byte, info *RelayInfo) ([]byte, bool, error) {
	if info == nil || !info.IsStream || len(body) == 0 {
		return body, false, nil
	}
	// stream_options 仅由 Chat Completions 定义；其他 relay mode 注入可能被上游拒绝。
	if info.RelayMode != relayconstant.RelayModeChatCompletions {
		return body, false, nil
	}
	if !info.SupportStreamOptions {
		return body, false, nil
	}
	if !gjson.ValidBytes(body) {
		return body, false, nil
	}
	parsed := gjson.ParseBytes(body)
	if !parsed.IsObject() {
		return body, false, nil
	}
	// body 显式 stream=false 时不要改写成流式请求语义。
	if stream := parsed.Get("stream"); stream.Exists() && !stream.Bool() {
		return body, false, nil
	}
	// stream_options 存在但不是对象（null/字符串等）时跳过，避免破坏异常 body。
	if streamOptions := parsed.Get("stream_options"); streamOptions.Exists() && !streamOptions.IsObject() {
		return body, false, nil
	}
	if parsed.Get("stream_options.include_usage").Bool() {
		return body, false, nil
	}

	patched, err := sjson.SetBytes(body, "stream_options.include_usage", true)
	if err != nil {
		return body, false, fmt.Errorf("inject stream_options.include_usage: %w", err)
	}
	return patched, true, nil
}
