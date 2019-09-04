import os
import random
import string
import sys
import tarfile
import tempfile
import threading
import time
from datetime import datetime
from pprint import pprint

import docker
import redis

REDIS_VERSION = "redis:5.0.5"

REMIRO_PORT = 6400
REDIS_SRC_PORT = 6410
REDIS_DST_PORT = 6420
REDIS_SRC_EXPECTED_PORT = 6411
REDIS_DST_EXPECTED_PORT = 6421

REDIS_SRC_DUMP = "redis_src_dump.rdb"
REDIS_DST_DUMP = "redis_dst_dump.rdb"
REDIS_SRC_EXPECTED_DUMP = "redis_src_expected_dump.rdb"
REDIS_DST_EXPECTED_DUMP = "redis_dst_expected_dump.rdb"

DEFAULT_BIND_PATH = "/data"
DEFAULT_BIND_VOLUME = {"bind": DEFAULT_BIND_PATH, "mode": "rw"}

REMIRO_CONFIG_DEFAULT = {"delete_on_get": "true", "delete_on_set": "true"}
REMIRO_CONFIG_TEMPLATE = """
DeleteOnGet = {delete_on_get}
DeleteOnSet = {delete_on_set}

[Source]
Addr = {src_addr}

[Destination]
Addr = {dst_addr}
"""

TEST_CASES = [
    {
        "id": "001",
        "name": """
        [Given] a key is available in "destination"
        [When] a GET request for the key is received
        [Then] GET and return the key value from "destination
        """,
        "test": {
            # "given_config": {"delete_on_get": "true", "delete_on_set": "true"},
            "given_data": {
                "src": [],
                "dst": [
                    {"set": {"name": "foo", "value": "bar"}},
                    {"set": {"name": "roo", "value": "car"}},
                ],
            },
            "when_req_then_resp": [
                {"req": {"set": {"name": "foo", "value": "car"}}, "resp": "HALO RESP"},
                {
                    "req": {"set": {"name": "foo", "value": "wherel"}},
                    "resp": "HALO RESP WHERE",
                },
            ],
            "then_data": {
                "src": [],
                "dst": [
                    {"set": {"name": "foo", "value": "bar"}},
                    {"set": {"name": "roo", "value": "car"}},
                ],
            },
        },
    }
]


def simple_tar(path):
    f = tempfile.NamedTemporaryFile()
    t = tarfile.open(mode="w", fileobj=f)

    abs_path = os.path.abspath(path)
    t.add(abs_path, arcname=os.path.basename(path), recursive=False)

    t.close()
    f.seek(0)
    return f


def async_print_container_log(container):
    x = threading.Thread(target=print_container_log, args=(container,), daemon=True)
    x.start()


def print_container_log(container):
    for log in container.logs(stream=True):
        print("{}>{}".format(container.name, log))


def current_time():
    # return datetime.now().strftime("%Y%m%d-%H%M%S-%f")
    return time.time()


def random_string(length=8):
    return "".join(
        [random.choice(string.ascii_letters + string.digits) for n in range(length)]
    )


def random_id():
    return "{:0>3}".format(random.randrange(999))


def run_container(client, name, image, command, network, volumes, ports=None):
    print("Creating container: {}".format(name))
    container = client.containers.run(
        name=name,
        image=image,
        detach=True,
        command=command,
        network=network,
        volumes=volumes,
        ports=ports,
    )
    pprint(container.attrs)
    return container


