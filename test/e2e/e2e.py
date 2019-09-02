import docker
import tempfile
import tarfile
import random
import string
import os
import threading
import time
from datetime import datetime
import redis
from pprint import pprint

redis_version = "redis:5.0.5"

remiro_port = 6400
redis_src_port = 6410
redis_dst_port = 6420
redis_src_expected_port = 6411
redis_dst_expected_port = 6421

redis_src_dump = "redis_src_dump.rdb"
redis_dst_dump = "redis_dst_dump.rdb"
redis_src_expected_dump = "redis_src_expected_dump.rdb"
redis_dst_expected_dump = "redis_dst_expected_dump.rdb"

default_bind_path = "/data"
default_bind_volume = {"bind": default_bind_path, "mode": "rw"}

remiro_config_template = """
DeleteOnGet = {delete_on_get}
DeleteOnSet = {delete_on_set}

[Source]
Addr = {src_addr}

[Destination]
Addr = {dst_addr}
"""

test_cases = [
    {
        "id": "1234",
        "name": "",
        "test": {
            "given": {"src": [], "dst": []},
            "when": [],
            "then": {"src": [], "dst": []},
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
    # test_id = "tid{}".format(random_id())
    test_id = test_case["id"]
    print("test_id: {} test_name: {}".format(test_id, test_case["name"]))

    print("Creating 'e2e-test-network' ...")
    e2e_test_network = client.networks.create(
        "e2e-test-network-{}-{}".format(e2e_id, test_id)
    )
    pprint(e2e_test_network)

    print("Creating 'e2e-test-volume' ...")
    e2e_test_volume = client.volumes.create(
        "e2e-test-volume-{}-{}".format(e2e_id, test_id)
    )
    pprint(e2e_test_volume.attrs)

    # === setup volume container: intermediary container to copy files from host to volume ===
    client.images.pull("hello-world:latest")
    e2e_test_volume_container = client.containers.create(
        name="e2e-test-volume-container-{}-{}".format(e2e_id, test_id),
        image="hello-world",
        volumes={e2e_test_volume.name: default_bind_volume},
    )
    # ===
    print("Creating containers ...")

    redis_src_container = run_container(
        client=client,
        name="redis-src-{}-{}".format(e2e_id, test_id),
        image=redis_version,
        command="redis-server --dir {} --dbfilename {}".format(
            default_bind_path, redis_src_dump
        ),
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: default_bind_volume},
        ports={"6379/tcp": redis_src_port},
    )
    async_print_container_log(redis_src_container)

    redis_dst_container = run_container(
        client,
        name="redis-dst-{}-{}".format(e2e_id, test_id),
        image=redis_version,
        command="redis-server --dir {} --dbfilename {}".format(
            default_bind_path, redis_dst_dump
        ),
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: default_bind_volume},
        ports={"6379/tcp": redis_dst_port},
    )
    async_print_container_log(redis_dst_container)

    # Redis Expected Containers
    redis_src_expected_container = run_container(
        client=client,
        name="redis-src-expected-{}-{}".format(e2e_id, test_id),
        image=redis_version,
        command="redis-server --dir {} --dbfilename {}".format(
            default_bind_path, redis_src_expected_dump
        ),
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: default_bind_volume},
        ports={"6379/tcp": redis_src_expected_port},
    )
    async_print_container_log(redis_src_container)

    redis_dst_expected_container = run_container(
        client=client,
        name="redis-dst-expected-{}-{}".format(e2e_id, test_id),
        image=redis_version,
        command="redis-server --dir {} --dbfilename {}".format(
            default_bind_path, redis_dst_expected_dump
        ),
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: default_bind_volume},
        ports={"6379/tcp": redis_dst_expected_port},
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

    remiro_config = remiro_config_template.format(
        delete_on_set="false",
        delete_on_get="false",
        src_addr='"{}:{}"'.format(redis_src_ip, 6379),
        dst_addr='"{}:{}"'.format(redis_dst_ip, 6379),
    )
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
        default_bind_path,
        simple_tar(remiro_config_file.name),
    )
    print("STATUS PUT_ARCHIVE: {}".format(status_put_archive))

    # ===

    remiro_config_path = "{}/config.toml".format(default_bind_path)

    remiro_container = run_container(
        client=client,
        name="remiro-{}-{}".format(e2e_id, test_id),
        image=remiro_image.id,
        command="-h 0.0.0.0 -p {} -c {}".format(remiro_port, remiro_config_path),
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: default_bind_volume},
        ports={"{}/tcp".format(remiro_port): remiro_port},
    )
    async_print_container_log(remiro_container)

    print("=== Test with Redis Client")
    test = test_case["test"]

    remiro_client = redis.Redis(host="127.0.0.1", port=remiro_port)
    print(remiro_client.set("foo", "bar"))
    print(remiro_client.get("foo"))

    redis_src_client = redis.Redis(host="127.0.0.1", port=redis_src_port)
    redis_src_client.save()

    redis_dst_client = redis.Redis(host="127.0.0.1", port=redis_dst_port)
    redis_dst_client.save()

    redis_src_expected_client = redis.Redis(
        host="127.0.0.1", port=redis_src_expected_port
    )
    redis_src_expected_client.save()

    redis_dst_expected_client = redis.Redis(
        host="127.0.0.1", port=redis_dst_expected_port
    )
    redis_dst_expected_client.save()
    # ===

    rdb_tools_container = run_container(
        client=client,
        name="rdb-tools-{}-{}".format(e2e_id, test_id),
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
            redis_src_dump,
            redis_src_dump,
            redis_dst_dump,
            redis_dst_dump,
            redis_src_expected_dump,
            redis_src_expected_dump,
            redis_dst_expected_dump,
            redis_dst_expected_dump,
            redis_src_dump,
            redis_src_expected_dump,
            redis_dst_dump,
            redis_dst_expected_dump,
        ),
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: default_bind_volume},
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


if __name__ == "__main__":

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

    # ==== Run a test ===

    for tc in test_cases:

        is_expected = run_test(
            client=client,
            api_client=api_client,
            remiro_image=remiro_image,
            rdb_tools_image=rdb_tools_image,
            e2e_id=e2e_id,
            test_case=tc,
        )

    # Remove images, network
    # print("Removing images, network, volume ...")

    # client.images.remove(image=rdb_tools_image.id)
    # client.images.remove(image=remiro_image.id)

    # print("VOLUME ID: {}".format(e2e_test_volume.id))
