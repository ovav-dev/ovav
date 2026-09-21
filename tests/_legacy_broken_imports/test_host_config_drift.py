import json

from tools.validators.check_host_config_drift import _is_benign_opencode_bootstrap_config


def test_schema_only_opencode_jsonc_is_benign(tmp_path):
    config = tmp_path / "opencode.jsonc"
    config.write_text(json.dumps({"$schema": "https://opencode.ai/config.json"}), encoding="utf-8")

    assert _is_benign_opencode_bootstrap_config(config) is True


def test_global_model_override_is_not_benign(tmp_path):
    config = tmp_path / "opencode.jsonc"
    config.write_text(
        json.dumps({"$schema": "https://opencode.ai/config.json", "model": "openai/example"}),
        encoding="utf-8",
    )

    assert _is_benign_opencode_bootstrap_config(config) is False


def test_global_agent_file_is_not_benign(tmp_path):
    config = tmp_path / "AGENTS.md"
    config.write_text("external instructions", encoding="utf-8")

    assert _is_benign_opencode_bootstrap_config(config) is False
