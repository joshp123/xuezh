from xuezh.core.envelope import ok, err


def test_ok_envelope_shape():
    out = ok(command="x", data={"a": 1})
    assert out["ok"] is True
    assert out["command"] == "x"
    assert "data" in out
    assert "artifacts" in out


def test_err_envelope_shape():
    out = err(command="x", error_type="E", message="m")
    assert out["ok"] is False
    assert out["error"]["type"] == "E"
