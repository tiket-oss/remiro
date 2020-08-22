import copy
import os
import traceback
import random
import string
import sys
import tarfile
import tempfile
import threading
import time
from pprint import pprint
import docker
import redis
from scenario import test_cases

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

REMIRO_CONFIG_DEFAULT = {
    "delete_on_get": "true",
    "delete_on_set": "false",
    "password": '""',
    "src_password": '""',
    "dst_password": '""',
}
REMIRO_CONFIG_TEMPLATE = """
DeleteOnGet = {delete_on_get}
DeleteOnSet = {delete_on_set}
Password = {password}

[Source]
Addr = {src_addr}
Password = {src_password}

[Destination]
Addr = {dst_addr}
Password = {dst_password}
"""


def create_tar_file(path):
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
        print(f"{container.name}>{log}")


def current_time():
    # return datetime.now().strftime("%Y%m%d-%H%M%S-%f")
    return time.time()


def random_string(length=8):
    return "".join(
        [random.choice(string.ascii_letters + string.digits) for n in range(length)]
    )


def random_id():
    return "{:0>3}".format(random.randrange(999))


def redis_client_call(redis_client, command, args):
    args_tupled = args
    if not isinstance(args, tuple):
        args_tupled = (args,)

    return redis_client.execute_command(command, *args_tupled)


def redis_client_call_inbulk(redis_client, list_command):
    for cmd_n_args in list_command:
        for cmd, args in cmd_n_args.items():
            redis_client_call(redis_client, cmd, args)


def run_container(client, name, image, command, network, volumes, ports=None):
    print(f"Creating container: {name}")
    container = client.containers.run(
        name=name,
        image=image,
        detach=True,
        command=command,
        network=network,
        volumes=volumes,
        ports=ports,
    )
    # pprint(container.attrs)
    return container


