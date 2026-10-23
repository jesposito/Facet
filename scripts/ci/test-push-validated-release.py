"""Exercise release validation ordering and failures without GitHub writes."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest


SCRIPT = Path(__file__).with_name("push-validated-release.sh")
MOCK = r'''#!/usr/bin/env python3
import json
import os
from pathlib import Path
import sys

root = Path(os.environ["MOCK_ROOT"])
tool = Path(sys.argv[0]).name
args = sys.argv[1:]
with (root / "calls").open("a") as log:
    log.write(json.dumps([tool, *args]) + "\n")
scenario = os.environ["MOCK_SCENARIO"]
revision = root / "revision"
if tool == "git":
    if args == ["rev-parse", "HEAD"]:
        print(revision.read_text() if revision.exists() else "sha-one")
    elif args == ["pull", "--rebase", "origin", "main"]:
        revision.write_text("sha-two")
    elif args == ["push", "origin", "HEAD:refs/heads/main"]:
        if scenario == "reject" or (scenario == "rebase" and not revision.exists()):
            sys.exit(1)
elif tool == "gh":
    if args[:2] == ["workflow", "run"] and scenario == "dispatch-failure":
        sys.exit(1)
    if args[:2] == ["run", "list"]:
        print("123")
    elif args[:2] == ["run", "watch"] and scenario == "failed-check":
        sys.exit(1)
'''


class ReleaseValidationTests(unittest.TestCase):
    def run_scenario(self, scenario):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            mock = root / "mock"
            mock.write_text(MOCK)
            mock.chmod(0o755)
            for tool in ("git", "gh"):
                (root / tool).symlink_to(mock)
            env = dict(os.environ, PATH=f"{root}:{os.environ['PATH']}",
                       MOCK_ROOT=directory, MOCK_SCENARIO=scenario,
                       GITHUB_REPOSITORY="example/facet", GITHUB_RUN_ID="42")
            result = subprocess.run(["bash", str(SCRIPT), "v1.2.3"],
                                    env=env, capture_output=True, text=True, timeout=10)
            calls = [json.loads(line) for line in (root / "calls").read_text().splitlines()]
            return result, calls

    def test_checks_finish_before_main_push(self):
        result, calls = self.run_scenario("success")
        self.assertEqual(result.returncode, 0, result.stderr)
        watch = next(i for i, c in enumerate(calls) if c[:3] == ["gh", "run", "watch"])
        push = calls.index(["git", "push", "origin", "HEAD:refs/heads/main"])
        self.assertLess(watch, push)
        self.assertIn("--exit-status", calls[watch])
        dispatch = next(c for c in calls if c[:3] == ["gh", "workflow", "run"])
        self.assertIn("scan_published_image=false", dispatch)
        self.assertEqual(calls[-1], ["git", "push", "origin", "--delete",
                                     "release/changelog-v1.2.3-42-1"])

    def test_failed_checks_never_push_main(self):
        result, calls = self.run_scenario("failed-check")
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn(["git", "push", "origin", "HEAD:refs/heads/main"], calls)
        self.assertEqual(calls[-1][3], "--delete")

    def test_failed_dispatch_never_pushes_main(self):
        result, calls = self.run_scenario("dispatch-failure")
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn(["git", "push", "origin", "HEAD:refs/heads/main"], calls)

    def test_rebase_requires_new_validation(self):
        result, calls = self.run_scenario("rebase")
        self.assertEqual(result.returncode, 0, result.stderr)
        runs = [c for c in calls if c[:3] == ["gh", "run", "list"]]
        self.assertEqual([c[c.index("--commit") + 1] for c in runs], ["sha-one", "sha-two"])
        rebase = calls.index(["git", "pull", "--rebase", "origin", "main"])
        later = calls[rebase + 1:]
        watch = next(i for i, c in enumerate(later) if c[:3] == ["gh", "run", "watch"])
        self.assertLess(watch, later.index(["git", "push", "origin", "HEAD:refs/heads/main"]))

    def test_repeated_push_rejection_fails_after_three_attempts(self):
        result, calls = self.run_scenario("reject")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(calls.count(["git", "push", "origin", "HEAD:refs/heads/main"]), 3)


class OptionalChangelogTests(unittest.TestCase):
    def test_optional_services_cannot_abort_release(self):
        workflow = SCRIPT.parents[2] / ".github/workflows/auto-tag.yml"
        source = workflow.read_text().split("          # Try GitHub Models (Copilot) first", 1)[1]
        source = source.split("          # Process AI response if successful", 1)[0]
        source = textwrap.dedent(source.split("\n", 1)[1])
        for scenario in ("dns", "timeout", "html", "bad-json", "wrong-shape", "valid", "openai-fallback"):
            with self.subTest(scenario=scenario), tempfile.TemporaryDirectory() as directory:
                mock = Path(directory) / "curl"
                mock.write_text('''#!/usr/bin/env python3
import json, os, sys
scenario = os.environ["MOCK_SCENARIO"]
if scenario == "dns": sys.exit(6)
if scenario == "timeout": sys.exit(28)
if scenario == "html": print("<html>Bad gateway</html>"); sys.exit(0)
if scenario == "openai-fallback" and any("azure.com" in x for x in sys.argv): sys.exit(6)
content = json.dumps([{"number": 490, "category": "BUG", "description": "Dependency fixes"}])
if scenario == "bad-json": content = "not json"
if scenario == "wrong-shape": content = '{"error":"not a list"}'
print(json.dumps({"choices":[{"message":{"content": content}}]}))
''')
                mock.chmod(0o755)
                env = dict(os.environ, PATH=f"{directory}:{os.environ['PATH']}",
                           MOCK_SCENARIO=scenario, GH_TOKEN="fixture-token",
                           OPENAI_API_KEY="fixture-token", PROMPT="fixture-prompt")
                script = 'AI_SUCCESS=false\nAI_SOURCE=""\n' + source + '\nprintf "RESULT=%s\\n" "$AI_SUCCESS"\n'
                result = subprocess.run(["bash", "-e", "-o", "pipefail", "-c", script],
                                        env=env, capture_output=True, text=True, timeout=10)
                self.assertEqual(result.returncode, 0, result.stderr)
                expected = "true" if scenario in ("valid", "openai-fallback") else "false"
                self.assertIn(f"RESULT={expected}", result.stdout)


if __name__ == "__main__":
    unittest.main()
