#!/usr/bin/env python3
"""Tests for docker/start.sh using stub processes.

Runs the real start script against temporary data/upload directories with
fake facet, node, caddy, wget and openssl commands, so no Docker daemon or
real service is needed. Each stub records its PID so the tests can check
that the script cleaned up exactly the children it started.

Run: python3 -I scripts/ci/test-start-sh.py
"""

import os
import signal
import stat
import subprocess
import sys
import tempfile
import time
import unittest

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
START_SH = os.path.join(ROOT, "docker", "start.sh")
STUB_KEY = "ab" * 32

# Each process stub writes its PID, then behaves according to an env var:
#   healthy      mark itself ready and run until signalled
#   never        run but never become ready
#   crash        exit 3 straight away
#   ignore_term  mark ready and ignore SIGTERM (needs SIGKILL)
#   exit_later   mark ready, then exit 7 after one second
PROCESS_STUB = """#!/bin/bash
name={name}
mode="${{STUB_{upper}_MODE:-healthy}}"
echo $$ > "$STUB_STATE/$name.pid"
case "$mode" in
    crash) exit 3 ;;
    never) ;;
    ignore_term) trap '' TERM; touch "$STUB_STATE/$name.ready" ;;
    exit_later) touch "$STUB_STATE/$name.ready"; sleep 1; exit 7 ;;
    *) touch "$STUB_STATE/$name.ready" ;;
esac
while :; do sleep 0.2; done
"""

# wget succeeds once the stub behind that port has marked itself ready.
WGET_STUB = """#!/bin/bash
url="${@: -1}"
case "$url" in
    *:8090/*) [ -e "$STUB_STATE/facet.ready" ] ;;
    *:8080/*) [ -e "$STUB_STATE/caddy.ready" ] && [ -e "$STUB_STATE/node.ready" ] ;;
    *) exit 1 ;;
esac
"""

OPENSSL_STUB = """#!/bin/bash
echo "{key}"
""".format(key=STUB_KEY)

ROOT_ID_STUB = """#!/bin/bash
if [ "${FACET_TEST_NONROOT:-}" = "1" ]; then echo 1000; else echo 0; fi
"""

ACCOUNT_COMMAND_STUB = """#!/bin/bash
printf '%s\\n' "$*" >> "$STUB_STATE/{name}.log"
"""

GOSU_STUB = """#!/bin/bash
shift
export FACET_TEST_NONROOT=1
exec bash "$@"
"""


def write_exec(path, body):
    with open(path, "w") as f:
        f.write(body)
    os.chmod(path, 0o755)


def pid_alive(pid):
    try:
        with open(f"/proc/{pid}/stat") as f:
            state = f.read().rsplit(")", 1)[1].split()[0]
    except FileNotFoundError:
        return False
    return state not in ("Z", "X")


class StartScriptTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="facet-start-sh-")
        base = self.tmp.name
        self.bin = os.path.join(base, "bin")
        self.work = os.path.join(base, "app")
        self.state = os.path.join(base, "state")
        self.data = os.path.join(base, "data")
        self.uploads = os.path.join(base, "uploads")
        for d in (self.bin, self.work, self.state, self.data, self.uploads,
                  os.path.join(self.work, "frontend")):
            os.makedirs(d)
        open(os.path.join(self.work, "Caddyfile"), "w").close()

        write_exec(os.path.join(self.work, "facet"), PROCESS_STUB.format(name="facet", upper="FACET"))
        write_exec(os.path.join(self.bin, "node"), PROCESS_STUB.format(name="node", upper="NODE"))
        write_exec(os.path.join(self.bin, "caddy"), PROCESS_STUB.format(name="caddy", upper="CADDY"))
        write_exec(os.path.join(self.bin, "wget"), WGET_STUB)
        write_exec(os.path.join(self.bin, "openssl"), OPENSSL_STUB)

    def tearDown(self):
        # Belt and braces: never leave a stub running if a test failed midway.
        for name in ("facet", "node", "caddy"):
            pid = self.pid(name)
            if pid and pid_alive(pid):
                os.kill(pid, signal.SIGKILL)
        self.tmp.cleanup()

    def env(self, **modes):
        env = {k: v for k, v in os.environ.items() if k != "ENCRYPTION_KEY"}
        env.update({
            "PATH": self.bin + os.pathsep + env.get("PATH", ""),
            "STUB_STATE": self.state,
            "FACET_DATA_DIR": self.data,
            "FACET_UPLOADS_DIR": self.uploads,
            "FACET_HEALTH_TIMEOUT": "3",
            "FACET_SHUTDOWN_TIMEOUT": "2",
        })
        for name, mode in modes.items():
            env[f"STUB_{name.upper()}_MODE"] = mode
        return env

    def pid(self, name):
        try:
            with open(os.path.join(self.state, f"{name}.pid")) as f:
                return int(f.read().strip())
        except FileNotFoundError:
            return None

    def run_start(self, timeout=30, **modes):
        start = time.monotonic()
        proc = subprocess.run(["bash", START_SH], cwd=self.work, env=self.env(**modes),
                              capture_output=True, text=True, timeout=timeout)
        return proc, time.monotonic() - start

    def run_root_start(self, puid="1000", pgid="1000"):
        write_exec(os.path.join(self.bin, "id"), ROOT_ID_STUB)
        write_exec(os.path.join(self.bin, "gosu"), GOSU_STUB)
        for name in ("groupmod", "usermod", "chown"):
            write_exec(os.path.join(self.bin, name), ACCOUNT_COMMAND_STUB.format(name=name))
        env = self.env(facet="crash")
        env.update(PUID=puid, PGID=pgid)
        return subprocess.run(["bash", START_SH], cwd=self.work, env=env,
                              capture_output=True, text=True, timeout=30)

    def command_log(self, name):
        with open(os.path.join(self.state, f"{name}.log")) as f:
            return f.read().splitlines()

    def test_root_setup_owns_runtime_dirs_without_recursive_app_chown(self):
        proc = self.run_root_start()
        self.assertEqual(proc.returncode, 1, proc.stdout + proc.stderr)
        self.assertEqual(self.command_log("chown"), [
            f"-R facet:facet {self.data} {self.uploads} /home/facet",
        ])

    def test_root_setup_applies_nondefault_ids_to_runtime_dirs(self):
        proc = self.run_root_start(puid="1234", pgid="5678")
        self.assertEqual(proc.returncode, 1, proc.stdout + proc.stderr)
        self.assertEqual(self.command_log("groupmod"), ["-o -g 5678 facet"])
        self.assertEqual(self.command_log("usermod"), ["-o -u 1234 -g facet facet"])
        self.assertEqual(self.command_log("chown"), [
            f"-R facet:facet {self.data} {self.uploads} /home/facet",
        ])

    def assert_stopped(self, *names):
        for name in names:
            pid = self.pid(name)
            self.assertIsNotNone(pid, f"{name} never started")
            self.assertFalse(pid_alive(pid), f"{name} (pid {pid}) still running")

    def test_backend_never_healthy_fails_and_cleans_up(self):
        proc, elapsed = self.run_start(facet="never")
        out = proc.stdout + proc.stderr
        self.assertEqual(proc.returncode, 1, out)
        self.assertIn("not healthy after 3s", out)
        self.assertNotIn("Facet is running", out)
        self.assertIsNone(self.pid("node"), "frontend started after a failed health check")
        self.assertIsNone(self.pid("caddy"), "proxy started after a failed health check")
        self.assert_stopped("facet")
        self.assertLess(elapsed, 3 + 2 + 5)

    def test_backend_crash_fails_fast(self):
        proc, elapsed = self.run_start(facet="crash")
        out = proc.stdout + proc.stderr
        self.assertEqual(proc.returncode, 1, out)
        self.assertIn("exited before becoming healthy", out)
        self.assertLess(elapsed, 3, "should not wait out the full health timeout")

    def test_proxy_never_healthy_stops_all_children(self):
        proc, _ = self.run_start(caddy="never")
        out = proc.stdout + proc.stderr
        self.assertEqual(proc.returncode, 1, out)
        self.assertNotIn("Facet is running", out)
        self.assert_stopped("facet", "node", "caddy")

    def test_child_exit_stops_the_rest_with_its_status(self):
        proc, _ = self.run_start(node="exit_later")
        out = proc.stdout + proc.stderr
        self.assertIn("Facet is running", out)
        self.assertEqual(proc.returncode, 7, out)
        self.assert_stopped("facet", "node", "caddy")

    def test_sigterm_stops_children_within_bound(self):
        # The backend ignores SIGTERM, so the script must escalate to SIGKILL.
        p = subprocess.Popen(["bash", START_SH], cwd=self.work, env=self.env(facet="ignore_term"),
                             stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        try:
            deadline = time.monotonic() + 15
            while not (self.pid("caddy") and os.path.exists(os.path.join(self.state, "caddy.ready"))):
                self.assertLess(time.monotonic(), deadline, "stack never came up")
                time.sleep(0.1)
            time.sleep(1.5)  # let the script reach `wait -n`
            sent = time.monotonic()
            p.send_signal(signal.SIGTERM)
            out, _ = p.communicate(timeout=15)
            elapsed = time.monotonic() - sent
        finally:
            if p.poll() is None:
                p.kill()
        self.assertEqual(p.returncode, 143, out)
        self.assertIn("Forcing stop after 2s", out)
        self.assert_stopped("facet", "node", "caddy")
        self.assertLess(elapsed, 2 + 4)

    def test_generated_key_saved_private_and_not_logged(self):
        proc, _ = self.run_start(facet="crash")
        out = proc.stdout + proc.stderr
        key_file = os.path.join(self.data, ".encryption_key")
        with open(key_file) as f:
            self.assertEqual(f.read().strip(), STUB_KEY)
        self.assertEqual(stat.S_IMODE(os.stat(key_file).st_mode), 0o600)
        self.assertNotIn(STUB_KEY, out)
        self.assertIn("ENCRYPTION KEY GENERATED", out)

    def test_existing_key_preserved_and_not_logged(self):
        key_file = os.path.join(self.data, ".encryption_key")
        existing = "cd" * 32 + "\n"
        with open(key_file, "w") as f:
            f.write(existing)
        os.chmod(key_file, 0o600)
        proc, _ = self.run_start(facet="crash")
        out = proc.stdout + proc.stderr
        with open(key_file) as f:
            self.assertEqual(f.read(), existing)
        self.assertEqual(stat.S_IMODE(os.stat(key_file).st_mode), 0o600)
        self.assertNotIn(existing.strip(), out)
        self.assertIn("Loaded ENCRYPTION_KEY", out)

    def test_empty_key_file_is_not_replaced(self):
        key_file = os.path.join(self.data, ".encryption_key")
        open(key_file, "w").close()
        proc, _ = self.run_start()
        self.assertEqual(proc.returncode, 1, proc.stdout + proc.stderr)
        self.assertIn("exists but is empty", proc.stderr)
        self.assertEqual(os.path.getsize(key_file), 0)
        self.assertIsNone(self.pid("facet"), "backend started without a key")


if __name__ == "__main__":
    unittest.main(verbosity=2)
