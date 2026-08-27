"""Stable import bridge for scripts whose directory name contains a hyphen."""

import importlib.util
from pathlib import Path

_path = Path(__file__).with_name("creator-signal") / "release_control.py"
_spec = importlib.util.spec_from_file_location("creator_signal_release_control", _path)
if _spec is None or _spec.loader is None:
    raise RuntimeError("unable to load release control")
release_control = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(release_control)
