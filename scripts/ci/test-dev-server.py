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

    def test_port_preflight_defaults_empty_host_to_loopback(self):
        with patch.object(dev.socket, 'socket') as socket_factory:
            dev.check_ports([8090], host='')
            socket_factory.return_value.__enter__.return_value.bind.assert_called_once_with(
                ('127.0.0.1', 8090)
            )

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

    def test_dev_host_defaults_to_loopback_and_accepts_plain_hosts(self):
        with patch.dict(os.environ, {}, clear=False):
            os.environ.pop('FACET_DEV_HOST', None)
            self.assertEqual(dev.dev_host(), '127.0.0.1')
        self.assertEqual(dev.dev_host(''), '127.0.0.1')
        for host in ['127.0.0.1', '0.0.0.0', 'localhost', 'dev-box.local', '192.168.1.20']:
            self.assertEqual(dev.dev_host(host), host)

    def test_dev_host_rejects_shell_and_air_syntax(self):
        # Air hands this value to a shell, so metacharacters and malformed names must never pass.
        for host in [
            '127.0.0.1;touch /tmp/pwned', '$(id)', '`id`', '${HOME}', 'a b', 'host\nname',
            '-h', '.local', 'bad..name', 'trailing-.local', '999.1.1.1', '::1', '[::1]',
            'host:8090', 'x' * 64, "'quoted'",
        ]:
            with self.assertRaisesRegex(RuntimeError, 'FACET_DEV_HOST', msg=host):
                dev.dev_host(host)

    def test_shell_scripts_reject_unsafe_dev_host_before_starting(self):
        marker_dir = tempfile.TemporaryDirectory()
        self.addCleanup(marker_dir.cleanup)
        marker = Path(marker_dir.name) / 'pwned'
        for script in ['scripts/dev-frontend.sh', 'scripts/dev-backend.sh']:
            with tempfile.TemporaryDirectory() as directory:
                bin_dir = Path(directory)
                # Stubs record any attempt to start tools after validation should have stopped the script.
                for tool in ['npm', 'npx', 'air', 'go']:
                    stub = bin_dir / tool
                    stub.write_text(f'#!/bin/sh\necho {tool} >> {marker}\n')
                    stub.chmod(0o755)
                env = {
                    **os.environ, 'PATH': f'{bin_dir}:{os.environ["PATH"]}',
                    'FACET_DEV_HOST': f'0.0.0.0;touch {marker}', 'DATA_DIR': directory,
                }
                result = subprocess.run(
                    ['sh' if script.endswith('frontend.sh') else 'bash', str(ROOT / script)],
                    cwd=ROOT, env=env, capture_output=True, text=True, timeout=30,
                )
                self.assertNotEqual(result.returncode, 0, script)
                self.assertIn('FACET_DEV_HOST', result.stdout + result.stderr, script)
                self.assertFalse(marker.exists(), f'{script} ran a tool or injected command')

    def test_wildcard_host_is_browsed_on_loopback(self):
        self.assertEqual(dev.browse_host('0.0.0.0'), '127.0.0.1')
        self.assertEqual(dev.browse_host('localhost'), 'localhost')

    def test_port_preflight_uses_requested_host(self):
        with socket.socket() as listener:
            listener.bind(('127.0.0.1', 0))
            listener.listen()
            port = listener.getsockname()[1]
            # A wildcard bind collides with a loopback listener, so 0.0.0.0 still reports it.
            with self.assertRaisesRegex(RuntimeError, 'in use'):
                dev.check_ports([port], host='0.0.0.0')

    def test_invalid_host_stops_before_lock_state_or_children(self):
        with tempfile.TemporaryDirectory() as directory:
            state = Path(directory) / 'dev-processes.json'
            with patch.dict(os.environ, {'FACET_DEV_HOST': '$(id)'}), patch.object(dev, 'STATE', state), \
                    patch.object(dev.subprocess, 'Popen') as popen:
                with self.assertRaisesRegex(RuntimeError, 'FACET_DEV_HOST'):
                    dev.run()
                popen.assert_not_called()
                self.assertFalse(state.exists())

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
