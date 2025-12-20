from __future__ import annotations

import os
import shutil
import sqlite3
from pathlib import Path
import typer

from xuezh.core import (
    audio,
    clock,
    config as config_core,
    content,
    datasets,
    db,
    envelope,
    events,
    paths,
    retention,
    reports,
    snapshot as snapshot_core,
    srs,
)
from xuezh.core.jsonio import dumps
from xuezh.core.audio import AzureSpeechError
from xuezh.core.process import ProcessFailedError, ToolMissingError

app = typer.Typer(add_completion=False, help="xuezh - local Chinese learning engine (ZFC/Unix-style)")


def _emit(out: dict) -> None:
    typer.echo(dumps(out))
    if out.get("ok") is True:
        raise typer.Exit(code=0)
    raise typer.Exit(code=1)


def _trim(text: str, limit: int = 2000) -> str:
    return text if len(text) <= limit else text[:limit]


def _resolve_audio_backend(
    *,
    cli_value: str | None,
    default: str,
    env_key: str,
    config_key: str | None = None,
) -> str:
    if cli_value:
        return cli_value
    if config_key:
        config_value = config_core.get_config_value("audio", config_key)
        if isinstance(config_value, str) and config_value.strip():
            return config_value
    config_global = config_core.get_config_value("audio", "backend_global")
    if isinstance(config_global, str) and config_global.strip():
        return config_global
    env_value = os.environ.get(env_key) or os.environ.get("XUEZH_AUDIO_BACKEND")
    return env_value or default


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
    workspace = paths.workspace_dir()
    db_path = paths.db_path()

    checks: list[dict] = []

    checks.append(
        {
            "name": "workspace.path",
            "ok": True,
            "details": {
                "path": str(workspace),
                "exists": workspace.exists(),
                "override": os.environ.get("XUEZH_WORKSPACE_DIR"),
            },
        }
    )

    db_exists = db_path.exists()
    db_details = {"path": str(db_path), "exists": db_exists, "override": os.environ.get("XUEZH_DB_PATH")}
    if db_exists:
        try:
            conn = sqlite3.connect(db_path)
            try:
                row = conn.execute("SELECT COUNT(*) FROM schema_migrations").fetchone()
                db_details["schema_migrations"] = int(row[0]) if row else 0
            finally:
                conn.close()
        except sqlite3.Error as exc:
            db_details["error"] = str(exc)
            checks.append({"name": "db.status", "ok": False, "details": db_details})
        else:
            checks.append({"name": "db.status", "ok": True, "details": db_details})
    else:
        checks.append({"name": "db.status", "ok": False, "details": db_details})

    for tool in ("ffmpeg", "edge-tts", "whisper"):
        path = shutil.which(tool)
        checks.append(
            {
                "name": f"tool.{tool}",
                "ok": path is not None,
                "details": {"path": path},
            }
        )

    try:
        import azure.cognitiveservices.speech as speechsdk  # type: ignore
        checks.append({"name": "tool.azure-speech-sdk", "ok": True, "details": {"version": speechsdk.__version__}})
    except Exception as exc:
        checks.append({"name": "tool.azure-speech-sdk", "ok": False, "details": {"error": str(exc)}})

    config_section = config_core.get_config_value("azure", "speech")
    config_key = None
    config_region = None
    config_key_file = None
    if isinstance(config_section, dict):
        config_key = config_section.get("key")
        config_key_file = config_section.get("key_file")
        config_region = config_section.get("region")
    config_key_present = bool(config_key)
    if config_key_file:
        try:
            config_key_present = bool(Path(str(config_key_file)).expanduser().read_text(encoding="utf-8").strip())
        except OSError:
            config_key_present = False

    env_key_present = bool(os.environ.get("AZURE_SPEECH_KEY"))
    env_region_present = bool(os.environ.get("AZURE_SPEECH_REGION"))
    config_region_present = bool(config_region)

    checks.append(
        {
            "name": "azure.speech.env",
            "ok": bool((env_key_present or config_key_present) and (env_region_present or config_region_present)),
            "details": {
                "AZURE_SPEECH_KEY": env_key_present,
                "AZURE_SPEECH_REGION": env_region_present,
                "config_key": config_key_present,
                "config_region": config_region_present,
                "config_path": str(config_core.config_path()),
            },
        }
    )

    out = envelope.ok(command="doctor", data={"checks": checks})
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
    recall_items = srs.list_due_items(limit=limit, now=now, review_type="recall")
    pronunciation_items = srs.list_due_items(limit=limit, now=now, review_type="pronunciation")
    out = envelope.ok(
        command="review.start",
        data={
            "items": [
                {"item_id": item.item_id, "due_at": item.due_at, "review_type": item.review_type}
                for item in recall_items
            ],
            "recall_items": [
                {"item_id": item.item_id, "due_at": item.due_at, "review_type": item.review_type}
                for item in recall_items
            ],
            "pronunciation_items": [
                {"item_id": item.item_id, "due_at": item.due_at, "review_type": item.review_type}
                for item in pronunciation_items
            ],
            "generated_at": now.isoformat(),
        },
        limits={"limit": limit},
    )
    _emit(out)


