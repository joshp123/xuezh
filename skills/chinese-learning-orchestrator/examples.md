# Examples (copy/paste)

These examples assume the authoritative CLI contract in `docs/cli-contract.md`.

## Review
```bash
chlearn snapshot --window 30d --due-limit 10 --evidence-limit 200 --max-bytes 200000 --json
chlearn review start --limit 10 --json
# ... user attempts ...
chlearn review grade --item W123 --grade 4 --json
```

## Tone practice (word-level)
```bash
chlearn audio tts --text "时间" --voice XiaoxiaoNeural --out /tmp/shijian.ogg --json
# user records voice note to /tmp/voice.ogg
chlearn audio process-voice --in /tmp/voice.ogg --ref-text "时间" --mode local --json
```