class TestCase:
    def __init__(
        self, client, api_client, remiro_image, rdb_tools_image, e2e_id, test_case
    ):
        self.client = client
        self.api_client = api_client
        self.remiro_image = remiro_image
        self.rdb_tools_image = rdb_tools_image
        self.e2e_id = e2e_id
        self.test_case = test_case

        self.setup_network_volume()

    def setup_network_volume(self):
        e2e_id = self.e2e_id
        # tc_id = "tid{}".format(random_id())
        tc_id = self.test_case["id"]
        tc_test = self.test_case["test"]

        print("=== Preparing a test ===")
        print(f"tc_id: {tc_id} test_name: {self.test_case['name']}")

        print("Creating 'e2e-test-network' ...")
        e2e_test_network = self.client.networks.create(
            f"e2e-test-network-{e2e_id}-{tc_id}"
        )
        pprint(e2e_test_network)

        print("Creating 'e2e-test-volume' ...")
        e2e_test_volume = self.client.volumes.create(
            f"e2e-test-volume-{e2e_id}-{tc_id}"
        )
        pprint(e2e_test_volume)

        # === setup volume container: intermediary container to copy files (remiro config file, in this case) from host to volume ===
        self.client.images.pull("hello-world:latest")
        e2e_test_volume_container = self.client.containers.create(
            name=f"e2e-test-volume-container-{e2e_id}-{tc_id}",
            image="hello-world",
            volumes={e2e_test_volume.name: DEFAULT_BIND_VOLUME},
        )
        # ===
        self.e2e_test_network = e2e_test_network
        self.e2e_test_volume = e2e_test_volume
        self.e2e_test_volume_container = e2e_test_volume_container

    def cleanup_network_volume(self):
        self.e2e_test_volume_container.stop()
        self.e2e_test_volume_container.remove()

        self.e2e_test_network.remove()
        self.e2e_test_volume.remove()

    def run_redis_container(self, name, dbfilename, port):
        return run_container(
            client=self.client,
            name=name,
            image=REDIS_VERSION,
            command=f"redis-server --dir {DEFAULT_BIND_PATH} --dbfilename {dbfilename}",
            network=self.e2e_test_network.name,
            volumes={self.e2e_test_volume.name: DEFAULT_BIND_VOLUME},
            ports={"6379/tcp": port},
        )

    def run_remiro_container(self, name, remiro_config_path, port):
        print(f"{name} - {remiro_config_path} - {port}")
        return run_container(
            client=self.client,
            name=name,
            image=self.remiro_image.id,
            command=f"-h 0.0.0.0 -p {port} -c {remiro_config_path}",
            network=self.e2e_test_network.name,
            volumes={self.e2e_test_volume.name: DEFAULT_BIND_VOLUME},
            ports={"{}/tcp".format(port): port},
        )

    def run_rdb_tools_container(self, name, command):
        return run_container(
            client=self.client,
            name=name,
            image=self.rdb_tools_image.id,
            command=command,
            network=self.e2e_test_network.name,
            volumes={self.e2e_test_volume.name: DEFAULT_BIND_VOLUME},
        )

    def create_remiro_config(self, redis_src_container, redis_dst_container):
        tc_id = self.test_case["id"]
        tc_test = self.test_case["test"]
        # print("NETWORK:")
        self.e2e_test_network.reload()
        # pprint(e2e_test_network.attrs)

        redis_src_ip = self.e2e_test_network.attrs["Containers"][
            redis_src_container.id
        ]["IPv4Address"][:-3]
        redis_dst_ip = self.e2e_test_network.attrs["Containers"][
            redis_dst_container.id
        ]["IPv4Address"][:-3]

        # === Copy Remiro Config File
        given_config = copy.deepcopy(REMIRO_CONFIG_DEFAULT)
        given_config.update(tc_test.get("given_config", {}))

        remiro_config = REMIRO_CONFIG_TEMPLATE.format(
            **given_config,
            src_addr=f'"{redis_src_ip}:{6379}"',
            dst_addr=f'"{redis_dst_ip}:{6379}"',
        )
        print(f"{tc_id} remiro_config:")
        print(remiro_config)
        return remiro_config

    def copy_remiro_config_to_volume(self, remiro_config, bind_path):

        temp_dir = tempfile.TemporaryDirectory()
        remiro_config_file = open(os.path.join(temp_dir.name, "config.toml"), mode="w+")
        try:
            remiro_config_file.writelines(remiro_config)
        finally:
            remiro_config_file.close()
        print("remiro_config_file: ", remiro_config_file.name)

        status_put_archive = self.api_client.put_archive(
            self.e2e_test_volume_container.name,
            bind_path,
            create_tar_file(remiro_config_file.name),
        )
        print(f"STATUS PUT_ARCHIVE: {status_put_archive}")

    def run_test(self):
        e2e_id = self.e2e_id
        tc_id = self.test_case["id"]
        tc_test = self.test_case["test"]
        e2e_test_network = self.e2e_test_network

        print("Creating containers ...")
        redis_src_container = self.run_redis_container(
            name=f"redis-src-{e2e_id}-{tc_id}",
            dbfilename=REDIS_SRC_DUMP,
            port=REDIS_SRC_PORT,
        )
        async_print_container_log(redis_src_container)

        redis_dst_container = self.run_redis_container(
            name=f"redis-dst-{e2e_id}-{tc_id}",
            dbfilename=REDIS_DST_DUMP,
            port=REDIS_DST_PORT,
        )
        async_print_container_log(redis_dst_container)

        # === Redis Expected Containers ===
        redis_src_expected_container = self.run_redis_container(
            name=f"redis-src-expected-{e2e_id}-{tc_id}",
            dbfilename=REDIS_SRC_EXPECTED_DUMP,
            port=REDIS_SRC_EXPECTED_PORT,
        )
        async_print_container_log(redis_src_expected_container)

        redis_dst_expected_container = self.run_redis_container(
            name=f"redis-dst-expected-{e2e_id}-{tc_id}",
            dbfilename=REDIS_DST_EXPECTED_DUMP,
            port=REDIS_DST_EXPECTED_PORT,
        )
        async_print_container_log(redis_dst_expected_container)

        remiro_config = self.create_remiro_config(
            redis_src_container=redis_src_container,
            redis_dst_container=redis_dst_container,
        )
        self.copy_remiro_config_to_volume(
            remiro_config=remiro_config, bind_path=DEFAULT_BIND_PATH
        )

        # ===

        remiro_config_path = f"{DEFAULT_BIND_PATH}/config.toml"

        remiro_container = self.run_remiro_container(
            name=f"remiro-{e2e_id}-{tc_id}",
            remiro_config_path=remiro_config_path,
            port=REMIRO_PORT,
        )
        async_print_container_log(remiro_container)

        print("=== init redis clients ===")
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
        rdb_tools_container = None
        rdb_tools_container_log = []
        is_expected_cmd_status = False
        try:

            if "given_data" in tc_test:
                given_data = tc_test["given_data"]
                redis_client_call_inbulk(redis_src_client, given_data["src"])
                redis_client_call_inbulk(redis_dst_client, given_data["dst"])

            if "then_data" in tc_test:
                then_data = tc_test["then_data"]
                redis_client_call_inbulk(redis_src_expected_client, then_data["src"])
                redis_client_call_inbulk(redis_dst_expected_client, then_data["dst"])
            # ===

            list_not_expected_resp = []
            if "when_req_then_resp" in tc_test:
                for when_req_then_resp in tc_test["when_req_then_resp"]:
                    when_req_cmd, when_req_args = list(
                        when_req_then_resp["req"].items()
                    )[0]

                    got_resp = None
                    try:
                        got_resp = redis_client_call(
                            remiro_client, when_req_cmd, when_req_args
                        )
                    except (
                        redis.exceptions.ResponseError,
                        redis.exceptions.AuthenticationError,
                    ) as ex:
                        if "respError" in when_req_then_resp:
                            if when_req_then_resp["respError"] != True:
                                list_not_expected_resp.append(ex)
                        else:
                            list_not_expected_resp.append(
                                f'{ex}. Hint: set "respError": True'
                            )

                    if "resp" in when_req_then_resp:
                        then_resp = when_req_then_resp["resp"]
                        if got_resp != then_resp:
                            msg_err = f"WHEN_REQ_CMD={when_req_cmd} WHEN_REQ_ARGS={when_req_args} THEN_RESP={then_resp} GOT_RESP={got_resp}"
                            list_not_expected_resp.append(msg_err)

            redis_client_call(redis_src_client, "save", ())
            redis_client_call(redis_dst_client, "save", ())
            redis_client_call(redis_src_expected_client, "save", ())
            redis_client_call(redis_dst_expected_client, "save", ())

            rdb_tools_container = self.run_rdb_tools_container(
                name=f"rdb-tools-{e2e_id}-{tc_id}",
                command=f"""
                /bin/sh -c "
                rdb --command diff /data/{REDIS_SRC_DUMP} | sort > /data/{REDIS_SRC_DUMP}.txt;
                rdb --command diff /data/{REDIS_DST_DUMP} | sort > /data/{REDIS_DST_DUMP}.txt;
                rdb --command diff /data/{REDIS_SRC_EXPECTED_DUMP} | sort > /data/{REDIS_SRC_EXPECTED_DUMP}.txt;
                rdb --command diff /data/{REDIS_DST_EXPECTED_DUMP} | sort > /data/{REDIS_DST_EXPECTED_DUMP}.txt;
                diff /data/{REDIS_SRC_DUMP}.txt /data/{REDIS_SRC_EXPECTED_DUMP}.txt && diff /data/{REDIS_DST_DUMP}.txt /data/{REDIS_DST_EXPECTED_DUMP}.txt;
                echo $?
                "
                """,
            )

            rdb_tools_container_log = list(rdb_tools_container.logs(stream=True))
            last_cmd_status = str(rdb_tools_container_log[-1], "utf-8").strip()

            is_expected_cmd_status = last_cmd_status == "0"
        except:
            print(f"=== Exception when running a test: ===")
            traceback.print_exc()
            return False

        finally:
            # === Delete containers, volume, network.

            if rdb_tools_container != None:
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

            self.cleanup_network_volume()
            # ===

        if list_not_expected_resp:
            print("=== got_resp is not matched with then_resp ===")
            for msg_err in list_not_expected_resp:
                print(msg_err)

        if not is_expected_cmd_status:
            print(
                "=== redis db comparison: (src vs src_expected) && (dst vs dst_expected) ==="
            )
            for log in rdb_tools_container_log:
                print(log)

        return (is_expected_cmd_status) and (not list_not_expected_resp)


