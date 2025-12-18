from __future__ import annotations

import json
from typing import Any, Dict

def dumps(obj: Dict[str, Any]) -> str:
    return json.dumps(obj, ensure_ascii=False, indent=2, sort_keys=True) + "\n"
