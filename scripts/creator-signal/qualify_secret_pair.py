#!/usr/bin/env python3
"""Native-only, isolated HTTP acceptance of the exact fork image pair operation.

The existing qualification/pull-back jobs own this helper. It has no live
provider URL, persistent configuration, release credential, or publication API.
"""
from __future__ import annotations

import argparse
import concurrent.futures
import http.client
import ipaddress
import json
import os
import re
import secrets
import selectors
import signal
import stat
import subprocess
import time


PURPOSE = "creator-signal-fork-secret-pair-native"
NAMES = ["ZOT_VM_ARTIFACT_PASSWORD", "ZOT_VM_ARTIFACT_USERNAME"]
DEADLINE = 0.0
LOCAL_DOCKER = None


class LocalDockerRoute:
    """Held metadata custody of the existing hosted Native Unix daemon route."""

    def __init__(self):
        self.fd = None
        self.pid = os.getpid()
        self.ancestors = self._ancestors()
        before = os.lstat("/run/docker.sock")
        if (not stat.S_ISSOCK(before.st_mode) or before.st_uid != 0
                or before.st_nlink != 1 or stat.S_IMODE(before.st_mode) not in (0o600, 0o660)):
            raise RuntimeError("native local Docker socket custody denied")
        self.identity = self._identity(before)
        self.fd = os.open("/run/docker.sock", os.O_PATH | os.O_NOFOLLOW | os.O_CLOEXEC)
        try:
            self.check()
        except BaseException:
            self.close()
            raise

    @staticmethod
    def _identity(value):
        return (value.st_dev, value.st_ino, value.st_mode, value.st_uid, value.st_gid,
                value.st_nlink, value.st_ctime_ns)

    @staticmethod
    def _ancestors():
        identities = []
        for path in ("/", "/var", "/run"):
            value = os.lstat(path)
            if not stat.S_ISDIR(value.st_mode) or value.st_uid != 0 or value.st_mode & 0o022:
                raise RuntimeError("native local Docker route denied")
            identities.append(LocalDockerRoute._identity(value)[:5])
        alias = os.lstat("/var/run")
        if not stat.S_ISLNK(alias.st_mode) or alias.st_uid != 0 or os.readlink("/var/run") != "/run":
            raise RuntimeError("native local Docker route denied")
        identities.append(LocalDockerRoute._identity(alias)[:5])
        return tuple(identities)

    def check(self):
        if (os.getpid() != self.pid or self._ancestors() != self.ancestors or self.fd is None
                or self._identity(os.fstat(self.fd)) != self.identity
                or self._identity(os.lstat("/run/docker.sock")) != self.identity):
            raise RuntimeError("native local Docker socket custody denied")

    def close(self):
        if self.fd is not None:
            os.close(self.fd)
            self.fd = None


def remaining(cap: float = 30) -> float:
    value = min(cap, DEADLINE - time.monotonic())
    if value <= 0:
        raise RuntimeError("native acceptance deadline exhausted")
    return value


def command(args: list[str]) -> bytes:
    # Do not inherit publication tokens into the CLI or image entrypoints.
    if (LOCAL_DOCKER is None or not args or args[0] != "docker"
            or any(arg in ("-H", "--host", "--context") or arg.startswith(("--host=", "--context=", "-H"))
                   for arg in args[1:])):
        raise RuntimeError("native fixed Docker command denied")
    LOCAL_DOCKER.check()
    env = {key: value for key, value in os.environ.items() if key in {"PATH", "HOME"}}
    deadline = time.monotonic() + remaining()
    child = None
    output = bytearray()
    total = 0
    try:
        child = subprocess.Popen(["docker", "--host", "unix:///var/run/docker.sock", *args[1:]],
                                 stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                 env=env, start_new_session=True)
        with selectors.DefaultSelector() as watcher:
            watcher.register(child.stdout, selectors.EVENT_READ, True)
            watcher.register(child.stderr, selectors.EVENT_READ, False)
            while watcher.get_map():
                budget = min(deadline - time.monotonic(), remaining())
                if budget <= 0:
                    raise RuntimeError("native CLI deadline exhausted")
                for key, _ in watcher.select(budget):
                    block = os.read(key.fileobj.fileno(), 16384)
                    if not block:
                        watcher.unregister(key.fileobj)
                        continue
                    total += len(block)
                    if total > 65536:
                        raise RuntimeError("native CLI output exceeded bound")
                    if key.data:
                        output.extend(block)
        budget = min(deadline - time.monotonic(), remaining())
        if budget <= 0:
            raise RuntimeError("native CLI deadline exhausted")
        if child.wait(timeout=budget) != 0:
            raise RuntimeError("native CLI denied")
        return bytes(output)
    finally:
        try:
            if child is not None:
                if child.poll() is None:
                    os.killpg(child.pid, signal.SIGKILL)
                    child.wait(timeout=5)
                child.stdout.close()
                child.stderr.close()
        finally:
            LOCAL_DOCKER.check()


