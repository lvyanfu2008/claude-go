package compo

import "goc/gou/ink"

// ProcessMessages applies the message processing pipeline.
func ProcessMessages(msgs []ink.Message) []ink.Message {
	msgs = normalizeMessages(msgs)
	msgs = reorderMessages(msgs)
	msgs = groupToolUses(msgs)
	msgs = collapseReadSearch(msgs)
	return msgs
}

func normalizeMessages(msgs []ink.Message) []ink.Message {
	for i := range msgs {
		if msgs[i].ContentBlocks == nil {
			msgs[i].ContentBlocks = []ink.ContentBlock{}
		}
	}
	return msgs
}

func reorderMessages(msgs []ink.Message) []ink.Message {
	result := make([]ink.Message, 0, len(msgs))
	for i := 0; i < len(msgs); i++ {
		msg := msgs[i]
		if msg.Type == "user" && i > 0 && len(result) > 0 {
			prev := &result[len(result)-1]
			if prev.Type == "assistant" {
				merged := false
				for bi := range msg.ContentBlocks {
					block := &msg.ContentBlocks[bi]
					if block.Type == "tool_result" {
						for pi := range prev.ContentBlocks {
							if prev.ContentBlocks[pi].Type == "tool_use" && prev.ContentBlocks[pi].State != "resolved" {
								prev.ContentBlocks[pi].Result = block
								prev.ContentBlocks[pi].State = "resolved"
								merged = true
								break
							}
						}
					}
				}
				if merged {
					continue
				}
			}
		}
		result = append(result, msg)
	}
	return result
}

func groupToolUses(msgs []ink.Message) []ink.Message {
	return msgs
}

func collapseReadSearch(msgs []ink.Message) []ink.Message {
	result := make([]ink.Message, 0, len(msgs))
	i := 0
	for i < len(msgs) {
		msg := msgs[i]
		if msg.Type == "assistant" && hasCollapsibleToolUse(msg) {
			group := ink.Message{
				UUID: msg.UUID,
				Type: "collapsed_read_search",
				Meta: make(map[string]interface{}),
			}
			var items []ink.Message
			j := i
			for j < len(msgs) {
				if msgs[j].Type == "assistant" && hasCollapsibleToolUse(msgs[j]) {
					items = append(items, msgs[j])
					j++
					if j < len(msgs) && msgs[j].Type == "user" {
						items = append(items, msgs[j])
						j++
					}
				} else {
					break
				}
			}
			group.Meta["items"] = items
			result = append(result, group)
			i = j
		} else {
			result = append(result, msg)
			i++
		}
	}
	return result
}

func hasCollapsibleToolUse(msg ink.Message) bool {
	for _, block := range msg.ContentBlocks {
		if block.Type == "tool_use" {
			switch block.Name {
			case "Read", "Grep", "Glob":
				return true
			}
		}
	}
	return false
}
