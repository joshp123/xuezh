from __future__ import annotations

from tests.contract._helpers import cli_to_command_id, extract_bdd_commands, load_contract


def test_all_bdd_commands_are_declared_in_contract() -> None:
    contract = load_contract()
    contract_ids = {c["id"] for c in contract["commands"]}

    for cli in extract_bdd_commands():
        cid = cli_to_command_id(cli)
        assert cid in contract_ids, f"BDD uses command not declared in contract.json: {cli} -> {cid}"