def main():
    # e2e_id = "e2e{}".format(random_id())
    e2e_id = "e2e"
    print(f"e2e_id: {e2e_id}")

    client = docker.from_env()

    # Use api_client to 'docker cp' files from host to container
    api_client = docker.APIClient()
    print("API Client VERSION: ", api_client.version())

    print("Building 'redis-rdb-tools' image ...")
    rdb_tools_image, rdb_tools_log = client.images.build(
        path="docker-redis-rdb-tools", tag=f"redis-rdb-tools-{e2e_id}"
    )
    for log in rdb_tools_log:
        print(log)
    pprint(rdb_tools_image)

    print("Building 'remiro' image ...")
    remiro_image, remiro_image_log = client.images.build(
        path="../../", tag=f"remiro-{e2e_id}"
    )
    for log in remiro_image_log:
        print(log)
    pprint(remiro_image)

    # ==== Run a Test Case ===

    for tc in test_cases:
        test_case = TestCase(
            client=client,
            api_client=api_client,
            remiro_image=remiro_image,
            rdb_tools_image=rdb_tools_image,
            e2e_id=e2e_id,
            test_case=tc,
        )

        is_expected = test_case.run_test()

        if is_expected:
            print(f"Test PASSED: [{tc['id']}] {tc['name']}")
            continue
        else:
            print(f"Test FAILED: [[{tc['id']}] {tc['name']}")
            sys.exit(1)


if __name__ == "__main__":
    main()