def inspect(container: str) -> dict:
    values = json.loads(command(["docker", "inspect", container]))
    if not isinstance(values, list) or len(values) != 1:
        raise RuntimeError("native container observation denied")
    return values[0]


def own_container(container: str, image: str, identity: str | None = None) -> dict:
    value = inspect(container)
    if (value.get("Name") != "/" + container or value.get("Image") != image
            or value.get("Config", {}).get("Labels", {}).get("creator-signal.purpose") != PURPOSE
            or value.get("Config", {}).get("Labels", {}).get("creator-signal.operation") != container
            or (identity is not None and value.get("Id") != identity)):
        raise RuntimeError("native container custody denied")
    return value


def own_network(name: str, identity: str) -> dict:
    values = json.loads(command(["docker", "network", "inspect", name]))
    if not isinstance(values, list) or len(values) != 1:
        raise RuntimeError("native network observation denied")
    value = values[0]
    if (value.get("Name") != name or value.get("Id") != identity
            or value.get("Driver") != "bridge" or value.get("Internal") is not True
            or value.get("Labels", {}).get("creator-signal.purpose") != PURPOSE
            or value.get("Labels", {}).get("creator-signal.operation") != name):
        raise RuntimeError("native network custody denied")
    return value


def container_resources(value: dict, network: str, network_id: str, destinations: list[str]) -> dict:
    host = value.get("HostConfig", {})
    if (host.get("NetworkMode") != network or host.get("Privileged") is not False
            or any(host.get(key) for key in ("Binds", "VolumesFrom", "Devices", "DeviceRequests", "CapAdd", "PidMode", "UTSMode", "SecurityOpt"))
            or host.get("IpcMode") not in ("private", "")):
        raise RuntimeError("native container isolation denied")
    networks = value.get("NetworkSettings", {}).get("Networks", {})
    if set(networks) != {network} or networks[network].get("NetworkID") != network_id:
        raise RuntimeError("native container network attachment denied")
    ports = value.get("NetworkSettings", {}).get("Ports", {})
    if not isinstance(ports, dict) or any(binding is not None for binding in ports.values()):
        raise RuntimeError("native published port inventory denied")
    port_config = host.get("PortBindings", {})
    if port_config not in (None, {}):
        raise RuntimeError("native declared port publication denied")
    mounts = value.get("Mounts")
    if not isinstance(mounts, list) or sorted(row.get("Destination", "") for row in mounts) != sorted(destinations):
        raise RuntimeError("native image volume inventory denied")
    for row in mounts:
        if (row.get("Type") != "volume" or row.get("Driver") != "local"
                or not re.fullmatch(r"[0-9a-f]{64}", row.get("Name", ""))
                or row.get("RW") is not True):
            raise RuntimeError("native anonymous volume custody denied")
    return json.loads(json.dumps({"mounts": mounts, "ports": ports, "portBindings": port_config,
                                "entrypoint": value.get("Config", {}).get("Entrypoint"),
                                "command": value.get("Config", {}).get("Cmd")}, sort_keys=True))