def run_test(client, api_client, remiro_image, rdb_tools_image, e2e_id, test_case):
    # tc_id = "tid{}".format(random_id())
    tc_id = test_case["id"]
    tc_test = test_case["test"]

    print("tc_id: {} test_name: {}".format(tc_id, test_case["name"]))

    print("Creating 'e2e-T-network' ...")
    e2e_test_network = client.networks.create(
        "e2e-T-network-{}-{}".format(e2e_id, tc_id)
    )
    pprint(e2e_test_network)

    print("Creating 'e2e-T-volume' ...")
    e2e_test_volume = client.volumes.create("e2e-T-volume-{}-{}".format(e2e_id, tc_id))
    pprint(e2e_test_volume.attrs)

    # === setup volume container: intermediary container to copy files from host to volume ===
    client.images.pull("hello-world:latest")
    e2e_test_volume_container = client.containers.create(
        name="e2e-T-volume-container-{}-{}".format(e2e_id, tc_id),
        image="hello-world",
        volumes={e2e_test_volume.name: DEFAULT_BIND_VOLUME},
    )
    # ===
    print("Creating containers ...")

    redis_src_container = run_container(
        client=client,
        name="redis-src-{}-{}".format(e2e_id, tc_id),
        image=REDIS_VERSION,
        command="redis-server --dir {} --dbfilename {}".format(
            DEFAULT_BIND_PATH, REDIS_SRC_DUMP
        ),
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: DEFAULT_BIND_VOLUME},
        ports={"6379/tcp": REDIS_SRC_PORT},
    )
    async_print_container_log(redis_src_container)

    redis_dst_container = run_container(
        client,
        name="redis-dst-{}-{}".format(e2e_id, tc_id),
        image=REDIS_VERSION,
        command="redis-server --dir {} --dbfilename {}".format(
            DEFAULT_BIND_PATH, REDIS_DST_DUMP
        ),
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: DEFAULT_BIND_VOLUME},
        ports={"6379/tcp": REDIS_DST_PORT},
    )
    async_print_container_log(redis_dst_container)

    # Redis Expected Containers
    redis_src_expected_container = run_container(
        client=client,
        name="redis-src-expected-{}-{}".format(e2e_id, tc_id),
        image=REDIS_VERSION,
        command="redis-server --dir {} --dbfilename {}".format(
            DEFAULT_BIND_PATH, REDIS_SRC_EXPECTED_DUMP
        ),
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: DEFAULT_BIND_VOLUME},
        ports={"6379/tcp": REDIS_SRC_EXPECTED_PORT},
    )
    async_print_container_log(redis_src_container)

    redis_dst_expected_container = run_container(
        client=client,
        name="redis-dst-expected-{}-{}".format(e2e_id, tc_id),
        image=REDIS_VERSION,
        command="redis-server --dir {} --dbfilename {}".format(
            DEFAULT_BIND_PATH, REDIS_DST_EXPECTED_DUMP
        ),
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: DEFAULT_BIND_VOLUME},
        ports={"6379/tcp": REDIS_DST_EXPECTED_PORT},
    )
    async_print_container_log(redis_dst_container)

    print("Creating remiro container ...")

    print("NETWORK:")
    e2e_test_network.reload()
    pprint(e2e_test_network.attrs)

    redis_src_ip = e2e_test_network.attrs["Containers"][redis_src_container.id][
        "IPv4Address"
    ][:-3]
    redis_dst_ip = e2e_test_network.attrs["Containers"][redis_dst_container.id][
        "IPv4Address"
    ][:-3]

    # === Copy Remiro Config File
    given_config = tc_test.get("given_config", REMIRO_CONFIG_DEFAULT)

    remiro_config = REMIRO_CONFIG_TEMPLATE.format(
        **given_config,
        src_addr='"{}:{}"'.format(redis_src_ip, 6379),
        dst_addr='"{}:{}"'.format(redis_dst_ip, 6379),
    )
    print("remiro_config:")
    print(remiro_config)

    temp_dir = tempfile.TemporaryDirectory()
    remiro_config_file = open(os.path.join(temp_dir.name, "config.toml"), mode="w+")
    try:
        remiro_config_file.writelines(remiro_config)
    finally:
        remiro_config_file.close()
    print("remiro_config_file: ", remiro_config_file.name)

    status_put_archive = api_client.put_archive(
        e2e_test_volume_container.name,
        DEFAULT_BIND_PATH,
        simple_tar(remiro_config_file.name),
    )
    print("STATUS PUT_ARCHIVE: {}".format(status_put_archive))

    # ===

    remiro_config_path = "{}/config.toml".format(DEFAULT_BIND_PATH)

    remiro_container = run_container(
        client=client,
        name="remiro-{}-{}".format(e2e_id, tc_id),
        image=remiro_image.id,
        command="-h 0.0.0.0 -p {} -c {}".format(REMIRO_PORT, remiro_config_path),
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: DEFAULT_BIND_VOLUME},
        ports={"{}/tcp".format(REMIRO_PORT): REMIRO_PORT},
    )
    async_print_container_log(remiro_container)

    print("=== init redis client ===")

    remiro_client = redis.Redis(host="127.0.0.1", port=REMIRO_PORT)
    redis_src_client = redis.Redis(host="127.0.0.1", port=REDIS_SRC_PORT)
    redis_dst_client = redis.Redis(host="127.0.0.1", port=REDIS_DST_PORT)
    redis_src_expected_client = redis.Redis(
        host="127.0.0.1", port=REDIS_SRC_EXPECTED_PORT
    )

    redis_dst_expected_client = redis.Redis(
        host="127.0.0.1", port=REDIS_DST_EXPECTED_PORT
    )
    # ===
    print("=== populate given_data and then_data ===")

    if "given_data" in tc_test:
        given_data = tc_test["given_data"]
        redis_client_call_inbulk_then_save(redis_src_client, given_data["src"])
        redis_client_call_inbulk_then_save(redis_dst_client, given_data["dst"])

    if "then_data" in tc_test:
        then_data = tc_test["then_data"]
        redis_client_call_inbulk_then_save(redis_src_expected_client, then_data["src"])
        redis_client_call_inbulk_then_save(redis_dst_expected_client, then_data["dst"])
    # ===

    if "when_req_then_resp" in tc_test:
        for when_req_then_resp in tc_test["when_req_then_resp"]:
            when_req_cmd, when_req_args = list(when_req_then_resp["req"].items())[0]
            then_resp = when_req_then_resp["resp"]

            got_resp = redis_client_call(remiro_client, when_req_cmd, when_req_args)
            print(f"THEN_RESP={then_resp} GOT_RESP={got_resp}")

    rdb_tools_container = run_container(
        client=client,
        name="rdb-tools-{}-{}".format(e2e_id, tc_id),
        image=rdb_tools_image.id,
        command="""
        /bin/sh -c "
        rdb --command diff /data/{} | sort > /data/{}.txt;
        rdb --command diff /data/{} | sort > /data/{}.txt;
        rdb --command diff /data/{} | sort > /data/{}.txt;
        rdb --command diff /data/{} | sort > /data/{}.txt;
        diff /data/{}.txt /data/{}.txt && diff /data/{}.txt /data/{}.txt;
        echo $?
        "
        """.format(
            REDIS_SRC_DUMP,
            REDIS_SRC_DUMP,
            REDIS_DST_DUMP,
            REDIS_DST_DUMP,
            REDIS_SRC_EXPECTED_DUMP,
            REDIS_SRC_EXPECTED_DUMP,
            REDIS_DST_EXPECTED_DUMP,
            REDIS_DST_EXPECTED_DUMP,
            REDIS_SRC_DUMP,
            REDIS_SRC_EXPECTED_DUMP,
            REDIS_DST_DUMP,
            REDIS_DST_EXPECTED_DUMP,
        ),
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: DEFAULT_BIND_VOLUME},
    )

    last_cmd_status = str(
        list(rdb_tools_container.logs(stream=True))[-1], "utf-8"
    ).strip()

    is_expected = last_cmd_status == "0"

    # === Delete containers, volume, network.
    e2e_test_volume_container.stop()
    e2e_test_volume_container.remove()

    rdb_tools_container.stop()
    rdb_tools_container.remove()

    remiro_container.stop()
    remiro_container.remove()

    redis_src_container.stop()
    redis_src_container.remove()

    redis_dst_container.stop()
    redis_dst_container.remove()

    redis_src_expected_container.stop()
    redis_src_expected_container.remove()

    redis_dst_expected_container.stop()
    redis_dst_expected_container.remove()
    # ===

    e2e_test_network.remove()
    e2e_test_volume.remove()

    return is_expected