@review_app.command("grade")
def review_grade(
    item: str = typer.Option(..., "--item", help="Item ID (see specs/id-scheme.md)"),
    grade: int | None = typer.Option(None, "--grade", min=0, max=5),
    recall: int | None = typer.Option(None, "--recall", min=0, max=5),
    pronunciation: int | None = typer.Option(None, "--pronunciation", min=0, max=5),
    next_due: str | None = typer.Option(None, "--next-due"),
    rule: str | None = typer.Option(None, "--rule", help="sm2|leitner"),
    json_output: bool = typer.Option(True, "--json"),
):
    now = clock.now_utc()
    if grade is not None and (recall is not None or pronunciation is not None):
        raise ValueError("Use --grade alone or --recall/--pronunciation, not both")
    if grade is None and recall is None and pronunciation is None:
        raise ValueError("Provide --grade or --recall/--pronunciation")

    if grade is not None:
        recall = grade

    recall_due_at = None
    recall_rule = None
    if recall is not None:
        recall_due_at, recall_rule = srs.schedule_next_due(
            grade=recall,
            now=now,
            rule=rule,
            next_due=next_due,
        )
    pronunciation_due_at = None
    pronunciation_rule = None
    if pronunciation is not None:
        pronunciation_next_due = next_due if recall is None else None
        pronunciation_due_at, pronunciation_rule = srs.schedule_next_due(
            grade=pronunciation,
            now=now,
            rule=rule,
            next_due=pronunciation_next_due,
        )

    srs.upsert_knowledge(
        item_id=item,
        recall_due_at=recall_due_at,
        recall_grade=recall,
        pronunciation_due_at=pronunciation_due_at,
        pronunciation_grade=pronunciation,
        now=now,
    )
    if recall is not None:
        srs.record_review_event(
            item_id=item,
            event_type="review.grade",
            payload={"review_type": "recall", "grade": recall, "rule": recall_rule, "next_due": recall_due_at},
            now=now,
        )
    if pronunciation is not None:
        srs.record_review_event(
            item_id=item,
            event_type="review.grade",
            payload={
                "review_type": "pronunciation",
                "grade": pronunciation,
                "rule": pronunciation_rule,
                "next_due": pronunciation_due_at,
            },
            now=now,
        )

    data: dict[str, object] = {"item": item}
    if recall is not None:
        data.update(
            {
                "recall_grade": recall,
                "recall_next_due": recall_due_at,
                "recall_rule_applied": recall_rule,
            }
        )
    if pronunciation is not None:
        data.update(
            {
                "pronunciation_grade": pronunciation,
                "pronunciation_next_due": pronunciation_due_at,
                "pronunciation_rule_applied": pronunciation_rule,
            }
        )
    if grade is not None:
        data.update({"grade": grade, "next_due": recall_due_at, "rule_applied": recall_rule})

    out = envelope.ok(command="review.grade", data=data)
    _emit(out)


