#!/usr/bin/env bash
# Dependency gate (renewal Phase 2, T2.B.1):
#  - only internal/provider and internal/agent may import tars pkg/llm, so the
#    timer, tool layer and engine stay model-free;
#  - tars pkg/agentloop and pkg/session (Experimental in tars) are not used.
set -euo pipefail
cd "$(dirname "$0")/.."

MODULE=github.com/devlikebear/break-reminder
LLM=github.com/devlikebear/tars/pkg/llm
allowed="$MODULE/internal/provider $MODULE/internal/agent"
status=0

for tags in "" "mas"; do
  while read -r pkg imports; do
    case " $imports " in
      *" $LLM "*)
        case " $allowed " in
          *" $pkg "*) ;;
          *) echo "dependency gate: $pkg imports $LLM (only internal/provider and internal/agent may)"; status=1 ;;
        esac ;;
    esac
  done < <(go list -tags "$tags" -f '{{.ImportPath}} {{join .Imports " "}} {{join .TestImports " "}}' ./...)

  if go list -tags "$tags" -deps ./... | grep -E '^github.com/devlikebear/tars/pkg/(agentloop|session)(/|$)'; then
    echo "dependency gate: tars pkg/agentloop or pkg/session is linked (build tags: '${tags:-none}')"
    status=1
  fi
done

[[ $status == 0 ]] && echo "dependency gate: ok"
exit $status