class ProbeCustody:
    def __init__(self, container, image, identity, network, network_id, destinations, resources):
        self.container, self.image, self.identity = container, image, identity
        self.network, self.network_id = network, network_id
        self.destinations, self.resources = destinations, resources
        self.endpoint = self._observe()

    def _observe(self):
        value = own_container(self.container, self.image, self.identity)
        if container_resources(value, self.network, self.network_id, self.destinations) != self.resources:
            raise RuntimeError("native probe resource identity denied")
        network = own_network(self.network, self.network_id)
        configs = network.get("IPAM", {}).get("Config")
        peers = network.get("Containers", {})
        endpoint = value.get("NetworkSettings", {}).get("Networks", {}).get(self.network, {})
        if (network.get("EnableIPv6") is not False or network.get("IPAM", {}).get("Driver") != "default"
                or not isinstance(configs, list) or len(configs) != 1
                or set(peers) != {self.identity}):
            raise RuntimeError("native probe endpoint denied")
        subnet = ipaddress.IPv4Network(configs[0].get("Subnet", ""), strict=True)
        private = [ipaddress.IPv4Network(prefix) for prefix in ("10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16")]
        address = ipaddress.IPv4Address(endpoint.get("IPAddress", ""))
        peer = peers[self.identity]
        endpoint_id = endpoint.get("EndpointID", "")
        if (not any(subnet.subnet_of(block) for block in private)
                or address not in subnet or address in (subnet.network_address, subnet.broadcast_address)
                or str(address) == configs[0].get("Gateway")
                or endpoint.get("IPPrefixLen") != subnet.prefixlen
                or not re.fullmatch(r"[0-9a-f]{64}", endpoint_id)
                or peer.get("Name") != self.container or peer.get("EndpointID") != endpoint_id
                or peer.get("IPv4Address") != f"{address}/{subnet.prefixlen}"
                or endpoint.get("GlobalIPv6Address", "") or peer.get("IPv6Address", "")):
            raise RuntimeError("native probe endpoint denied")
        return str(address), endpoint_id, str(subnet), configs[0].get("Gateway")

    def prove(self):
        LOCAL_DOCKER.check()
        if self._observe() != self.endpoint:
            raise RuntimeError("native probe endpoint identity denied")
        LOCAL_DOCKER.check()
        return self.endpoint[0]

    def restart(self):
        # Only this original-container restart may earn a successor endpoint.
        self.prove()
        command(["docker", "restart", "--time", "10", self.identity])
        self.endpoint = self._observe()

    def exec(self, args):
        self.prove()
        try:
            return command(["docker", "exec", "--user", "git", self.identity, *args])
        finally:
            self.prove()


class Provider:
    def __init__(self, custody: ProbeCustody):
        self.custody = custody
        self.token: str | None = None
        self.private_values: set[bytes] = set()

    def request(self, method: str, path: str, body=None, status: int = 200):
        if not path.startswith("/api/") or "\n" in path or "\r" in path:
            raise RuntimeError("native API path denied")
        headers = {"Content-Type": "application/json"}
        if self.token:
            headers["Authorization"] = "token " + self.token
        raw = body if isinstance(body, bytes) else (json.dumps(body).encode() if body is not None else None)
        address = self.custody.prove()
        connection = http.client.HTTPConnection(address, 3000, timeout=remaining(15))
        deadline = time.monotonic() + remaining(15)
        try:
            connection.request(method, path, raw, headers)
            budget = min(deadline - time.monotonic(), remaining(15))
            if budget <= 0:
                raise RuntimeError("native API deadline exhausted")
            if connection.sock is not None:
                connection.sock.settimeout(budget)
            response = connection.getresponse()
            received = bytearray()
            while True:
                budget = min(deadline - time.monotonic(), remaining(15))
                if budget <= 0:
                    raise RuntimeError("native API deadline exhausted")
                if connection.sock is not None:
                    connection.sock.settimeout(budget)
                block = response.read1(16384)
                if not block:
                    break
                received.extend(block)
                if len(received) > 65536:
                    raise RuntimeError("native API response exceeded bound")
            if response.status != status:
                raise RuntimeError("native API assertion denied")
            if any(value in received for value in self.private_values):
                raise RuntimeError("native API private-material disclosure denied")
            return json.loads(received) if received else None
        finally:
            connection.close()
            self.custody.prove()


