from __future__ import annotations

import typer

from xuezh.core import clock, datasets, db, envelope, events, paths, retention, reports, snapshot as snapshot_core, srs
from xuezh.core.jsonio import dumps

app = typer.Typer(add_completion=False, help="xuezh - local Chinese learning engine (ZFC/Unix-style)")


def _emit(out: dict) -> None:
    typer.echo(dumps(out))
    if out.get("ok") is True:
        raise typer.Exit(code=0)
    raise typer.Exit(code=1)


# ---- Sub-apps (public CLI contract) ----
db_app = typer.Typer(add_completion=False)
dataset_app = typer.Typer(add_completion=False)
review_app = typer.Typer(add_completion=False)
srs_app = typer.Typer(add_completion=False)
report_app = typer.Typer(add_completion=False)
audio_app = typer.Typer(add_completion=False)
content_app = typer.Typer(add_completion=False)
event_app = typer.Typer(add_completion=False)

app.add_typer(db_app, name="db")
app.add_typer(dataset_app, name="dataset")
app.add_typer(review_app, name="review")
app.add_typer(srs_app, name="srs")
app.add_typer(report_app, name="report")
app.add_typer(audio_app, name="audio")
app.add_typer(content_app, name="content")
app.add_typer(event_app, name="event")


# ---- Global commands ----
@app.command()
def version(json_output: bool = typer.Option(False, "--json", help="Output JSON envelope")):
    if json_output:
        _emit(envelope.ok(command="version", data={"version": "0.1.0"}))
    typer.echo("xuezh 0.1.0")


@app.command()
def snapshot(
    window: str = typer.Option("30d", "--window"),
    due_limit: int = typer.Option(80, "--due-limit"),
    evidence_limit: int = typer.Option(200, "--evidence-limit"),
    max_bytes: int = typer.Option(200_000, "--max-bytes"),
    json_output: bool = typer.Option(True, "--json"),
):
    """Bounded context pack for the LLM planner. Single-user system."""
    result = snapshot_core.build_snapshot(
        window=window,
        due_limit=due_limit,
        evidence_limit=evidence_limit,
        max_bytes=max_bytes,
    )
    out = envelope.ok(
        command="snapshot",
        data=result.data,
        artifacts=result.artifacts,
        truncated=result.truncated,
        limits=result.limits,
    )
    _emit(out)


@app.command()
def doctor(json_output: bool = typer.Option(True, "--json")):
    out = envelope.err(
        command="doctor",
        error_type="NOT_IMPLEMENTED",
        message="doctor is not implemented yet (see ticket T-14).",
        details={},
    )
    _emit(out)


@app.command()
def gc(
    apply: bool = typer.Option(False, "--apply", help="Actually delete files"),
    dry_run: bool = typer.Option(True, "--dry-run", help="Preview deletions (default)"),
    json_output: bool = typer.Option(True, "--json"),
):
    if apply and dry_run:
        dry_run = False
    if not apply and not dry_run:
        dry_run = True

    workspace = paths.ensure_workspace()
    candidates = retention.collect_gc_candidates(workspace, now=clock.now_utc())
    rel_candidates = [str(path.relative_to(workspace)) for path in candidates]

    deleted_count = 0
    bytes_freed = 0
    if apply:
        for path in candidates:
            if not path.exists() or not path.is_file():
                continue
            bytes_freed += path.stat().st_size
            path.unlink()
            deleted_count += 1

    out = envelope.ok(
        command="gc",
        data={
            "dry_run": dry_run,
            "apply": apply,
            "candidates": rel_candidates,
            "deleted_count": deleted_count,
            "bytes_freed": bytes_freed,
        },
    )
    _emit(out)


# ---------------- db ----------------
@db_app.command("init")
def db_init(json_output: bool = typer.Option(True, "--json")):
    db_path = db.init_db()
    out = envelope.ok(command="db.init", data={"db_path": str(db_path)})
    _emit(out)


# -------------- dataset --------------
@dataset_app.command("import")
def dataset_import(
    type: str = typer.Option(..., "--type", help="hsk_vocab|hsk_chars|hsk_grammar|frequency"),
    path: str = typer.Option(..., "--path"),
    json_output: bool = typer.Option(True, "--json"),
):
    dataset_id, rows_loaded = datasets.import_dataset(type, path)
    out = envelope.ok(
        command="dataset.import",
        data={"type": type, "rows_loaded": rows_loaded, "dataset_id": dataset_id},
    )
    _emit(out)


# -------------- review --------------
@review_app.command("start")
def review_start(
    limit: int = typer.Option(10, "--limit"),
    json_output: bool = typer.Option(True, "--json"),
):
    now = clock.now_utc()
    items = srs.list_due_items(limit=limit, now=now)
    out = envelope.ok(
        command="review.start",
        data={
            "items": [{"item_id": item.item_id, "due_at": item.due_at} for item in items],
            "generated_at": now.isoformat(),
        },
        limits={"limit": limit},
    )
    _emit(out)


