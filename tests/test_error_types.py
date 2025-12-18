from __future__ import annotations

import pytest

from xuezh.core import envelope


def test_envelope_err_rejects_unknown_error_type() -> None:
    with pytest.raises(ValueError, match="Unknown error type"):
        envelope.err(command="snapshot", error_type="BOGUS", message="nope", details={})
