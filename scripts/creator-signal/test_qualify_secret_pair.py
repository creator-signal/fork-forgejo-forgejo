"""Contracts for the existing native-only image qualification helper."""
import importlib.util
import io
import json
import os
from contextlib import redirect_stdout
from pathlib import Path
import unittest
from unittest.mock import Mock, patch


spec = importlib.util.spec_from_file_location(
    "qualify_secret_pair", Path(__file__).with_name("qualify_secret_pair.py")
)
native = importlib.util.module_from_spec(spec)
spec.loader.exec_module(native)


class NativeQualificationContracts(unittest.TestCase):
    def setUp(self):
        native.DEADLINE = 100

    def test_denial_codes_match_only_exact_source_owned_messages(self):
        self.assertEqual(native.denial_code(RuntimeError("native API assertion denied")), "ApiAssertion")
        for message, code in native.DENIAL_CODES.items():
            with self.subTest(code=code):
                self.assertEqual(native.denial_code(RuntimeError(message)), code)
                for altered in [message + ": sensitive-private", "sensitive-private " + message,
                                message + "\nsensitive-private", message.upper()]:
                    self.assertEqual(native.denial_code(RuntimeError(altered)), "UnknownDenied")

    def test_unknown_parser_and_exception_arguments_cannot_be_rendered(self):
        class HostileError(RuntimeError):
            def __str__(self):
                raise AssertionError("exception rendering is forbidden")

        class HostileArgument:
            def __str__(self):
                raise AssertionError("argument rendering is forbidden")

        errors = [json.JSONDecodeError("sensitive-private", '{"password":"sensitive-private"}', 1),
                  OSError("sensitive-private"), RuntimeError(),
                  RuntimeError("native API assertion denied", "sensitive-private"),
                  RuntimeError(HostileArgument()), HostileError("native API assertion denied")]
        for error in errors:
            with self.subTest(kind=type(error).__name__):
                self.assertEqual(native.denial_code(error), "UnknownDenied")

    def test_main_denial_remains_failed_and_emits_only_closed_metadata(self):
        argv = ["qualify_secret_pair", "--image", "sha256:" + "a" * 64,
                "--source", "b" * 40, "--run-id", "1", "--variant", "rootful", "--arch", "amd64"]
        for error, code in [(RuntimeError("native API assertion denied"), "ApiAssertion"),
                            (json.JSONDecodeError("sensitive-private", "sensitive-private", 0), "UnknownDenied"),
                            (RuntimeError("native API assertion denied: sensitive-private"), "UnknownDenied")]:
            output = io.StringIO()
            with patch("sys.argv", argv), patch.object(native.time, "monotonic", return_value=0), \
                 patch.object(native, "acceptance", side_effect=error) as accept, redirect_stdout(output):
                self.assertEqual(native.main(), 1)
            accept.assert_called_once()
            self.assertEqual(json.loads(output.getvalue()),
                             {"event": "forgejo-secret-pair-native-denied", "code": code})
            self.assertEqual(output.getvalue().count("\n"), 1)
            self.assertNotIn("sensitive-private", output.getvalue())

    def test_expired_command_never_starts_process(self):
        with patch.object(native.time, "monotonic", return_value=101), patch.object(native.subprocess, "Popen") as start:
            with self.assertRaises(RuntimeError):
                native.command(["docker", "inspect", "owned"])
            start.assert_not_called()

    def test_command_cannot_inherit_release_credentials(self):
        with patch.dict(os.environ, {"PATH": "/bin", "GH_TOKEN": "private", "ACTIONS_RUNTIME_TOKEN": "private"}, clear=True), \
             patch.object(native.time, "monotonic", return_value=0), \
             patch.object(native.subprocess, "Popen", side_effect=OSError("closed")) as start:
            with self.assertRaises(OSError):
                native.command(["docker", "inspect", "owned"])
            self.assertEqual(start.call_args.kwargs["env"], {"PATH": "/bin"})
            self.assertTrue(start.call_args.kwargs["start_new_session"])

    def test_cleanup_custody_checks_original_id_image_and_labels(self):
        observed = {"Name": "/owned", "Image": "sha256:" + "a" * 64, "Id": "b" * 64,
                    "Config": {"Labels": {"creator-signal.purpose": native.PURPOSE,
                                           "creator-signal.operation": "owned"}}}
        with patch.object(native, "inspect", return_value=observed):
            self.assertEqual(native.own_container("owned", observed["Image"], observed["Id"]), observed)
            for identity, image, name in [("c" * 64, observed["Image"], "owned"),
                                          (observed["Id"], "sha256:" + "c" * 64, "owned"),
                                          (observed["Id"], observed["Image"], "foreign")]:
                with self.assertRaises(RuntimeError):
                    native.own_container(name, image, identity)
            observed["Config"]["Labels"]["creator-signal.purpose"] = "foreign"
            with self.assertRaises(RuntimeError):
                native.own_container("owned", observed["Image"], observed["Id"])

    def provider_response(self, blocks, status=200):
        response = Mock(status=status)
        response.read1.side_effect = blocks
        connection = Mock()
        connection.getresponse.return_value = response
        return connection

    def test_provider_fixed_loopback_status_and_private_value_denial(self):
        provider = native.Provider(1234)
        provider.token = "token-private"
        provider.private_values = {b"synthetic-private"}
        connection = self.provider_response([b'{"value":"synthetic-private"}', b""])
        with patch.object(native.time, "monotonic", return_value=0), \
             patch.object(native.http.client, "HTTPConnection", return_value=connection) as start:
            with self.assertRaisesRegex(RuntimeError, "disclosure denied"):
                provider.request("POST", "/api/v1/closed", {})
            self.assertEqual(start.call_args.args, ("127.0.0.1", 1234))
            connection.close.assert_called_once()

    def test_provider_rejects_redirect_and_oversized_response(self):
        for blocks, status in [([b"", b""], 302), ([b"x" * 16384] * 5, 200)]:
            connection = self.provider_response(blocks, status)
            with patch.object(native.time, "monotonic", return_value=0), \
                 patch.object(native.http.client, "HTTPConnection", return_value=connection):
                with self.assertRaises(RuntimeError):
                    native.Provider(1234).request("GET", "/api/v1/closed")
                connection.close.assert_called_once()

    def test_provider_rejects_expired_budget_before_network(self):
        with patch.object(native.time, "monotonic", return_value=101), \
             patch.object(native.http.client, "HTTPConnection") as start:
            with self.assertRaises(RuntimeError):
                native.Provider(1234).request("GET", "/api/healthz")
            start.assert_not_called()

    def test_unknown_created_container_custody_prevents_cleanup(self):
        image = "sha256:" + "a" * 64
        image_info = [{"Id": image, "Architecture": "amd64", "Config": {"Volumes": {"/data": {}}, "Labels": {
            "org.opencontainers.image.revision": "b" * 40}}}]
        calls = []

        def command(args):
            calls.append(args)
            if args[:3] == ["docker", "image", "inspect"]:
                return json.dumps(image_info).encode()
            if args[:3] == ["docker", "network", "create"]:
                return ("c" * 64).encode()
            raise RuntimeError("creation response lost")

        with patch.object(native, "command", side_effect=command), \
             patch.object(native, "own_network", return_value={"Containers": {}}), \
             patch.object(native, "own_container", side_effect=RuntimeError("custody unknown")):
            with self.assertRaisesRegex(RuntimeError, "custody unknown"):
                native.acceptance(image, "b" * 40, "1", "rootful", "amd64")
        self.assertFalse(any(args[:2] == ["docker", "rm"] for args in calls))
        self.assertFalse(any(args[:3] == ["docker", "network", "rm"] for args in calls))

    def test_container_resource_guard_denies_foreign_mounts_and_host_access(self):
        value = {"HostConfig": {"NetworkMode": "owned-net", "Privileged": False, "IpcMode": "private",
                                "PortBindings": {"3000/tcp": [{"HostIp": "127.0.0.1", "HostPort": "1234"}]}},
                 "NetworkSettings": {"Networks": {"owned-net": {"NetworkID": "b" * 64}},
                                     "Ports": {"3000/tcp": [{"HostIp": "127.0.0.1", "HostPort": "1234"}]}},
                 "Mounts": [{"Type": "volume", "Driver": "local", "Name": "a" * 64,
                             "Destination": "/data", "Source": "/owned/anonymous", "RW": True}]}
        expected = native.container_resources(value, "owned-net", "b" * 64, ["/data"])
        self.assertEqual(expected["mounts"], value["Mounts"])
        for key, bad in [("Privileged", True), ("Binds", ["/foreign:/data"]),
                         ("VolumesFrom", ["foreign"]), ("CapAdd", ["SYS_ADMIN"]),
                         ("NetworkMode", "host"), ("PidMode", "host"), ("IpcMode", "host")]:
            altered = json.loads(json.dumps(value))
            altered["HostConfig"][key] = bad
            with self.assertRaises(RuntimeError):
                native.container_resources(altered, "owned-net", "b" * 64, ["/data"])
        value["Mounts"][0]["Type"] = "bind"
        with self.assertRaises(RuntimeError):
            native.container_resources(value, "owned-net", "b" * 64, ["/data"])

    def test_container_resource_guard_rejects_nonloopback_or_extra_ports(self):
        value = {"HostConfig": {"NetworkMode": "owned-net", "Privileged": False, "IpcMode": "private",
                                "PortBindings": {"3000/tcp": [{"HostIp": "127.0.0.1", "HostPort": "1234"}]}},
                 "NetworkSettings": {"Networks": {"owned-net": {"NetworkID": "b" * 64}},
                                     "Ports": {"3000/tcp": [{"HostIp": "0.0.0.0", "HostPort": "1234"}]}},
                 "Mounts": []}
        with self.assertRaises(RuntimeError):
            native.container_resources(value, "owned-net", "b" * 64, [])
        value["NetworkSettings"]["Ports"]["3000/tcp"][0]["HostIp"] = "127.0.0.1"
        value["NetworkSettings"]["Ports"]["9999/tcp"] = [{"HostIp": "127.0.0.1", "HostPort": "9999"}]
        with self.assertRaises(RuntimeError):
            native.container_resources(value, "owned-net", "b" * 64, [])

    def test_published_port_denials_are_closed_and_preserve_all_predicates(self):
        value = {"HostConfig": {"NetworkMode": "owned-net", "Privileged": False, "IpcMode": "private",
                                "PortBindings": {"3000/tcp": [{"HostIp": "127.0.0.1", "HostPort": "1234"}]}},
                 "NetworkSettings": {"Networks": {"owned-net": {"NetworkID": "b" * 64}},
                                     "Ports": {"3000/tcp": [{"HostIp": "127.0.0.1", "HostPort": "1234"}],
                                               "22/tcp": None}}, "Mounts": []}
        self.assertEqual(native.container_resources(value, "owned-net", "b" * 64, [])["ports"],
                         value["NetworkSettings"]["Ports"])
        cases = [(None, "PublishedPortBindingsAbsent"), ({}, "PublishedPortBindingType"),
                 ([], "PublishedPortBindingCardinality"),
                 ([{"HostIp": "127.0.0.1", "HostPort": "1234"}] * 2, "PublishedPortBindingCardinality"),
                 ([{"HostIp": "0.0.0.0", "HostPort": "1234"}], "PublishedPortLoopback"),
                 ([{"HostIp": "127.0.0.1", "HostPort": "sensitive-private"}], "PublishedPortSyntax"),
                 ([{"HostIp": "127.0.0.1", "HostPort": "65536"}], "PublishedPortRange")]
        for bindings, code in cases:
            altered = json.loads(json.dumps(value))
            altered["NetworkSettings"]["Ports"]["3000/tcp"] = bindings
            with self.assertRaises(RuntimeError) as denied:
                native.container_resources(altered, "owned-net", "b" * 64, [])
            self.assertEqual(native.denial_code(denied.exception), code)
        for port in ["0", "01234", "-1", "655350", "1234\nsensitive-private"]:
            altered = json.loads(json.dumps(value))
            altered["NetworkSettings"]["Ports"]["3000/tcp"][0]["HostPort"] = port
            with self.assertRaises(RuntimeError) as denied:
                native.container_resources(altered, "owned-net", "b" * 64, [])
            self.assertEqual(native.denial_code(denied.exception), "PublishedPortSyntax")
        altered = json.loads(json.dumps(value))
        altered["NetworkSettings"]["Ports"]["22/tcp"] = [{"HostIp": "127.0.0.1", "HostPort": "1234"}]
        with self.assertRaises(RuntimeError) as denied:
            native.container_resources(altered, "owned-net", "b" * 64, [])
        self.assertEqual(native.denial_code(denied.exception), "ExtraPublishedPort")

    def test_network_guard_requires_original_private_identity(self):
        network = {"Name": "owned-net", "Id": "b" * 64, "Driver": "bridge", "Internal": True,
                   "Labels": {"creator-signal.purpose": native.PURPOSE, "creator-signal.operation": "owned-net"}}
        with patch.object(native, "command", return_value=json.dumps([network]).encode()):
            self.assertEqual(native.own_network("owned-net", "b" * 64), network)
            with self.assertRaises(RuntimeError):
                native.own_network("owned-net", "c" * 64)
        network["Internal"] = False
        with patch.object(native, "command", return_value=json.dumps([network]).encode()):
            with self.assertRaises(RuntimeError):
                native.own_network("owned-net", "b" * 64)


if __name__ == "__main__":
    unittest.main()
