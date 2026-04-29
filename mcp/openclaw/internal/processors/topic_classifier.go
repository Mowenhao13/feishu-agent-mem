package processors

// TopicClassifier 议题归属分类器（openclaw-architecture.md §2.1.1 Stage 2b）
// 根据决策内容判断属于哪个议题（Topic）
type TopicClassifier struct {
	topics []string
}

// NewTopicClassifier 创建议题分类器
func NewTopicClassifier(topics []string) *TopicClassifier {
	return &TopicClassifier{
		topics: topics,
	}
}

// Classify 根据决策内容判断议题归属
// 返回 (topic, confidence)；若无法确定，返回 ("", 0)
func (tc *TopicClassifier) Classify(decisionContent string, keywords []string) (string, float64) {
	if len(tc.topics) == 0 {
		return "", 0
	}

	bestTopic := ""
	bestScore := 0.0

	for _, topic := range tc.topics {
		score := scoreTopicMatch(topic, decisionContent, keywords)
		if score > bestScore {
			bestScore = score
			bestTopic = topic
		}
	}

	if bestScore < 0.3 {
		return "", bestScore
	}

	return bestTopic, bestScore
}

// scoreTopicMatch 计算议题与决策内容的匹配度
func scoreTopicMatch(topic, content string, keywords []string) float64 {
	score := 0.0

	// 议题名出现在内容中
	if contains(content, topic) {
		score += 0.6
	}

	// 关键词匹配
	for _, kw := range keywords {
		if contains(topic, kw) || contains(kw, topic) {
			score += 0.2
		}
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}
