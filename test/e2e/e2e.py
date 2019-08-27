import docker
import tempfile
import tarfile
import os
from pprint import pprint


def simple_tar(path):
    f = tempfile.NamedTemporaryFile()
    t = tarfile.open(mode="w", fileobj=f)

    abs_path = os.path.abspath(path)
    t.add(abs_path, arcname=os.path.basename(path), recursive=False)

    t.close()
    f.seek(0)
    return f


if __name__ == "__main__":
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
        path="docker-redis-rdb-tools", tag="redis-rdb-tools"
    )
    for log in rdb_tools_log:
        print(log)

    pprint(rdb_tools_image)
    # print('attrs:', rdb_tools_image.attrs)
    # print('tags:', rdb_tools_image.tags)
    # print('id:', rdb_tools_image.id)
    # print('short_id:', rdb_tools_image.short_id)

    print("Building 'remiro' image ...")
    remiro_image, remiro_log = client.images.build(path="../../", tag="remiro")
    for log in remiro_log:
        print(log)

    pprint(remiro_image)

    print("Creating 'e2e-test-network' ...")
    e2e_test_network = client.networks.create("e2e-test-network")
    pprint(e2e_test_network)

    print("Creating 'e2e-test-volume' ...")
    e2e_test_volume = client.volumes.create("e2e-test-volume")
    pprint(e2e_test_volume)

    print("Creating containers ...")

    rdb_tools_container = client.containers.run(
        image=rdb_tools_image.id,
        detach=True,
        command="--help",
        network=e2e_test_network.name,
        volumes={e2e_test_volume.name: {"bind": "/data", "mode": "rw"}},
    )
    print("rdb-tools: ", rdb_tools_container)

    for log in rdb_tools_container.logs(stream=True):
        print(log)

    rdb_tools_container.stop()
    rdb_tools_container.remove()

    # Remove images, network
    print("Removing images, network, volume ...")
    client.images.remove(image=rdb_tools_image.id)
    client.images.remove(image=remiro_image.id)
    e2e_test_network.remove()
    e2e_test_volume.remove()
