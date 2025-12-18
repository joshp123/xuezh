FORBIDDEN_KEYS = {
    "recommended_next",
    "priority_score",
    "should_learn",
    "next_best_action",
}


def assert_no_forbidden_keys(obj):
    if isinstance(obj, dict):
        for k, v in obj.items():
            assert k not in FORBIDDEN_KEYS, f"Forbidden key present: {k}"
            assert_no_forbidden_keys(v)
    elif isinstance(obj, list):
        for it in obj:
            assert_no_forbidden_keys(it)


def test_forbidden_keys_set_is_nonempty():
    assert len(FORBIDDEN_KEYS) > 0
