#!/usr/bin/env python3
"""Safety regressions for the native development commands."""

import importlib.util
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('dev_server', ROOT / 'scripts/dev-server.py')
dev = importlib.util.module_from_spec(spec)
spec.loader.exec_module(dev)


class DevelopmentSafetyTests(unittest.TestCase):
    def test_stale_or_foreign_pid_is_never_stopped(self):
        with tempfile.TemporaryDirectory() as directory:
            state = Path(directory) / 'state.json'
            state.write_text(json.dumps({'pid': os.getpid(), 'start': 'stale'}))
            self.assertIsNone(dev.active_supervisor(state))
            state.write_text(json.dumps({'pid': os.getpid(), 'start': dev.process_start(os.getpid())}))
            # Even a matching start time does not authorize stopping an unrelated process.
            self.assertIsNone(dev.active_supervisor(state))
            for content in ['{broken', '{}', '{"pid":1,"start":"1"}', 'null']:
                state.write_text(content)
                self.assertIsNone(dev.active_supervisor(state))

    def test_occupied_port_is_reported_without_disrupting_listener(self):
        with socket.socket() as listener:
            listener.bind(('127.0.0.1', 0))
            listener.listen()
            port = listener.getsockname()[1]
            with self.assertRaisesRegex(RuntimeError, 'in use'):
                dev.check_ports([port])
            with socket.create_connection(('127.0.0.1', port), timeout=1):
                pass

    def test_shutdown_only_signals_owned_process_groups(self):
        owned = subprocess.Popen([sys.executable, '-c', 'import time; time.sleep(60)'], start_new_session=True)
        other = subprocess.Popen([sys.executable, '-c', 'import time; time.sleep(60)'], start_new_session=True)
        try:
            dev.stop_children([owned])
            self.assertIsNotNone(owned.poll())
            self.assertIsNone(other.poll())
        finally:
            other.terminate()
            other.wait(timeout=5)
            if owned.poll() is None:
                owned.kill()
                owned.wait()

    def test_backend_start_failure_removes_state_and_reports_failure(self):
        with tempfile.TemporaryDirectory() as directory:
            state = Path(directory) / 'dev-processes.json'
            with patch.object(dev, 'STATE', state), patch.object(dev, 'check_ports'), patch.object(
                dev.subprocess, 'Popen', side_effect=FileNotFoundError('missing watcher')
            ), patch.object(dev.signal, 'signal'):
                with self.assertRaises(FileNotFoundError):
                    dev.run()
                self.assertFalse(state.exists())

    def test_cache_reset_preserves_database_and_runtime_evidence(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ['pb_data/data.db', 'tmp/backend.log', 'tmp/dev-processes.json', 'uploads/file.txt']:
                target = root / name
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_text('preserve me')
            (root / 'frontend/.svelte-kit').mkdir(parents=True)
            result = subprocess.run(['make', '-f', str(ROOT / 'Makefile'), 'dev-reset'], cwd=root, capture_output=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            for name in ['pb_data/data.db', 'tmp/backend.log', 'tmp/dev-processes.json', 'uploads/file.txt']:
                self.assertEqual((root / name).read_text(), 'preserve me')
            self.assertFalse((root / 'frontend/.svelte-kit').exists())

    def test_make_lint_propagates_failure(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'frontend').mkdir()
            (root / 'bin').mkdir()
            npm = root / 'bin/npm'
            npm.write_text('#!/bin/sh\nexit 23\n')
            npm.chmod(0o755)
            result = subprocess.run(
                ['make', '-f', str(ROOT / 'Makefile'), 'lint-frontend'], cwd=root,
                env={**os.environ, 'PATH': f'{root / "bin"}:{os.environ["PATH"]}'}, capture_output=True,
            )
            self.assertNotEqual(result.returncode, 0)


if __name__ == '__main__':
    unittest.main()
