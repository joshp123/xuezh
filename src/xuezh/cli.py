from __future__ import annotations

import typer

from xuezh.core import envelope
from xuezh.core.jsonio import dumps

app = typer.Typer(add_completion=False, help="xuezh - local Chinese learning engine (ZFC/Unix-style)")

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
        typer.echo(dumps(envelope.ok(command="version", data={"version": "0.1.0"})))
    else:
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
    out = envelope.err(
        command="snapshot",
        error_type="NOT_IMPLEMENTED",
        message="snapshot is not implemented yet (see ticket T-05).",
        details={
            "window": window,
            "due_limit": due_limit,
            "evidence_limit": evidence_limit,
            "max_bytes": max_bytes,
        },
    )
    typer.echo(dumps(out))


@app.command()
def doctor(json_output: bool = typer.Option(True, "--json")):
    out = envelope.err(
        command="doctor",
        error_type="NOT_IMPLEMENTED",
        message="doctor is not implemented yet (see ticket T-14).",
        details={},
    )
    typer.echo(dumps(out))


@app.command()
def gc(
    apply: bool = typer.Option(False, "--apply", help="Actually delete files"),
    dry_run: bool = typer.Option(True, "--dry-run", help="Preview deletions (default)"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="gc",
        error_type="NOT_IMPLEMENTED",
        message="gc is not implemented yet (see ticket T-02B).",
        details={"apply": apply, "dry_run": dry_run},
    )
    typer.echo(dumps(out))


# ---------------- db ----------------
@db_app.command("init")
def db_init(json_output: bool = typer.Option(True, "--json")):
    out = envelope.err(
        command="db.init",
        error_type="NOT_IMPLEMENTED",
        message="db init is not implemented yet (see ticket T-03).",
        details={},
    )
    typer.echo(dumps(out))


# -------------- dataset --------------
@dataset_app.command("import")
def dataset_import(
    type: str = typer.Option(..., "--type", help="hsk_vocab|hsk_chars|hsk_grammar|frequency"),
    path: str = typer.Option(..., "--path"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="dataset.import",
        error_type="NOT_IMPLEMENTED",
        message="dataset import is not implemented yet (see ticket T-04).",
        details={"type": type, "path": path},
    )
    typer.echo(dumps(out))


# -------------- review --------------
@review_app.command("start")
def review_start(
    limit: int = typer.Option(10, "--limit"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="review.start",
        error_type="NOT_IMPLEMENTED",
        message="review start is not implemented yet (see ticket T-06).",
        details={"limit": limit},
    )
    typer.echo(dumps(out))


@review_app.command("grade")
def review_grade(
    item: str = typer.Option(..., "--item", help="Item ID (see specs/id-scheme.md)"),
    grade: int = typer.Option(..., "--grade", min=0, max=5),
    next_due: str | None = typer.Option(None, "--next-due"),
    rule: str | None = typer.Option(None, "--rule", help="sm2|leitner"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="review.grade",
        error_type="NOT_IMPLEMENTED",
        message="review grade is not implemented yet (see ticket T-06).",
        details={"item": item, "grade": grade, "next_due": next_due, "rule": rule},
    )
    typer.echo(dumps(out))


@review_app.command("bury")
def review_bury(
    item: str = typer.Option(..., "--item"),
    reason: str = typer.Option("unspecified", "--reason"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="review.bury",
        error_type="NOT_IMPLEMENTED",
        message="review bury is not implemented yet (see ticket T-06).",
        details={"item": item, "reason": reason},
    )
    typer.echo(dumps(out))


# -------------- srs --------------
@srs_app.command("preview")
def srs_preview(
    days: int = typer.Option(14, "--days"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="srs.preview",
        error_type="NOT_IMPLEMENTED",
        message="srs preview is not implemented yet (see ticket T-06).",
        details={"days": days},
    )
    typer.echo(dumps(out))


# -------------- report --------------
@report_app.command("hsk")
def report_hsk(
    level: int = typer.Option(..., "--level", min=1, max=6),
    window: str = typer.Option("30d", "--window"),
    max_items: int = typer.Option(200, "--max-items"),
    max_bytes: int = typer.Option(200_000, "--max-bytes"),
    include_chars: bool = typer.Option(False, "--include-chars", help="Optional: include character audit if dataset exists"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="report.hsk",
        error_type="NOT_IMPLEMENTED",
        message="report hsk is not implemented yet (see ticket T-07).",
        details={
            "level": level,
            "window": window,
            "max_items": max_items,
            "max_bytes": max_bytes,
            "include_chars": include_chars,
        },
    )
    typer.echo(dumps(out))


@report_app.command("mastery")
def report_mastery(
    item_type: str = typer.Option("word", "--item-type", help="word|character|grammar"),
    window: str = typer.Option("90d", "--window"),
    max_items: int = typer.Option(200, "--max-items"),
    max_bytes: int = typer.Option(200_000, "--max-bytes"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="report.mastery",
        error_type="NOT_IMPLEMENTED",
        message="report mastery is not implemented yet (see ticket T-07).",
        details={"item_type": item_type, "window": window, "max_items": max_items, "max_bytes": max_bytes},
    )
    typer.echo(dumps(out))


@report_app.command("due")
def report_due(
    limit: int = typer.Option(50, "--limit"),
    max_bytes: int = typer.Option(200_000, "--max-bytes"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="report.due",
        error_type="NOT_IMPLEMENTED",
        message="report due is not implemented yet (see ticket T-06/T-07).",
        details={"limit": limit, "max_bytes": max_bytes},
    )
    typer.echo(dumps(out))


# -------------- audio --------------
@audio_app.command("convert")
def audio_convert(
    in_path: str = typer.Option(..., "--in"),
    out_path: str = typer.Option(..., "--out"),
    format: str = typer.Option(..., "--format", help="wav|ogg|mp3"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="audio.convert",
        error_type="NOT_IMPLEMENTED",
        message="audio convert is not implemented yet (see ticket T-08).",
        details={"in": in_path, "out": out_path, "format": format},
    )
    typer.echo(dumps(out))


@audio_app.command("tts")
def audio_tts(
    text: str = typer.Option(..., "--text"),
    voice: str = typer.Option("XiaoxiaoNeural", "--voice"),
    out_path: str = typer.Option(..., "--out"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="audio.tts",
        error_type="NOT_IMPLEMENTED",
        message="audio tts is not implemented yet (see ticket T-08).",
        details={"text": text, "voice": voice, "out": out_path},
    )
    typer.echo(dumps(out))


@audio_app.command("stt")
def audio_stt(
    in_path: str = typer.Option(..., "--in"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="audio.stt",
        error_type="NOT_IMPLEMENTED",
        message="audio stt is not implemented yet (see ticket T-09).",
        details={"in": in_path},
    )
    typer.echo(dumps(out))


@audio_app.command("assess")
def audio_assess(
    ref_text: str = typer.Option(..., "--ref-text"),
    in_path: str = typer.Option(..., "--in"),
    mode: str = typer.Option("local", "--mode", help="local|azure"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="audio.assess",
        error_type="NOT_IMPLEMENTED",
        message="audio assess is not implemented yet (see ticket T-10).",
        details={"ref_text": ref_text, "in": in_path, "mode": mode},
    )
    typer.echo(dumps(out))


@audio_app.command("process-voice")
def audio_process_voice(
    in_path: str = typer.Option(..., "--in"),
    ref_text: str = typer.Option(..., "--ref-text"),
    mode: str = typer.Option("local", "--mode", help="local|azure"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="audio.process-voice",
        error_type="NOT_IMPLEMENTED",
        message="audio process-voice is not implemented yet (see ticket T-10).",
        details={"in": in_path, "ref_text": ref_text, "mode": mode},
    )
    typer.echo(dumps(out))


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
    out = envelope.err(
        command="event.log",
        error_type="NOT_IMPLEMENTED",
        message="event log is not implemented yet (see ticket T-06B).",
        details={"type": type, "modality": modality, "items": items, "items_file": items_file, "context": context},
    )
    typer.echo(dumps(out))


@event_app.command("list")
def event_list(
    since: str = typer.Option("7d", "--since", help="e.g. 7d, 24h, or an ISO timestamp"),
    limit: int = typer.Option(200, "--limit"),
    json_output: bool = typer.Option(True, "--json"),
):
    out = envelope.err(
        command="event.list",
        error_type="NOT_IMPLEMENTED",
        message="event list is not implemented yet (see ticket T-06B).",
        details={"since": since, "limit": limit},
    )
    typer.echo(dumps(out))


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
    typer.echo(dumps(out))


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
    typer.echo(dumps(out))


if __name__ == "__main__":
    app()