def acceptance(image: str, source: str, run_id: str, variant: str, arch: str, actions_enabled: bool = True) -> dict:
    container = f"cs-pair-{variant}-{arch}-{run_id}" + ("" if actions_enabled else "-disabled")
    identity = None
    attempted = False
    network = container + "-net"
    network_id = None
    mounts = None
    destinations = []
    try:
        image_info = json.loads(command(["docker", "image", "inspect", image]))[0]
        if (image_info.get("Id") != image or image_info.get("Architecture") != arch
                or image_info.get("Config", {}).get("Labels", {}).get("org.opencontainers.image.revision") != source):
            raise RuntimeError("native image identity denied")
        destinations = sorted((image_info.get("Config", {}).get("Volumes") or {}).keys())
        expected_destinations = ["/data"] if variant == "rootful" else ["/var/lib/gitea"]
        if destinations != expected_destinations:
            raise RuntimeError("native image declared volumes denied")
        network_id = command(["docker", "network", "create", "--internal",
                              "--label", "creator-signal.purpose=" + PURPOSE,
                              "--label", "creator-signal.operation=" + network, network]).decode().strip()
        if not re.fullmatch(r"[0-9a-f]{64}", network_id):
            raise RuntimeError("native network creation response denied")
        own_network(network, network_id)
        attempted = True
        created = command(["docker", "run", "--detach", "--name", container,
                           "--label", "creator-signal.purpose=" + PURPOSE,
                           "--label", "creator-signal.operation=" + container,
                           "--network", network,
                           "--env", "FORGEJO__database__DB_TYPE=sqlite3",
                           "--env", "FORGEJO__server__DISABLE_SSH=true",
                           "--env", "FORGEJO__security__INSTALL_LOCK=true",
                           "--env", "FORGEJO__actions__ENABLED=" + ("true" if actions_enabled else "false"), image]).decode().strip()
        if not re.fullmatch(r"[0-9a-f]{64}", created):
            raise RuntimeError("native creation response denied")
        identity = created
        value = own_container(container, image, identity)
        mounts = container_resources(value, network, network_id, destinations)
        if (mounts["entrypoint"] != image_info.get("Config", {}).get("Entrypoint")
                or mounts["command"] != image_info.get("Config", {}).get("Cmd")):
            raise RuntimeError("native image entrypoint substitution denied")
        provider = Provider(ProbeCustody(container, image, identity, network, network_id, destinations, mounts))
        for _ in range(90):
            try:
                provider.request("GET", "/api/healthz")
                break
            except (OSError, RuntimeError):
                time.sleep(min(2, remaining()))
        else:
            raise RuntimeError("native readiness denied")
        provider.custody.exec(["forgejo", "admin", "user", "create",
                 "--username", "cs-pair-native", "--email", "native@example.invalid", "--random-password",
                 "--must-change-password=false", "--admin"])
        token = provider.custody.exec(["forgejo", "admin", "user",
                         "generate-access-token", "--username", "cs-pair-native", "--token-name", "pair-native",
                         "--scopes", "write:repository,write:user", "--raw"]).decode().strip()
        if not re.fullmatch(r"[0-9a-f]{40,64}", token):
            raise RuntimeError("native synthetic token denied")
        provider.token = token
        provider.private_values.add(token.encode())
        prefix = "/api/v1/repos/cs-pair-native/atomic-pair"
        repo = provider.request("POST", "/api/v1/user/repos", {"name": "atomic-pair", "private": True}, 201)
        operation = secrets.token_hex(32)
        binding = {key: secrets.token_hex(32) for key in ("ownershipId", "transactionId", "nonce")}
        body = {"schema": "creator-signal.actions-secret-pair-operation/v1", "purpose": "vm-artifact-publisher",
                "operationId": operation, "binding": binding,
                "secrets": {"ZOT_VM_ARTIFACT_USERNAME": "vm-image-publisher",
                            "ZOT_VM_ARTIFACT_PASSWORD": "cs-vm-artifact-" + secrets.token_urlsafe(32)}}
        provider.private_values.add(body["secrets"]["ZOT_VM_ARTIFACT_PASSWORD"].encode())
        route = prefix + "/actions/secret-pair-operations/" + operation
        if not actions_enabled:
            provider.request("POST", route, body, 404)
            provider.request("GET", prefix + "/actions/secrets", status=404)
            return {"checks": ["actions-disabled-route-denial"]}
        expected = {"schema": "creator-signal.actions-secret-pair-operation-result/v1", "operationId": operation,
                    "binding": binding, "state": "Applied", "replayed": False}
        result = provider.request("POST", route, body, 201)
        if result != expected:
            raise RuntimeError("native closed operation reply denied")
        expected["replayed"] = True
        with concurrent.futures.ThreadPoolExecutor(max_workers=4) as executor:
            replies = list(executor.map(lambda _: provider.request("POST", route, body), range(4)))
        if any(reply != expected for reply in replies):
            raise RuntimeError("native concurrent replay denied")
        altered = json.loads(json.dumps(body))
        altered["binding"]["nonce"] = secrets.token_hex(32)
        provider.request("POST", route, altered, 409)
        changed_material = json.loads(json.dumps(body))
        changed_material["secrets"]["ZOT_VM_ARTIFACT_PASSWORD"] = "cs-vm-artifact-" + secrets.token_urlsafe(32)
        provider.private_values.add(changed_material["secrets"]["ZOT_VM_ARTIFACT_PASSWORD"].encode())
        provider.request("POST", route, changed_material, 409)
        for name in NAMES:
            provider.request("PUT", prefix + "/actions/secrets/" + name, {"data": "foreign"}, 409)
            provider.request("DELETE", prefix + "/actions/secrets/" + name, status=409)
        duplicate = json.dumps(body).encode()[:-1] + b',"purpose":"vm-artifact-publisher"}'
        provider.request("POST", route, duplicate, 400)
        extra = dict(body, unknown=True)
        provider.request("POST", route, extra, 400)
        before = provider.request("GET", prefix + "/actions/secrets")
        if not isinstance(before, list) or sorted(row.get("name") for row in before) != NAMES:
            raise RuntimeError("native actual secret inventory denied")
        provider.custody.restart()
        if container_resources(own_container(container, image, identity), network, network_id, destinations) != mounts:
            raise RuntimeError("native restart resource identity denied")
        for _ in range(45):
            try:
                provider.request("GET", "/api/healthz")
                break
            except (OSError, RuntimeError):
                time.sleep(min(2, remaining()))
        else:
            raise RuntimeError("native restart readiness denied")
        if provider.request("POST", route, body) != expected or provider.request("GET", prefix + "/actions/secrets") != before:
            raise RuntimeError("native restart no-write replay denied")
        provider.request("DELETE", prefix, status=204)
        recreated = provider.request("POST", "/api/v1/user/repos", {"name": "atomic-pair", "private": True}, 201)
        if recreated.get("id") == repo.get("id"):
            raise RuntimeError("native repository identity reuse denied")
        provider.request("POST", route, body, 409)
        if provider.request("GET", prefix + "/actions/secrets") != []:
            raise RuntimeError("native revoked operation resurrected")
        foreign_prefix = "/api/v1/repos/cs-pair-native/foreign-pair"
        provider.request("POST", "/api/v1/user/repos", {"name": "foreign-pair", "private": True}, 201)
        provider.request("PUT", foreign_prefix + "/actions/secrets/UNRELATED_SECRET", {"data": "foreign"}, 201)
        for name in NAMES:
            # Reserved destinations cannot be populated by the legacy route,
            # including when no pair operation currently owns that repository.
            provider.request("PUT", foreign_prefix + "/actions/secrets/" + name, {"data": "foreign"}, 409)
        foreign_inventory = provider.request("GET", foreign_prefix + "/actions/secrets")
        if [row.get("name") for row in foreign_inventory] != ["UNRELATED_SECRET"]:
            raise RuntimeError("native unrelated secret compatibility denied")
        return {"schema": "creator-signal.forgejo-secret-pair-native/v1", "sourceRevision": source,
                "imageId": image, "variant": variant, "architecture": arch,
                "checks": ["atomic-create", "concurrent-replay", "binding-conflict", "legacy-mutation-denial",
                           "closed-json", "restart-no-write-replay", "repository-tombstone",
                           "reserved-destination-denial", "unrelated-secret-compatibility"],
                "sourceQualified": False, "factoryQualified": False}
    finally:
        if attempted:
            # Ambiguous creation/observation does not authorize broad cleanup.
            current_mounts = container_resources(own_container(container, image, identity), network, network_id, destinations)
            if mounts is None or current_mounts != mounts:
                raise RuntimeError("native retained resource cleanup denied")
            command(["docker", "rm", "--force", "--volumes", identity])
        if network_id:
            current_network = own_network(network, network_id)
            if current_network.get("Containers"):
                raise RuntimeError("native network cleanup has active consumers")
            command(["docker", "network", "rm", network_id])