@review_app.command("bury")
def review_bury(
    item: str = typer.Option(..., "--item"),
    reason: str = typer.Option("unspecified", "--reason"),
    json_output: bool = typer.Option(True, "--json"),
):
    now = clock.now_utc()
    due_at, _ = srs.schedule_next_due(grade=0, now=now, rule="leitner", next_due=None)
    srs.upsert_knowledge(item_id=item, recall_due_at=due_at, recall_grade=None, now=now)
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
    forecast = {
        "recall": srs.preview_due(days=days, now=now, review_type="recall"),
        "pronunciation": srs.preview_due(days=days, now=now, review_type="pronunciation"),
    }
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
    items = srs.list_due_items(limit=limit, now=now, review_type="recall")
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
    backend: str | None = typer.Option(None, "--backend", help="Audio backend id (see specs/audio-backends.md)"),
    json_output: bool = typer.Option(True, "--json"),
):
    backend = _resolve_audio_backend(
        cli_value=backend,
        default="ffmpeg",
        env_key="XUEZH_AUDIO_CONVERT_BACKEND",
        config_key="convert_backend",
    )
    try:
        result = audio.convert_audio(in_path=in_path, out_path=out_path, fmt=format, backend=backend)
        out = envelope.ok(command="audio.convert", data=result.data, artifacts=result.artifacts)
    except ToolMissingError as exc:
        out = envelope.err(
            command="audio.convert",
            error_type="TOOL_MISSING",
            message=str(exc),
            details={"tool": exc.tool, "in": in_path, "out": out_path, "format": format, "backend": backend},
        )
    except ProcessFailedError as exc:
        out = envelope.err(
            command="audio.convert",
            error_type="BACKEND_FAILED",
            message="audio backend failed during conversion",
            details={
                "cmd": exc.cmd,
                "returncode": exc.returncode,
                "stderr": _trim(exc.stderr),
                "in": in_path,
                "out": out_path,
                "format": format,
                "backend": backend,
            },
        )
    except (ValueError, FileNotFoundError) as exc:
        out = envelope.err(
            command="audio.convert",
            error_type="INVALID_ARGUMENT",
            message=str(exc),
            details={"in": in_path, "out": out_path, "format": format, "backend": backend},
        )
    _emit(out)


@audio_app.command("tts")
def audio_tts(
    text: str = typer.Option(..., "--text"),
    voice: str = typer.Option("XiaoxiaoNeural", "--voice"),
    out_path: str = typer.Option(..., "--out"),
    backend: str | None = typer.Option(None, "--backend", help="Audio backend id (see specs/audio-backends.md)"),
    json_output: bool = typer.Option(True, "--json"),
):
    backend = _resolve_audio_backend(
        cli_value=backend,
        default="edge-tts",
        env_key="XUEZH_AUDIO_TTS_BACKEND",
        config_key="tts_backend",
    )
    try:
        result = audio.tts_audio(text=text, voice=voice, out_path=out_path, backend=backend)
        out = envelope.ok(command="audio.tts", data=result.data, artifacts=result.artifacts)
    except ToolMissingError as exc:
        out = envelope.err(
            command="audio.tts",
            error_type="TOOL_MISSING",
            message=str(exc),
            details={"tool": exc.tool, "text": text, "voice": voice, "out": out_path, "backend": backend},
        )
    except ProcessFailedError as exc:
        out = envelope.err(
            command="audio.tts",
            error_type="BACKEND_FAILED",
            message="audio backend failed during tts",
            details={
                "cmd": exc.cmd,
                "returncode": exc.returncode,
                "stderr": _trim(exc.stderr),
                "text": text,
                "voice": voice,
                "out": out_path,
                "backend": backend,
            },
        )
    except ValueError as exc:
        out = envelope.err(
            command="audio.tts",
            error_type="INVALID_ARGUMENT",
            message=str(exc),
            details={"text": text, "voice": voice, "out": out_path, "backend": backend},
        )
    _emit(out)