@review_app.command("grade")
def review_grade(
    item: str = typer.Option(..., "--item", help="Item ID (see specs/id-scheme.md)"),
    grade: int = typer.Option(..., "--grade", min=0, max=5),
    next_due: str | None = typer.Option(None, "--next-due"),
    rule: str | None = typer.Option(None, "--rule", help="sm2|leitner"),
    json_output: bool = typer.Option(True, "--json"),
):
    now = clock.now_utc()
    due_at, applied_rule = srs.schedule_next_due(grade=grade, now=now, rule=rule, next_due=next_due)
    srs.upsert_knowledge(item_id=item, due_at=due_at, grade=grade, now=now)
    srs.record_review_event(
        item_id=item,
        event_type="review.grade",
        payload={"grade": grade, "rule": applied_rule, "next_due": due_at},
        now=now,
    )
    out = envelope.ok(
        command="review.grade",
        data={"item": item, "grade": grade, "next_due": due_at, "rule_applied": applied_rule},
    )
    _emit(out)


@review_app.command("bury")
def review_bury(
    item: str = typer.Option(..., "--item"),
    reason: str = typer.Option("unspecified", "--reason"),
    json_output: bool = typer.Option(True, "--json"),
):
    now = clock.now_utc()
    due_at, _ = srs.schedule_next_due(grade=0, now=now, rule="leitner", next_due=None)
    srs.upsert_knowledge(item_id=item, due_at=due_at, grade=None, now=now)
    srs.record_review_event(
        item_id=item,
        event_type="review.bury",
        payload={"reason": reason, "next_due": due_at},
        now=now,
    )
    out = envelope.ok(command="review.bury", data={"item": item, "reason": reason, "next_due": due_at})
    _emit(out)


# -------------- srs --------------
@srs_app.command("preview")
def srs_preview(
    days: int = typer.Option(14, "--days"),
    json_output: bool = typer.Option(True, "--json"),
):
    now = clock.now_utc()
    forecast = srs.preview_due(days=days, now=now)
    out = envelope.ok(command="srs.preview", data={"days": days, "forecast": forecast})
    _emit(out)


# -------------- report --------------
@report_app.command("hsk")
def report_hsk(
    level: str = typer.Option(..., "--level"),
    window: str = typer.Option("30d", "--window"),
    max_items: int = typer.Option(200, "--max-items"),
    max_bytes: int = typer.Option(200_000, "--max-bytes"),
    include_chars: bool = typer.Option(False, "--include-chars", help="Optional: include character audit if dataset exists"),
    json_output: bool = typer.Option(True, "--json"),
):
    result = reports.build_hsk_report(
        level=level,
        window=window,
        max_items=max_items,
        max_bytes=max_bytes,
        include_chars=include_chars,
    )
    out = envelope.ok(
        command="report.hsk",
        data=result.data,
        artifacts=result.artifacts,
        truncated=result.truncated,
        limits=result.limits,
    )
    _emit(out)


@report_app.command("mastery")
def report_mastery(
    item_type: str = typer.Option("word", "--item-type", help="word|character|grammar"),
    window: str = typer.Option("90d", "--window"),
    max_items: int = typer.Option(200, "--max-items"),
    max_bytes: int = typer.Option(200_000, "--max-bytes"),
    json_output: bool = typer.Option(True, "--json"),
):
    result = reports.build_mastery_report(
        item_type=item_type,
        window=window,
        max_items=max_items,
        max_bytes=max_bytes,
    )
    out = envelope.ok(
        command="report.mastery",
        data=result.data,
        artifacts=result.artifacts,
        truncated=result.truncated,
        limits=result.limits,
    )
    _emit(out)


@report_app.command("due")
def report_due(
    limit: int = typer.Option(50, "--limit"),
    max_bytes: int = typer.Option(200_000, "--max-bytes"),
    json_output: bool = typer.Option(True, "--json"),
):
    now = clock.now_utc()
    items = srs.list_due_items(limit=limit, now=now)
    out = envelope.ok(
        command="report.due",
        data={"items": [{"item_id": item.item_id, "due_at": item.due_at} for item in items]},
        limits={"limit": limit, "max_bytes": max_bytes},
    )
    _emit(out)


