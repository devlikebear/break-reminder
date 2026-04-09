## Codebase Analysis

Architecture and module analysis available at `.analysis/AI_CONTEXT.md`.
Read it first when you need to understand the project structure, dependencies, or key data flows.

## Development Workflow

### PR Review

1. `gh pr view <N>` + `gh pr diff <N>` 으로 변경 내용 파악
2. 관련 소스 파일을 읽고 변경의 정확성, 엣지 케이스, 보안 검토
3. 테스트가 실제 런타임 조건(기본 check interval 60초, launchd 60초)에서도 유효한지 확인
4. 승인/변경요청은 `gh pr review <N> --approve/--request-changes --body "..."`

### Merge (fork PR)

외부 fork에서 온 PR은 `gh pr merge`가 안 될 수 있음. 로컬 머지 절차:

1. `gh pr checkout <N>` → `git fetch origin main` → `git merge origin/main`
2. conflict 해결 후 `go test ./...` 확인
3. `git checkout main` → `git merge <branch> --no-ff`
4. `git push origin main`

### Release

1. `VERSION` 파일 업데이트 (예: `0.4.0`)
2. 커밋: `chore: bump version to <version>`
3. `git push origin main`
4. `git tag v<version>` → `git push origin v<version>`
5. GitHub Actions release workflow가 자동 실행:
   - Go + Swift 빌드, 테스트
   - GitHub Release 생성 (바이너리 첨부)
   - Homebrew formula 업데이트 (로컬 + tap repo)

**주의**: VERSION 파일과 태그 버전이 일치해야 CI가 통과함.

### Versioning

- feat PR 포함 → minor bump (0.3.0 → 0.4.0)
- fix만 → patch bump (0.3.0 → 0.3.1)

### Issue 등록

- 버그/레이아웃 문제 등은 `gh issue create`로 등록
- 스크린샷이 필요한 경우 GitHub 웹에서 이미지 첨부

## Testing

```bash
go test ./...           # 전체 테스트
go test ./internal/timer  # 특정 패키지
```

## Key Review Checklist

- config validation: `merge()` 에서 zero-value와 null 값 구분 (`raw` map 활용)
- timer 로직: 기본 CheckIntervalSec(60초)과 launchd(60초) 기준으로 실제 도달 가능한 타이밍인지
- 상태 파일: `state.Save()`에서 새 필드 추가 시 Load/Save 양쪽 반영 확인
- break 전환: `EnterBreak()` 헬퍼를 통해 일관된 상태 초기화
