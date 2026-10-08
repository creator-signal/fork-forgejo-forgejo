#!/usr/bin/env python3
"""Native-only, isolated HTTP acceptance of the exact fork image pair operation.

The existing qualification/pull-back jobs own this helper. It has no live
provider URL, persistent configuration, release credential, or publication API.
"""
from __future__ import annotations

import argparse
import concurrent.futures
import http.client
import json
import os
import re
import secrets
import selectors
import signal
import subprocess
import time


PURPOSE = "creator-signal-fork-secret-pair-native"
NAMES = ["ZOT_VM_ARTIFACT_PASSWORD", "ZOT_VM_ARTIFACT_USERNAME"]
DEADLINE = 0.0


def remaining(cap: float = 30) -> float:
    value = min(cap, DEADLINE - time.monotonic())
    if value <= 0:
        raise RuntimeError("native acceptance deadline exhausted")
    return value


def command(args: list[str]) -> bytes:
    # Do not inherit publication tokens into the CLI or image entrypoints.
    env = {key: value for key, value in os.environ.items() if key in
           {"PATH", "HOME", "DOCKER_HOST", "DOCKER_TLS_VERIFY", "DOCKER_CERT_PATH"}}
    deadline = time.monotonic() + remaining()
    child = subprocess.Popen(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                             env=env, start_new_session=True)
    output = bytearray()
    total = 0
    try:
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
        if child.poll() is None:
            os.killpg(child.pid, signal.SIGKILL)
            child.wait(timeout=5)
        child.stdout.close()
        child.stderr.close()


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
    bindings = ports.get("3000/tcp")
    if (not isinstance(bindings, list) or len(bindings) != 1
            or bindings[0].get("HostIp") != "127.0.0.1"
            or not re.fullmatch(r"[1-9][0-9]{0,4}", bindings[0].get("HostPort", ""))
            or int(bindings[0]["HostPort"]) > 65535
            or any(binding for port, binding in ports.items() if port != "3000/tcp")):
        raise RuntimeError("native published port inventory denied")
    port_config = host.get("PortBindings", {})
    if (set(port_config) != {"3000/tcp"} or len(port_config["3000/tcp"]) != 1
            or port_config["3000/tcp"][0].get("HostIp") != "127.0.0.1"):
        raise RuntimeError("native declared loopback binding denied")
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


class Provider:
    def __init__(self, port: int):
        self.port = port
        self.token: str | None = None
        self.private_values: set[bytes] = set()

    def request(self, method: str, path: str, body=None, status: int = 200):
        if not path.startswith("/api/") or "\n" in path or "\r" in path:
            raise RuntimeError("native API path denied")
        headers = {"Content-Type": "application/json"}
        if self.token:
            headers["Authorization"] = "token " + self.token
        raw = body if isinstance(body, bytes) else (json.dumps(body).encode() if body is not None else None)
        connection = http.client.HTTPConnection("127.0.0.1", self.port, timeout=remaining(15))
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
                           "--publish", "127.0.0.1::3000",
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
        bindings = value.get("NetworkSettings", {}).get("Ports", {}).get("3000/tcp")
        if not isinstance(bindings, list) or len(bindings) != 1 or bindings[0].get("HostIp") != "127.0.0.1":
            raise RuntimeError("native loopback binding denied")
        provider = Provider(int(bindings[0]["HostPort"]))
        for _ in range(90):
            try:
                provider.request("GET", "/api/healthz")
                break
            except (OSError, RuntimeError):
                time.sleep(min(2, remaining()))
        else:
            raise RuntimeError("native readiness denied")
        command(["docker", "exec", "--user", "git", container, "forgejo", "admin", "user", "create",
                 "--username", "cs-pair-native", "--email", "native@example.invalid", "--random-password",
                 "--must-change-password=false", "--admin"])
        token = command(["docker", "exec", "--user", "git", container, "forgejo", "admin", "user",
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
        own_container(container, image, identity)
        command(["docker", "restart", "--time", "10", container])
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
            command(["docker", "rm", "--force", "--volumes", container])
        if network_id:
            current_network = own_network(network, network_id)
            if current_network.get("Containers"):
                raise RuntimeError("native network cleanup has active consumers")
            command(["docker", "network", "rm", network])


def main() -> int:
    global DEADLINE
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
    try:
        result = acceptance(args.image, args.source, args.run_id, args.variant, args.arch)
        disabled = acceptance(args.image, args.source, args.run_id, args.variant, args.arch, actions_enabled=False)
        result["checks"].extend(disabled["checks"])
        print(json.dumps(result, sort_keys=True, separators=(",", ":")))
        return 0
    except Exception:
        # CLI/API exception text can contain synthetic private material.
        print('{"event":"forgejo-secret-pair-native-denied"}')
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