# -------------- audio --------------
@audio_app.command("convert")
def audio_convert(
    in_path: str = typer.Option(..., "--in"),
    out_path: str = typer.Option(..., "--out"),
    format: str = typer.Option(..., "--format", help="wav|ogg|mp3"),
    backend: str = typer.Option("ffmpeg", "--backend", help="Audio backend id (see specs/audio-backends.md)"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="audio.convert",
        error_type="NOT_IMPLEMENTED",
        message="audio convert is not implemented yet (see ticket T-08).",
        details={"in": in_path, "out": out_path, "format": format, "backend": backend},
    )
    _emit(out)


@audio_app.command("tts")
def audio_tts(
    text: str = typer.Option(..., "--text"),
    voice: str = typer.Option("XiaoxiaoNeural", "--voice"),
    out_path: str = typer.Option(..., "--out"),
    backend: str = typer.Option("edge-tts", "--backend", help="Audio backend id (see specs/audio-backends.md)"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="audio.tts",
        error_type="NOT_IMPLEMENTED",
        message="audio tts is not implemented yet (see ticket T-08).",
        details={"text": text, "voice": voice, "out": out_path, "backend": backend},
    )
    _emit(out)


@audio_app.command("stt")
def audio_stt(
    in_path: str = typer.Option(..., "--in"),
    backend: str = typer.Option("whisper", "--backend", help="Audio backend id (see specs/audio-backends.md)"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="audio.stt",
        error_type="NOT_IMPLEMENTED",
        message="audio stt is not implemented yet (see ticket T-09).",
        details={"in": in_path, "backend": backend},
    )
    _emit(out)


@audio_app.command("assess")
def audio_assess(
    ref_text: str = typer.Option(..., "--ref-text"),
    in_path: str = typer.Option(..., "--in"),
    backend: str = typer.Option("local", "--backend", help="Audio backend id (see specs/audio-backends.md)"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="audio.assess",
        error_type="NOT_IMPLEMENTED",
        message="audio assess is not implemented yet (see ticket T-10).",
        details={"ref_text": ref_text, "in": in_path, "backend": backend},
    )
    _emit(out)


@audio_app.command("process-voice")
def audio_process_voice(
    in_path: str = typer.Option(..., "--in"),
    ref_text: str = typer.Option(..., "--ref-text"),
    backend: str = typer.Option("local", "--backend", help="Audio backend id (see specs/audio-backends.md)"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="audio.process-voice",
        error_type="NOT_IMPLEMENTED",
        message="audio process-voice is not implemented yet (see ticket T-10).",
        details={"in": in_path, "ref_text": ref_text, "backend": backend},
    )
    _emit(out)


# -------------- event --------------
@event_app.command("log")
def event_log(
    type: str = typer.Option(..., "--type", help="exposure|review|pronunciation_attempt|content_served"),
    modality: str = typer.Option(..., "--modality", help="reading|listening|speaking|typing|mixed"),
    items: str | None = typer.Option(None, "--items", help="Comma-separated item IDs (see specs/id-scheme.md)"),
    items_file: str | None = typer.Option(None, "--items-file", help="File with newline-delimited item IDs"),
    context: str | None = typer.Option(None, "--context"),
    json_output: bool = typer.Option(True, "--json"),
):
    parsed = events.parse_items(items=items, items_file=items_file)
    event = events.log_event(event_type=type, modality=modality, items=parsed, context=context)
    out = envelope.ok(
        command="event.log",
        data={
            "event_id": event.event_id,
            "event_type": event.event_type,
            "ts": event.ts,
            "modality": event.modality,
            "items": event.items,
            "context": event.context,
        },
    )
    _emit(out)


@event_app.command("list")
def event_list(
    since: str = typer.Option("7d", "--since", help="e.g. 7d, 24h, or an ISO timestamp"),
    limit: int = typer.Option(200, "--limit"),
    json_output: bool = typer.Option(True, "--json"),
):
    items = events.list_events(since=since, limit=limit)
    out = envelope.ok(
        command="event.list",
        data={
            "events": [
                {
                    "event_id": event.event_id,
                    "event_type": event.event_type,
                    "ts": event.ts,
                    "modality": event.modality,
                    "items": event.items,
                    "context": event.context,
                }
                for event in items
            ]
        },
        limits={"limit": limit, "since": since},
    )
    _emit(out)


# -------------- content --------------
cache_app = typer.Typer(add_completion=False)
content_app.add_typer(cache_app, name="cache")


@cache_app.command("put")
def cache_put(
    type: str = typer.Option(..., "--type", help="story|dialogue|exercise"),
    key: str = typer.Option(..., "--key"),
    in_path: str = typer.Option(..., "--in"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="content.cache.put",
        error_type="NOT_IMPLEMENTED",
        message="content cache put is not implemented yet (see ticket T-11).",
        details={"type": type, "key": key, "in": in_path},
    )
    _emit(out)


@cache_app.command("get")
def cache_get(
    type: str = typer.Option(..., "--type", help="story|dialogue|exercise"),
    key: str = typer.Option(..., "--key"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="content.cache.get",
        error_type="NOT_IMPLEMENTED",
        message="content cache get is not implemented yet (see ticket T-11).",
        details={"type": type, "key": key},
    )
    _emit(out)


if __name__ == "__main__":
    app()
