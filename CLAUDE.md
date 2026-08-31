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

Release 워크플로 수동 실행(`workflow_dispatch`)이 기본 경로. 로컬에서 태그를 만들 필요 없음.

1. 릴리스할 변경사항이 main에 머지돼 있고 CHANGELOG `## [Unreleased]` 섹션이 채워졌는지 확인
2. Actions → Release → Run workflow에서 `bump` 선택 (또는 `gh workflow run release.yml -f bump=minor`)
   - `patch`/`minor`/`major`: 워크플로가 `VERSION`을 올리고 CHANGELOG의 Unreleased를 버전 섹션으로 바꿔 커밋한 뒤 태그 생성
   - `none`: `VERSION` 파일 값을 그대로 릴리스 (준비 커밋을 이미 만들어 둔 경우)
3. 이후 자동 진행: Go + Swift 빌드/테스트 → 버전 커밋 → 태그 push → GitHub Release 생성(바이너리 첨부) → Homebrew formula 갱신 (로컬 + tap repo)

버전 커밋과 태그는 **빌드·테스트 통과 후에만** 생성되므로 실패해도 main은 그대로 남는다.
봇이 push한 커밋은 CI를 다시 트리거하지 않지만, 릴리스 잡이 같은 테스트를 이미 돌린다.

수동 태그 push(`git tag v<version> && git push origin v<version>`)도 그대로 동작한다.
**주의**: 이 경로에서는 VERSION 파일과 태그 버전이 일치해야 워크플로가 통과함.

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