DENIAL_CODES = {
    "native acceptance deadline exhausted": "AcceptanceDeadline",
    "native CLI deadline exhausted": "CliDeadline",
    "native CLI output exceeded bound": "CliOutputBound",
    "native CLI denied": "CliDenied",
    "native container observation denied": "ContainerObservation",
    "native container custody denied": "ContainerCustody",
    "native network observation denied": "NetworkObservation",
    "native network custody denied": "NetworkCustody",
    "native container isolation denied": "ContainerIsolation",
    "native container network attachment denied": "NetworkAttachment",
    "native published port inventory denied": "PublishedPortInventory",
    "native declared port publication denied": "DeclaredPortPublication",
    "native local Docker route denied": "LocalDockerRoute",
    "native local Docker socket custody denied": "LocalDockerSocketCustody",
    "native fixed Docker command denied": "FixedDockerCommand",
    "native probe resource identity denied": "ProbeResourceIdentity",
    "native probe endpoint denied": "ProbeEndpoint",
    "native probe endpoint identity denied": "ProbeEndpointIdentity",
    "native image volume inventory denied": "ImageVolumeInventory",
    "native anonymous volume custody denied": "AnonymousVolumeCustody",
    "native API path denied": "ApiPath",
    "native API deadline exhausted": "ApiDeadline",
    "native API response exceeded bound": "ApiResponseBound",
    "native API assertion denied": "ApiAssertion",
    "native API private-material disclosure denied": "ApiPrivateMaterialDisclosure",
    "native image identity denied": "ImageIdentity",
    "native image declared volumes denied": "ImageDeclaredVolumes",
    "native network creation response denied": "NetworkCreationResponse",
    "native creation response denied": "CreationResponse",
    "native image entrypoint substitution denied": "ImageEntrypointSubstitution",
    "native readiness denied": "Readiness",
    "native synthetic token denied": "SyntheticToken",
    "native closed operation reply denied": "ClosedOperationReply",
    "native concurrent replay denied": "ConcurrentReplay",
    "native actual secret inventory denied": "ActualSecretInventory",
    "native restart resource identity denied": "RestartResourceIdentity",
    "native restart readiness denied": "RestartReadiness",
    "native restart no-write replay denied": "RestartNoWriteReplay",
    "native repository identity reuse denied": "RepositoryIdentityReuse",
    "native revoked operation resurrected": "RevokedOperationResurrection",
    "native unrelated secret compatibility denied": "UnrelatedSecretCompatibility",
    "native retained resource cleanup denied": "RetainedResourceCleanup",
    "native network cleanup has active consumers": "NetworkCleanupActiveConsumers",
}


