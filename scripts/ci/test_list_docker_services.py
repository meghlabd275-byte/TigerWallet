#!/usr/bin/env python3
"""Tests for the container build-matrix generator.

The generator decides what CI builds. Two failure modes matter and are covered
here:

  1. Dropping build arguments. The shared root-context Go Dockerfile selects
     which module it compiles via TARGET_MODULE, so losing that argument makes
     CI build the default binary and publish it under a different service name
     (e.g. the wallet_api binary tagged as master-wallet-backend).
  2. Aborting argument parsing on a comment line, which silently drops every
     argument that follows it.

Run: python3 scripts/ci/test_list_docker_services.py
"""

from __future__ import annotations

import importlib.util
import os
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
SPEC = importlib.util.spec_from_file_location(
    "list_docker_services", os.path.join(HERE, "list-docker-services.py")
)
lds = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(lds)


def write_compose(tmpdir, body):
    path = os.path.join(tmpdir, "docker-compose.yml")
    with open(path, "w", encoding="utf-8") as fh:
        fh.write(body)
    return path


class BuildArgsTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmpdir = self._tmp.name

    def tearDown(self):
        self._tmp.cleanup()

    def _resolve(self, body):
        # The generator resolves contexts against the process cwd, so run from
        # a directory where the context actually exists.
        path = write_compose(self.tmpdir, body)
        cwd = os.getcwd()
        os.chdir(self.tmpdir)
        try:
            return lds.resolve(path)
        finally:
            os.chdir(cwd)

    def test_build_args_are_captured(self):
        os.makedirs(os.path.join(self.tmpdir, "go", "wallet_api"), exist_ok=True)
        with open(os.path.join(self.tmpdir, "go", "wallet_api", "Dockerfile"), "w") as fh:
            fh.write("FROM scratch\n")

        services = self._resolve(
            "services:\n"
            "  wallet-api:\n"
            "    build:\n"
            "      context: ./go/wallet_api\n"
            "      dockerfile: Dockerfile\n"
            "      args:\n"
            "        TARGET_MODULE: go/wallet_api\n"
        )
        self.assertEqual(len(services), 1)
        self.assertEqual(services[0]["build_args"], ["TARGET_MODULE=go/wallet_api"])

    def test_comment_inside_args_does_not_abort_parsing(self):
        """A comment before an argument must not drop that argument."""
        os.makedirs(os.path.join(self.tmpdir, "web"), exist_ok=True)
        with open(os.path.join(self.tmpdir, "web", "Dockerfile"), "w") as fh:
            fh.write("FROM scratch\n")

        services = self._resolve(
            "services:\n"
            "  web:\n"
            "    build:\n"
            "      context: ./web\n"
            "      dockerfile: Dockerfile\n"
            "      args:\n"
            "        # Build-time API URL baked into the SPA.\n"
            "        VITE_API_URL: \"http://localhost:8095/api/v1\"\n"
        )
        self.assertEqual(len(services), 1)
        self.assertEqual(
            services[0]["build_args"], ["VITE_API_URL=http://localhost:8095/api/v1"]
        )

    def test_service_without_args_has_empty_list(self):
        os.makedirs(os.path.join(self.tmpdir, "web"), exist_ok=True)
        with open(os.path.join(self.tmpdir, "web", "Dockerfile"), "w") as fh:
            fh.write("FROM scratch\n")

        services = self._resolve(
            "services:\n"
            "  web:\n"
            "    build:\n"
            "      context: ./web\n"
            "      dockerfile: Dockerfile\n"
        )
        self.assertEqual(services[0]["build_args"], [])

    def test_empty_arg_value_is_preserved(self):
        """VITE_API_URL: \"\" is a real value, not an absent argument."""
        os.makedirs(os.path.join(self.tmpdir, "web"), exist_ok=True)
        with open(os.path.join(self.tmpdir, "web", "Dockerfile"), "w") as fh:
            fh.write("FROM scratch\n")

        services = self._resolve(
            "services:\n"
            "  web:\n"
            "    build:\n"
            "      context: ./web\n"
            "      dockerfile: Dockerfile\n"
            "      args:\n"
            "        VITE_API_URL: \"\"\n"
        )
        self.assertEqual(services[0]["build_args"], ["VITE_API_URL="])


class RealComposeTest(unittest.TestCase):
    """Guard the arguments that the shipped compose files actually declare."""

    def test_repo_compose_declares_target_module_for_both_go_services(self):
        repo_root = os.path.dirname(os.path.dirname(HERE))
        compose = os.path.join(repo_root, "docker-compose.yml")
        if not os.path.isfile(compose):
            self.skipTest("docker-compose.yml not present")

        cwd = os.getcwd()
        os.chdir(repo_root)
        try:
            services = {s["name"]: s for s in lds.resolve(compose)}
        finally:
            os.chdir(cwd)

        for name, module in (
            ("wallet-api", "go/wallet_api"),
            ("master-wallet-backend", "master_wallet/backend"),
        ):
            with self.subTest(service=name):
                self.assertIn(name, services)
                self.assertIn(
                    "TARGET_MODULE=" + module,
                    services[name]["build_args"],
                    "%s must select its module via TARGET_MODULE, otherwise CI "
                    "publishes the default module's binary" % name,
                )


if __name__ == "__main__":
    unittest.main(verbosity=2)
