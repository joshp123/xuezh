# Examples (copy/paste)

These examples assume the authoritative CLI contract in `docs/cli-contract.md`.

## Review
```bash
xuezh snapshot --window 30d --due-limit 10 --evidence-limit 200 --max-bytes 200000 --json
xuezh review start --limit 10 --json
# ... user attempts ...
xuezh review grade --item w_aaaaaaaaaaaa --grade 4 --json
```

## Tone practice (word-level)
```bash
xuezh audio tts --text "时间" --voice XiaoxiaoNeural --out /tmp/shijian.ogg --json
# user records voice note to /tmp/voice.ogg
xuezh audio process-voice --in /tmp/voice.ogg --ref-text "时间" --json
```