def redis_client_call(redis_client, command, args):
    cmd_func = getattr(redis_client, command)
    cmd_func(**args)


def redis_client_call_inbulk_then_save(redis_client, list_command):
    for cmd_n_args in list_command:
        for cmd, args in cmd_n_args.items():
            redis_client_call(redis_client, cmd, args)
            redis_client_call(redis_client, "save", {})


def main():
    e2e_id = "e2e{}".format(random_id())
    print("e2e_id: {}".format(e2e_id))

    client = docker.from_env()

    print("VERSION: ", client.version())

    # Use api_client to 'docker cp' files from host to container
    api_client = docker.APIClient()
    print("API Client VERSION: ", api_client.version())

    # pprint(vars(client.containers.list()))
    for c in client.containers.list():
        # pprint(vars(c.attrs))
        pprint(c.attrs["Name"])

    print("Building 'redis-rdb-tools' image ...")
    rdb_tools_image, rdb_tools_log = client.images.build(
        path="docker-redis-rdb-tools", tag="redis-rdb-tools-{}".format(e2e_id)
    )
    for log in rdb_tools_log:
        print(log)
    pprint(rdb_tools_image)

    print("Building 'remiro' image ...")
    remiro_image, remiro_image_log = client.images.build(
        path="../../", tag="remiro-{}".format(e2e_id)
    )
    for log in remiro_image_log:
        print(log)
    pprint(remiro_image)

    # ==== Run a T ===

    for tc in TEST_CASES:

        is_expected = run_test(
            client=client,
            api_client=api_client,
            remiro_image=remiro_image,
            rdb_tools_image=rdb_tools_image,
            e2e_id=e2e_id,
            test_case=tc,
        )

        if is_expected:
            print("Test PASSED: [{}] {}".format(tc["id"], tc["name"]))
            continue
        else:
            print("Test FAILED: [{}] {}".format(tc["id"], tc["name"]))
            sys.exit(1)

    # Remove images, network
    # print("Removing images, network, volume ...")

    # client.images.remove(image=rdb_tools_image.id)
    # client.images.remove(image=remiro_image.id)

    # print("VOLUME ID: {}".format(e2e_test_volume.id))


if __name__ == "__main__":
    main()