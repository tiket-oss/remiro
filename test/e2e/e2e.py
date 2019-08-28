import docker
import tempfile
import tarfile
import random
import string
import os
import threading
from datetime import datetime
import time
from pprint import pprint

remiro_config_template = """
DeleteOnGet = {delete_on_get}
DeleteOnSet = {delete_on_set}

[Source]
Addr = {src_addr}

[Destination]
Addr = {dst_addr}
"""


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
    return "{:>3}".format(random.randrange(999))


if __name__ == "__main__":
    # remiro_config = remiro_config_template.format(
    #     delete_on_set="true",
    #     delete_on_get="false",
    #     src_addr='"127.0.0.1:3456"',
    #     dst_addr='"127.0.0.1:4567"',
    # )
    # print(remiro_config)

    e2e_id = "e2e{}".format(random_id())
    print("e2e_id: {}".format(e2e_id))

    test_id = "tid{}".format(random_id())
    print("test_id: {}".format(test_id))

    remiro_port = 6400
    redis_src_port = 6410
    redis_dst_port = 6420
    redis_src_expected_port = 6411
    redis_dst_expected_port = 6421

    default_bind_path = "/data"
    default_bind_volume = {"bind": default_bind_path, "mode": "rw"}

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

    print("Creating 'e2e-test-network' ...")
    e2e_test_network = client.networks.create("e2e-test-network-{}".format(e2e_id))
    pprint(e2e_test_network)

    print("Creating 'e2e-test-volume' ...")
    e2e_test_volume = client.volumes.create(
        "e2e-test-volume-{}-{}".format(e2e_id, test_id)
    )
    pprint(e2e_test_volume)

    print("Creating containers ...")

    print("Creating rdb-tools container ...")
    rdb_tools_container_name = "rdb-tools-{}-{}".format(e2e_id, test_id)
    rdb_tools_container = client.containers.run(
        name=rdb_tools_container_name,
        image=rdb_tools_image.id,
        detach=True,
        command="--help",
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: default_bind_volume},
    )
    print("rdb-tools: ", rdb_tools_container)

    async_print_container_log(rdb_tools_container)

    print("Creating remiro container ...")
    remiro_container_name = "remiro-{}-{}".format(e2e_id, test_id)
    remiro_container = client.containers.run(
        name=remiro_container_name,
        image=remiro_image.id,
        detach=True,
        command="-h 0.0.0.0 -p {}".format(remiro_port),
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: default_bind_volume},
        ports={"{}/tcp".format(remiro_port): remiro_port},
    )
    print("remiro: ", remiro_container)

    async_print_container_log(remiro_container)

    # Delete containers. (Temporarily)
    rdb_tools_container.stop()
    rdb_tools_container.remove()

    # remiro_container.stop()
    # remiro_container.remove()

    # Remove images, network
    print("Removing images, network, volume ...")
    client.images.remove(image=rdb_tools_image.id)
    # client.images.remove(image=remiro_image.id)
    # e2e_test_network.remove()
    # e2e_test_volume.remove()
