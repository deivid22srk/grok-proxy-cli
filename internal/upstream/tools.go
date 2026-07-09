package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"grok-desktop/internal/store"
)

// ToolCall is an OpenAI-style function call from the model.
type ToolCall struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// CompleteResult is a non-stream chat.completions response (used for tool rounds).
type CompleteResult struct {
	ID           string
	Model        string
	Content      string
	Reasoning    string
	ToolCalls    []ToolCall
	FinishReason string
	Usage        *Usage
}

func webSearchToolDef() []map[string]any {
	return []map[string]any{
		{
			"type": "function",
			"function": map[string]any{
				"name":        "web_search",
				"description": "Search the live web with DuckDuckGo. Use when you need current events, facts you are unsure about, recent news, prices, docs online, or anything time-sensitive. Do NOT use for pure math, coding logic you already know, or casual chit-chat.",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "Focused search query (keywords, not a full essay).",
						},
					},
					"required": []string{"query"},
				},
			},
		},
	}
}

func toolsSystemPreamble() string {
	return `You are Grok. You may call the web_search tool (DuckDuckGo) when up-to-date or external information would improve the answer.
Rules:
- Call web_search only when necessary (news, current facts, unknown details).
- Prefer one precise query; you may call again if results are insufficient.
- After tool results, answer the user clearly and cite URLs when useful.
- Never invent search results.`
}

// messagesToAPI converts ChatMessage slice into OpenAI wire format (incl. tools).
func messagesToAPI(msgs []ChatMessage) []map[string]any {
	out := make([]map[string]any, 0, len(msgs)+1)
	for _, m := range msgs {
		item := map[string]any{"role": m.Role}
		if m.Content != "" || m.Role != "assistant" || len(m.ToolCalls) == 0 {
			item["content"] = m.Content
		}
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			tcs := make([]map[string]any, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				tcs = append(tcs, map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Name,
						"arguments": tc.Arguments,
					},
				})
			}
			item["tool_calls"] = tcs
			if m.Content == "" {
				item["content"] = nil
			}
		}
		if m.Role == "tool" {
			item["tool_call_id"] = m.ToolCallID
			if m.Name != "" {
				item["name"] = m.Name
			}
			item["content"] = m.Content
		}
		out = append(out, item)
	}
	return out
}

// ChatComplete runs a single non-streaming completion (optionally with tools).
func (c *Client) ChatComplete(
	ctx context.Context,
	token string,
	settings store.Settings,
	model, effort string,
	messages []ChatMessage,
	withTools bool,
) (*CompleteResult, error) {
	body := map[string]any{
		"model":            model,
		"messages":         messagesToAPI(messages),
		"stream":           false,
		"reasoning_effort": effort,
	}
	if withTools {
		body["tools"] = webSearchToolDef()
		body["tool_choice"] = "auto"
	}
	raw, _ := json.Marshal(body)
	url := c.baseURL(settings) + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header = c.authHeaders(token, settings.ClientVersion)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("chat complete HTTP %d: %s", resp.StatusCode, string(b))
	}
	var parsed map[string]any
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil, err
	}
	out := &CompleteResult{
		ID:    strField(parsed["id"]),
		Model: strField(parsed["model"]),
	}
	if u, ok := parsed["usage"].(map[string]any); ok {
		out.Usage = parseChatUsage(u)
	}
	choices, _ := parsed["choices"].([]any)
	if len(choices) == 0 {
		return nil, fmt.Errorf("empty choices")
	}
	ch, _ := choices[0].(map[string]any)
	out.FinishReason = strField(ch["finish_reason"])
	msg, _ := ch["message"].(map[string]any)
	if msg == nil {
		return out, nil
	}
	out.Content = strField(msg["content"])
	out.Reasoning = strField(msg["reasoning_content"])
	if tcs, ok := msg["tool_calls"].([]any); ok {
		for _, rawTC := range tcs {
			m, ok := rawTC.(map[string]any)
			if !ok {
				continue
			}
			fn, _ := m["function"].(map[string]any)
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:        strField(m["id"]),
				Type:      strField(m["type"]),
				Name:      strField(fn["name"]),
				Arguments: strField(fn["arguments"]),
			})
		}
	}
	return out, nil
}