@audio_app.command("process-voice")
def audio_process_voice(
    in_path: str = typer.Option(..., "--in"),
    ref_text: str = typer.Option(..., "--ref-text"),
    json_output: bool = typer.Option(True, "--json"),
):
    backend = _resolve_audio_backend(
        cli_value=None,
        default="azure.speech",
        env_key="XUEZH_AUDIO_PROCESS_VOICE_BACKEND",
        config_key="process_voice_backend",
    )
    try:
        result = audio.process_voice(in_path=in_path, ref_text=ref_text, backend=backend)
        out = envelope.ok(
            command="audio.process-voice",
            data=result.data,
            artifacts=result.artifacts,
            truncated=result.truncated,
            limits=result.limits,
        )
    except AzureSpeechError as exc:
        error_type = "BACKEND_FAILED"
        if exc.kind == "quota":
            error_type = "QUOTA_EXCEEDED"
        elif exc.kind == "auth":
            error_type = "AUTH_FAILED"
        out = envelope.err(
            command="audio.process-voice",
            error_type=error_type,
            message=str(exc),
            details={"ref_text": ref_text, "in": in_path, "backend": backend, **exc.details},
        )
    except ToolMissingError as exc:
        out = envelope.err(
            command="audio.process-voice",
            error_type="TOOL_MISSING",
            message=str(exc),
            details={"tool": exc.tool, "ref_text": ref_text, "in": in_path, "backend": backend},
        )
    except ProcessFailedError as exc:
        out = envelope.err(
            command="audio.process-voice",
            error_type="BACKEND_FAILED",
            message="audio backend failed during voice processing",
            details={
                "cmd": exc.cmd,
                "returncode": exc.returncode,
                "stderr": _trim(exc.stderr),
                "ref_text": ref_text,
                "in": in_path,
                "backend": backend,
            },
        )
    except (ValueError, FileNotFoundError) as exc:
        out = envelope.err(
            command="audio.process-voice",
            error_type="INVALID_ARGUMENT",
            message=str(exc),
            details={"ref_text": ref_text, "in": in_path, "backend": backend},
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
    try:
        result = content.put_content(content_type=type, key=key, in_path=in_path)
        out = envelope.ok(command="content.cache.put", data=result.data, artifacts=result.artifacts)
    except FileNotFoundError as exc:
        out = envelope.err(
            command="content.cache.put",
            error_type="INVALID_ARGUMENT",
            message=str(exc),
            details={"type": type, "key": key, "in": in_path},
        )
    except ValueError as exc:
        out = envelope.err(
            command="content.cache.put",
            error_type="INVALID_ARGUMENT",
            message=str(exc),
            details={"type": type, "key": key, "in": in_path},
        )
    _emit(out)


@cache_app.command("get")
def cache_get(
    type: str = typer.Option(..., "--type", help="story|dialogue|exercise"),
    key: str = typer.Option(..., "--key"),
    json_output: bool = typer.Option(True, "--json"),
):
    try:
        result = content.get_content(content_type=type, key=key)
        out = envelope.ok(command="content.cache.get", data=result.data, artifacts=result.artifacts)
    except FileNotFoundError as exc:
        out = envelope.err(
            command="content.cache.get",
            error_type="NOT_FOUND",
            message=str(exc),
            details={"type": type, "key": key},
        )
    except ValueError as exc:
        out = envelope.err(
            command="content.cache.get",
            error_type="INVALID_ARGUMENT",
            message=str(exc),
            details={"type": type, "key": key},
        )
    _emit(out)


if __name__ == "__main__":
    app()
