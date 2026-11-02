#!/usr/bin/env python3
"""Supervise native development services without touching other checkouts."""

import argparse
import fcntl
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parent.parent
STATE = ROOT / 'tmp' / 'dev-processes.json'


def process_start(pid):
    """Linux start ticks distinguish a saved process from a reused PID."""
    try:
        stat = Path(f'/proc/{pid}/stat').read_text()
        return stat[stat.rfind(')') + 2:].split()[19]
    except (OSError, IndexError):
        return None


def active_supervisor(state_path=STATE):
    try:
        state = json.loads(state_path.read_text())
        pid = state['pid']
        if not isinstance(pid, int) or pid <= 1 or not state.get('start'):
            return None
        if process_start(pid) != state['start']:
            return None
        command = Path(f'/proc/{pid}/cmdline').read_bytes().split(b'\0')
        if os.fsencode(Path(__file__).resolve()) not in command:
            return None
        return pid
    except (OSError, ValueError, KeyError, TypeError):
        return None


def check_ports(ports=(8090, 5173)):
    for port in ports:
        with socket.socket() as sock:
            try:
                sock.bind(('127.0.0.1', port))
            except OSError as exc:
                raise RuntimeError(f'Port {port} is in use. Stop its owner before starting Facet.') from exc


def stop_children(children):
    # Each child owns a new process group, including its watcher and server.
    for child in children:
        try:
            os.killpg(child.pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
    deadline = time.monotonic() + 5
    for child in children:
        try:
            child.wait(timeout=max(0, deadline - time.monotonic()))
        except subprocess.TimeoutExpired:
            pass
    for child in children:
        try:
            os.killpg(child.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        child.wait()


def run():
    if not Path('/proc/self/stat').exists():
        raise RuntimeError('The native development supervisor requires Linux /proc.')
    STATE.parent.mkdir(parents=True, exist_ok=True)
    with (STATE.parent / 'dev.lock').open('w') as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as exc:
            raise RuntimeError('Development services are already running for this checkout.') from exc
        check_ports()
        children = []
        stopping = False

        def request_stop(_signum, _frame):
            nonlocal stopping
            stopping = True

        signal.signal(signal.SIGTERM, request_stop)
        signal.signal(signal.SIGINT, request_stop)
        STATE.write_text(json.dumps({'pid': os.getpid(), 'start': process_start(os.getpid())}))
        try:
            with (STATE.parent / 'backend.log').open('w') as backend_log:
                backend = subprocess.Popen(
                    [str(ROOT / 'scripts/dev-backend.sh')], cwd=ROOT,
                    start_new_session=True, stdout=backend_log, stderr=subprocess.STDOUT,
                )
                children.append(backend)
                print(f'Starting backend; logs: {backend_log.name}', flush=True)
                deadline = time.monotonic() + 180
                while not stopping:
                    if backend.poll() is not None:
                        raise RuntimeError(f'Backend exited; inspect {backend_log.name}. Your data is unchanged.')
                    try:
                        with urllib.request.urlopen('http://127.0.0.1:8090/api/health', timeout=1) as response:
                            if response.status == 200:
                                break
                    except (urllib.error.URLError, TimeoutError):
                        pass
                    if time.monotonic() >= deadline:
                        raise RuntimeError(f'Backend did not become healthy; inspect {backend_log.name}.')
                    time.sleep(0.2)
                if stopping:
                    return 0
                frontend = subprocess.Popen(
                    [str(ROOT / 'scripts/dev-frontend.sh')], cwd=ROOT, start_new_session=True,
                )
                children.append(frontend)
                print('Frontend starting at http://127.0.0.1:5173. Ctrl+C or make stop stops this session.', flush=True)
                while not stopping:
                    if any(child.poll() is not None for child in children):
                        raise RuntimeError('A development service exited unexpectedly.')
                    time.sleep(0.2)
                return 0
        finally:
            stop_children(children)
            STATE.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    group = parser.add_mutually_exclusive_group()
    group.add_argument('--stop', action='store_true')
    group.add_argument('--check-stopped', action='store_true')
    args = parser.parse_args()
    if args.stop or args.check_stopped:
        pid = active_supervisor()
        if not pid:
            print('No supervised development session is running for this checkout.')
            return 0
        if args.check_stopped:
            raise RuntimeError('Stop development services before backing up: make stop')
        os.kill(pid, signal.SIGTERM)
        deadline = time.monotonic() + 10
        while active_supervisor() == pid:
            if time.monotonic() >= deadline:
                raise RuntimeError('Development shutdown timed out; inspect the session before retrying.')
            time.sleep(0.1)
        print('Stopped this checkout\'s development services.')
        return 0
    return run()


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (RuntimeError, OSError) as error:
        print(f'facet dev: {error}', file=sys.stderr)
        sys.exit(1)
