from __future__ import annotations

from tests.contract._helpers import (
    cli_to_command_id,
    extract_bdd_commands,
    load_contract,
    parse_ticket_frontmatter,
    repo_root,
)


def test_every_contract_command_has_schema_bdd_and_ticket_mapping() -> None:
    root = repo_root()
    contract = load_contract()

    # Collect command IDs from contract
    contract_cmd_ids = [c["id"] for c in contract["commands"]]
    assert len(contract_cmd_ids) == len(set(contract_cmd_ids)), "Duplicate command ids in contract.json"

    # Collect command IDs referenced in BDD specs
    bdd_cmd_ids = {cli_to_command_id(c) for c in extract_bdd_commands()}

    for cmd in contract["commands"]:
        cid = cmd["id"]

        # 1) Schema exists for every command (by convention + contract field)
        schema_path = root / cmd.get("schema_ok", f"schemas/{cid}.schema.json")
        assert schema_path.exists(), f"Missing schema for command {cid}: {schema_path}"

        # 2) BDD scenario exists (command is referenced at least once in feature files)
        assert cid in bdd_cmd_ids, f"Command {cid} is not referenced by any BDD feature scenario"

        # 3) Ticket mapping exists and is consistent
        ticket_id = cmd.get("ticket")
        assert ticket_id, f"Command {cid} is missing ticket mapping in contract.json"

        ticket_path = root / "tickets" / f"{ticket_id}.md"
        assert ticket_path.exists(), f"Mapped ticket missing for command {cid}: {ticket_id}"

        fm = parse_ticket_frontmatter(ticket_path)
        impl = fm.get("implements_commands")
        assert isinstance(impl, list), f"Ticket {ticket_id} missing implements_commands list"
        assert cid in impl, f"Ticket {ticket_id} does not declare it implements {cid} (implements_commands={impl})"


def test_no_orphan_command_schemas() -> None:
    """Ensure we don't accumulate schema files for commands that aren't in the contract."""
    root = repo_root()
    contract = load_contract()
    cmd_ids = {c["id"] for c in contract["commands"]}

    schema_dir = root / "schemas"
    for p in schema_dir.glob("*.schema.json"):
        name = p.name.replace(".schema.json", "")
        if name.startswith("envelope."):
            continue
        assert name in cmd_ids, f"Orphan command schema (not in contract): {p}"
