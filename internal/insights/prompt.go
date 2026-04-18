package insights

import (
	"encoding/json"
	"fmt"

	"github.com/devlikebear/break-reminder/internal/ai"
)

const promptTemplate = `다음은 사용자의 최근 작업/휴식 기록입니다:

%s

다음 두 가지를 분석하여 JSON으로만 응답하세요 (마크다운 코드 블록 없이):

1. daily_report: 오늘의 작업/휴식 요약을 한국어 2-3문장으로 작성
2. patterns: 눈에 띄는 패턴 2-3가지를 배열로 작성 (각 항목은 type, title, description, suggestion 필드 포함)

type 값은 다음 중 하나:
- "warning": 주의가 필요한 패턴 (예: 슬럼프, 휴식 부족)
- "positive": 긍정적 개선 추세
- "info": 중립적 관찰 (예: 최적 작업 시간대)

응답 형식:
{
  "daily_report": "...",
  "patterns": [
    {"type": "warning", "title": "...", "description": "...", "suggestion": "..."},
    {"type": "positive", "title": "...", "description": "...", "suggestion": "..."}
  ]
}`

// BuildPrompt constructs the AI prompt from the given history entries.
func BuildPrompt(history []ai.DailySummary) string {
	if history == nil {
		history = []ai.DailySummary{}
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		data = []byte("[]")
	}
	return fmt.Sprintf(promptTemplate, string(data))
}
