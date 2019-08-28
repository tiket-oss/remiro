# Remiro End-to-End Testing

## Test Methodology

`e2e.py` builds and spins up new 6 Docker containers for each test cases:

- `remiro` : testing subject
- `redis-src`: source Redis server
- `redis-dst`: destination Redis server
- `redis-src-expected`: expected source Redis server
- `redis-dst-expected`: expected destination Redis server
- `redis-rdb-tools`: a tool to compare two dump files of Redis data (.rdb) (<https://github.com/sripathikrishnan/redis-rdb-tools>)

then, `e2e.py` does following tasks while running a test case :

- populate setup data to `redis-src` and `redis-dst`
- populate expected data to `redis-src-expected` and `redis-dst-expected`
- run a test to `remiro` by using Redis Python client [redis-py](https://github.com/andymccurdy/redis-py)
- run `SAVE` command for each Redis server containers to get Redis data dump files: `dump-redis-src.rdb`, `dump-redis-dst.rdb`, `dump-redis-src-expected.rdb`, `dump-redis-dst-expected.rdb`
- by using `redis-rdb-tools`, compare:
    1) `dump-redis-src.rdb` & `dump-redis-src-expected.rdb`
    2) `dump-redis-dst.rdb` & `dump-redis-dst-expected.rdb`
