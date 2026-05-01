package detector

import (
	"testing"
	"time"

	larkadapter "feishu-mem/internal/lark-adapter"
)

func TestMinutesDetector(t *testing.T) {
	cfg := larkadapter.LoadConfig()
	detector := larkadapter.NewMinutesExtractor(cfg)

	// 第一次检测：基线
	t.Run("baseline detection", func(t *testing.T) {
		result, err := larkadapter.ExtractDetect(detector)
		if err != nil {
			t.Errorf("检测失败: %v", err)
			return
		}
		t.Logf("基线检测结果: HasChanges=%v", result.HasChanges)
	})

	// 第二次检测：模拟1小时后
	t.Run("incremental detection", func(t *testing.T) {
		fakeLastCheck := time.Now().Add(-1 * time.Hour)
		result, err := detector.Detect(fakeLastCheck)
		if err != nil {
			t.Errorf("检测失败: %v", err)
			return
		}
		t.Logf("增量检测结果: HasChanges=%v, Changes=%d", result.HasChanges, len(result.Changes))
		for _, c := range result.Changes {
			t.Logf("  - [%s] %s: %s", c.Type, c.EntityType, c.Summary)
		}
	})
}
