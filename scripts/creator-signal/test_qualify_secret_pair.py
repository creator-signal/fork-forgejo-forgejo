"""Contracts for the existing native-only image qualification helper."""
import importlib.util
import io
import json
import os
import stat
from contextlib import redirect_stdout
from pathlib import Path
from types import SimpleNamespace
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
        native.LOCAL_DOCKER = Mock()

    def provider(self):
        return native.Provider(Mock(prove=Mock(return_value="172.18.0.2")))

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
                 patch.object(native, "LocalDockerRoute"), \
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

    def test_provider_fixed_owned_endpoint_status_and_private_value_denial(self):
        provider = self.provider()
        provider.token = "token-private"
        provider.private_values = {b"synthetic-private"}
        connection = self.provider_response([b'{"value":"synthetic-private"}', b""])
        with patch.object(native.time, "monotonic", return_value=0), \
             patch.object(native.http.client, "HTTPConnection", return_value=connection) as start:
            with self.assertRaisesRegex(RuntimeError, "disclosure denied"):
                provider.request("POST", "/api/v1/closed", {})
            self.assertEqual(start.call_args.args, ("172.18.0.2", 3000))
            connection.close.assert_called_once()

    def test_provider_rejects_redirect_and_oversized_response(self):
        for blocks, status in [([b"", b""], 302), ([b"x" * 16384] * 5, 200)]:
            connection = self.provider_response(blocks, status)
            with patch.object(native.time, "monotonic", return_value=0), \
                 patch.object(native.http.client, "HTTPConnection", return_value=connection):
                with self.assertRaises(RuntimeError):
                    self.provider().request("GET", "/api/v1/closed")
                connection.close.assert_called_once()

    def test_provider_rejects_expired_budget_before_network(self):
        with patch.object(native.time, "monotonic", return_value=101), \
             patch.object(native.http.client, "HTTPConnection") as start:
            with self.assertRaises(RuntimeError):
                self.provider().request("GET", "/api/healthz")
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
                                "PortBindings": {}},
                 "NetworkSettings": {"Networks": {"owned-net": {"NetworkID": "b" * 64}},
                                     "Ports": {"3000/tcp": None}},
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

    def probe_fixture(self):
        value = {"HostConfig": {"NetworkMode": "owned-net", "Privileged": False, "IpcMode": "private",
                                "PortBindings": {}},
                 "NetworkSettings": {"Networks": {"owned-net": {"NetworkID": "b" * 64,
                        "EndpointID": "c" * 64, "IPAddress": "172.18.0.2", "IPPrefixLen": 16}},
                                     "Ports": {"3000/tcp": None, "22/tcp": None}}, "Mounts": []}
        network = {"IPAM": {"Driver": "default", "Config": [{"Subnet": "172.18.0.0/16", "Gateway": "172.18.0.1"}]},
                   "EnableIPv6": False, "Containers": {"a" * 64: {"Name": "owned", "EndpointID": "c" * 64,
                                                              "IPv4Address": "172.18.0.2/16"}}}
        resources = native.container_resources(value, "owned-net", "b" * 64, [])
        return value, network, resources

    def test_nonpublished_internal_probe_denies_any_host_publication(self):
        value, _, resources = self.probe_fixture()
        self.assertEqual(resources["ports"], {"3000/tcp": None, "22/tcp": None})
        for key, bad in [("3000/tcp", [{"HostIp": "127.0.0.1", "HostPort": "1234"}]),
                         ("22/tcp", [{"HostIp": "0.0.0.0", "HostPort": "22"}]), ("3000/tcp", [])]:
            altered = json.loads(json.dumps(value))
            altered["NetworkSettings"]["Ports"][key] = bad
            with self.assertRaises(RuntimeError):
                native.container_resources(altered, "owned-net", "b" * 64, [])
        value["HostConfig"]["PortBindings"] = {"3000/tcp": [{"HostIp": "127.0.0.1", "HostPort": ""}]}
        with self.assertRaises(RuntimeError):
            native.container_resources(value, "owned-net", "b" * 64, [])

    def test_probe_requires_original_endpoint_and_only_original_private_network_peer(self):
        value, network, resources = self.probe_fixture()
        with patch.object(native, "own_container", return_value=value), patch.object(native, "own_network", return_value=network):
            custody = native.ProbeCustody("owned", "sha256:" + "d" * 64, "a" * 64, "owned-net", "b" * 64, [], resources)
            self.assertEqual(custody.prove(), "172.18.0.2")
            mutations = []
            for address in ("8.8.8.8", "172.19.0.2", "172.18.0.0", "172.18.255.255", "172.18.0.1"):
                changed_value, changed_network = json.loads(json.dumps(value)), json.loads(json.dumps(network))
                changed_value["NetworkSettings"]["Networks"]["owned-net"]["IPAddress"] = address
                changed_network["Containers"]["a" * 64]["IPv4Address"] = address + "/16"
                mutations.append((changed_value, changed_network))
            changed_network = json.loads(json.dumps(network))
            changed_network["Containers"]["e" * 64] = changed_network["Containers"]["a" * 64]
            mutations.append((value, changed_network))
            changed_value = json.loads(json.dumps(value))
            changed_value["NetworkSettings"]["Networks"]["foreign"] = {}
            mutations.append((changed_value, network))
            changed_value, changed_network = json.loads(json.dumps(value)), json.loads(json.dumps(network))
            changed_value["NetworkSettings"]["Networks"]["owned-net"]["EndpointID"] = "e" * 64
            changed_network["Containers"]["a" * 64]["EndpointID"] = "e" * 64
            mutations.append((changed_value, changed_network))
            for changed_value, changed_network in mutations:
                with patch.object(native, "own_container", return_value=changed_value), \
                     patch.object(native, "own_network", return_value=changed_network), self.assertRaises((RuntimeError, ValueError)):
                    custody.prove()

    def test_http_proves_original_custody_before_and_after_and_denies_drift(self):
        provider = self.provider()
        connection = self.provider_response([b"{}", b""])
        with patch.object(native.time, "monotonic", return_value=0), \
             patch.object(native.http.client, "HTTPConnection", return_value=connection):
            self.assertEqual(provider.request("GET", "/api/healthz"), {})
            self.assertEqual(provider.custody.prove.call_count, 2)
        provider.custody.prove.side_effect = RuntimeError("native probe endpoint identity denied")
        with patch.object(native.http.client, "HTTPConnection") as start, self.assertRaises(RuntimeError):
            provider.request("GET", "/api/healthz")
        start.assert_not_called()
        provider.custody.prove.side_effect = ["172.18.0.2", RuntimeError("native probe endpoint identity denied")]
        with patch.object(native.time, "monotonic", return_value=0), \
             patch.object(native.http.client, "HTTPConnection", return_value=self.provider_response([b"{}", b""])), \
             self.assertRaises(RuntimeError):
            provider.request("GET", "/api/healthz")

    def test_only_controlled_original_restart_can_adopt_successor_endpoint(self):
        value, network, resources = self.probe_fixture()
        with patch.object(native, "own_container", return_value=value), patch.object(native, "own_network", return_value=network), \
             patch.object(native, "command") as run:
            custody = native.ProbeCustody("owned", "sha256:" + "d" * 64, "a" * 64, "owned-net", "b" * 64, [], resources)
            with patch.object(custody, "_observe", side_effect=[custody.endpoint, ("172.18.0.2", "e" * 64, "172.18.0.0/16", "172.18.0.1")]):
                custody.restart()
            run.assert_called_once_with(["docker", "restart", "--time", "10", "a" * 64])
            self.assertEqual(custody.endpoint[1], "e" * 64)

    def test_local_socket_route_holds_named_identity_and_denies_mutable_ancestors(self):
        def metadata(mode, inode=1, uid=0):
            return SimpleNamespace(st_mode=mode, st_dev=1, st_ino=inode, st_uid=uid, st_gid=123,
                                   st_nlink=1, st_ctime_ns=1)
        paths = {path: metadata(stat.S_IFDIR | 0o755) for path in ("/", "/var", "/run")}
        paths["/var/run"] = metadata(stat.S_IFLNK | 0o777)
        paths["/run/docker.sock"] = metadata(stat.S_IFSOCK | 0o660, 2)
        with patch.object(native.os, "lstat", side_effect=lambda path: paths[path]), \
             patch.object(native.os, "readlink", return_value="/run"), patch.object(native.os, "open", return_value=99) as opened, \
             patch.object(native.os, "fstat", return_value=paths["/run/docker.sock"]), patch.object(native.os, "close") as close:
            route = native.LocalDockerRoute()
            opened.assert_called_once_with("/run/docker.sock", os.O_PATH | os.O_NOFOLLOW | os.O_CLOEXEC)
            route.check()
            paths["/run/docker.sock"] = metadata(stat.S_IFSOCK | 0o660, 3)
            with self.assertRaises(RuntimeError):
                route.check()
            route.close()
            close.assert_called_once_with(99)
            paths["/run"] = metadata(stat.S_IFDIR | 0o777)
            with self.assertRaises(RuntimeError):
                native.LocalDockerRoute()
        for mode in (stat.S_IFSOCK | 0o666, stat.S_IFREG | 0o660, stat.S_IFLNK | 0o777):
            paths["/run"] = metadata(stat.S_IFDIR | 0o755)
            paths["/run/docker.sock"] = metadata(mode)
            with patch.object(native.os, "lstat", side_effect=lambda path: paths[path]), \
                 patch.object(native.os, "readlink", return_value="/run"), patch.object(native.os, "open") as opened:
                with self.assertRaises(RuntimeError):
                    native.LocalDockerRoute()
                opened.assert_not_called()

    def test_command_uses_fixed_local_route_without_context_or_remote_environment(self):
        with patch.dict(os.environ, {"PATH": "/bin", "HOME": "/home/native", "DOCKER_HOST": "tcp://foreign:2375",
                                     "DOCKER_TLS_VERIFY": "1", "DOCKER_CERT_PATH": "/private"}, clear=True), \
             patch.object(native.time, "monotonic", return_value=0), \
             patch.object(native.subprocess, "Popen", side_effect=OSError("closed")) as start:
            with self.assertRaises(OSError):
                native.command(["docker", "inspect", "owned"])
            self.assertEqual(start.call_args.args[0], ["docker", "--host", "unix:///var/run/docker.sock", "inspect", "owned"])
            self.assertEqual(start.call_args.kwargs["env"], {"PATH": "/bin", "HOME": "/home/native"})
        for args in (["other", "inspect"], ["docker", "--host", "tcp://foreign"], ["docker", "--context=foreign", "inspect"]):
            with patch.object(native.subprocess, "Popen") as start, self.assertRaises(RuntimeError):
                native.command(args)
            start.assert_not_called()

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
