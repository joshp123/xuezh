from __future__ import annotations

from typing import Final

# The backend uses typed errors so tool callers can branch safely without parsing strings.
# Keep this list minimal and grow it only when a new type is actually emitted.
KNOWN_ERROR_TYPES: Final[set[str]] = {
    "BACKEND_FAILED",
    "INVALID_ARGUMENT",
    "NOT_IMPLEMENTED",
    "NOT_FOUND",
    "TOOL_MISSING",
}


def assert_known_error_type(error_type: str) -> None:
    if error_type not in KNOWN_ERROR_TYPES:
        raise ValueError(f"Unknown error type: {error_type!r}. Add it to xuezh.core.error_types.KNOWN_ERROR_TYPES.")
