import docker
from pprint import pprint

client = docker.from_env()

# pprint(vars(client.containers.list()))
for c in client.containers.list():
    # pprint(vars(c.attrs))
    pprint(c.attrs["Name"])