def denial_code(error: Exception) -> str:
    # Never render an exception: even parser diagnostics can carry private input.
    if type(error) is RuntimeError and len(error.args) == 1 and type(error.args[0]) is str:
        return DENIAL_CODES.get(error.args[0], "UnknownDenied")
    return "UnknownDenied"


def main() -> int:
    global DEADLINE, LOCAL_DOCKER
    parser = argparse.ArgumentParser()
    parser.add_argument("--image", required=True)
    parser.add_argument("--source", required=True)
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--variant", choices=["rootful", "rootless"], required=True)
    parser.add_argument("--arch", choices=["amd64", "arm64"], required=True)
    args = parser.parse_args()
    if not re.fullmatch(r"sha256:[0-9a-f]{64}", args.image) or not re.fullmatch(r"[0-9a-f]{40}", args.source) or not re.fullmatch(r"[1-9][0-9]{0,19}", args.run_id):
        parser.error("invalid immutable native binding")
    DEADLINE = time.monotonic() + 600
    LOCAL_DOCKER = None
    try:
        LOCAL_DOCKER = LocalDockerRoute()
        result = acceptance(args.image, args.source, args.run_id, args.variant, args.arch)
        disabled = acceptance(args.image, args.source, args.run_id, args.variant, args.arch, actions_enabled=False)
        result["checks"].extend(disabled["checks"])
        LOCAL_DOCKER.check()
        print(json.dumps(result, sort_keys=True, separators=(",", ":")))
        return 0
    except Exception as error:
        # CLI/API exception text can contain synthetic private material.
        print(json.dumps({"event": "forgejo-secret-pair-native-denied", "code": denial_code(error)},
                         sort_keys=True, separators=(",", ":")))
        return 1
    finally:
        if LOCAL_DOCKER is not None:
            LOCAL_DOCKER.close()
            LOCAL_DOCKER = None


if __name__ == "__main__":
    raise SystemExit(main())