func strField(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

// ensureSystemToolsHint prepends/merges tool instructions once.
func ensureSystemToolsHint(msgs []ChatMessage) []ChatMessage {
	hint := toolsSystemPreamble()
	if len(msgs) > 0 && msgs[0].Role == "system" {
		if !strings.Contains(msgs[0].Content, "web_search") {
			msgs[0].Content = hint + "\n\n" + msgs[0].Content
		}
		return msgs
	}
	return append([]ChatMessage{{Role: "system", Content: hint}}, msgs...)
}

// ParseSearchQuery extracts {"query":"..."} from tool arguments JSON.
func ParseSearchQuery(args string) string {
	args = strings.TrimSpace(args)
	if args == "" {
		return ""
	}
	var m map[string]any
	if json.Unmarshal([]byte(args), &m) == nil {
		if q, ok := m["query"].(string); ok {
			return strings.TrimSpace(q)
		}
		if q, ok := m["q"].(string); ok {
			return strings.TrimSpace(q)
		}
	}
	// bare string fallback
	return strings.Trim(args, `"' `)
}

// AccumUsage merges usage counters.
func AccumUsage(dst *Usage, src *Usage) *Usage {
	if src == nil {
		return dst
	}
	if dst == nil {
		cp := *src
		return &cp
	}
	dst.PromptTokens += src.PromptTokens
	dst.CompletionTokens += src.CompletionTokens
	dst.ReasoningTokens += src.ReasoningTokens
	dst.CachedTokens += src.CachedTokens
	dst.TotalTokens += src.TotalTokens
	if dst.TotalTokens == 0 {
		dst.TotalTokens = dst.PromptTokens + dst.CompletionTokens
	}
	return dst
}

// StreamChatWithTools runs an agentic loop: model may call web_search zero or more times.
// searchFn executes a query and returns JSON string content for the tool role message.
func (c *Client) StreamChatWithTools(
	ctx context.Context,
	token string,
	settings store.Settings,
	model, effort string,
	req ChatRequest,
	emit func(StreamEvent),
	searchFn func(ctx context.Context, query string) (resultJSON string, err error),
) error {
	msgs := ensureSystemToolsHint(append([]ChatMessage{}, req.Messages...))
	var totalUsage *Usage
	t0 := time.Now()
	var ttftMs int64
	const maxRounds = 5

	for round := 0; round < maxRounds; round++ {
		// Non-stream tool planning round (reliable tool_calls)
		complete, err := c.ChatComplete(ctx, token, settings, model, effort, msgs, true)
		if err != nil {
			return err
		}
		totalUsage = AccumUsage(totalUsage, complete.Usage)
		if complete.Reasoning != "" {
			if ttftMs == 0 {
				ttftMs = time.Since(t0).Milliseconds()
			}
			emit(StreamEvent{Type: "thinking", Text: complete.Reasoning + "\n", ID: complete.ID, Model: complete.Model})
		}

		if len(complete.ToolCalls) > 0 {
			// Record assistant tool call turn
			asst := ChatMessage{
				Role:      "assistant",
				Content:   complete.Content,
				ToolCalls: complete.ToolCalls,
			}
			msgs = append(msgs, asst)

			for _, tc := range complete.ToolCalls {
				name := tc.Name
				if name == "" {
					name = "web_search"
				}
				emit(StreamEvent{
					Type:  "tool_call",
					ID:    tc.ID,
					Model: complete.Model,
					Text:  name,
					// reuse fields
				})
				// richer payload via a parallel event shape using Error field? better add Tool meta in Text as JSON
				// We'll emit dedicated tool_call with arguments in a second convention:
				args := tc.Arguments
				emit(StreamEvent{Type: "tool_args", ID: tc.ID, Text: args, Model: complete.Model})

				var toolContent string
				if name == "web_search" && searchFn != nil {
					q := ParseSearchQuery(args)
					emit(StreamEvent{Type: "search_query", ID: tc.ID, Text: q, Model: complete.Model})
					toolContent, err = searchFn(ctx, q)
					if err != nil {
						toolContent = fmt.Sprintf(`{"error":%q}`, err.Error())
						emit(StreamEvent{Type: "tool_error", ID: tc.ID, Error: err.Error()})
					}
				} else {
					toolContent = `{"error":"unknown tool"}`
				}
				msgs = append(msgs, ChatMessage{
					Role:       "tool",
					ToolCallID: tc.ID,
					Name:       name,
					Content:    toolContent,
				})
				emit(StreamEvent{Type: "tool_done", ID: tc.ID, Text: name})
			}
			// continue loop for model to read tool results
			continue
		}

		// Final answer: stream a second request for nicer UX if content empty? 
		// We already have content from complete — stream-emulate or stream again.
		if strings.TrimSpace(complete.Content) != "" {
			if ttftMs == 0 {
				ttftMs = time.Since(t0).Milliseconds()
			}
			// Emit as content (single chunk is fine; markdown UI handles it)
			emit(StreamEvent{Type: "content", Text: complete.Content, ID: complete.ID, Model: complete.Model})
			lat := time.Since(t0).Milliseconds()
			if totalUsage == nil {
				totalUsage = &Usage{}
			}
			emit(StreamEvent{
				Type: "usage", Usage: totalUsage, ID: complete.ID, Model: complete.Model,
				LatencyMs: lat, TTFTMs: ttftMs,
			})
			emit(StreamEvent{
				Type: "done", ID: complete.ID, Model: complete.Model, Usage: totalUsage,
				LatencyMs: lat, TTFTMs: ttftMs,
			})
			return nil
		}

		// Empty content without tools — try streaming pass without forcing tools
		req2 := req
		req2.Messages = msgs
		req2.WebSearch = false
		return c.streamChatCompletions(ctx, token, settings, model, effort, req2, emit)
	}

	return fmt.Errorf("tool loop exceeded %d rounds", maxRounds)
}
