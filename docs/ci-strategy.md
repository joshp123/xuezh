# CI packaging strategy (POC vs production)

## Problem
GitHub runners hit disk exhaustion when combining Nix + Python deps (torch/openai-whisper).

## Options
1) **Nix CI**
   - Pros: matches local devenv, reproducible.
   - Cons: heavy store + pip wheels; disk pressure on GitHub runners.

2) **Python-only CI**
   - Pros: smaller footprint, simpler setup.
   - Cons: loses parity with Nix; still large for torch/whisper.

3) **Docker layering (prebuilt image)**
   - Pros: cache heavy deps (torch/whisper/ffmpeg) once; fast CI.
   - Cons: extra image maintenance; separate build pipeline.

## Decision
- **POC**: no CI beyond local `./scripts/check.sh`.
- **Production**: use **Python-only CI + Docker layering** for heavy audio deps. Keep Nix for local dev.

## Implications
- Audio backend tests can live in a separate workflow that uses the prebuilt image.
- Main CI can skip heavy audio installs and still run contract/unit/integration tests.
- Update this doc if CI reliability or dependency footprint changes.
