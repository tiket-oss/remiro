# Remiro

Remiro is a service that serves as a proxy in front of multiple Redis server.

## Rationale

Suppose you have a single Redis server, it holds various types of data throughout your system, and you decided to move a group of similar types of data into a new Redis server. Problem is, you didn't have any convention of naming the key to indicate its type.

There are of course multiple solution which can be done with existing tools:

1. Duplicate the Redis server and then have the system use different server based on their cases.
2. Create a new empty Redis server, and have the system populate it over time.

The drawbacks from such approach is that you don't have a clean distribution of data. Approach number 1 will clutter both redis server with unneeded data since we can't assume that the data has expiration deadline set, and approach number 2 will hamper the performance of the running system that should use the new redis server, since then it will have to look up the keys which will be non-existent, and then populate it with the new value it derived from somewhere else

### Solution

Remiro can act as an intermediary for populating the new redis server over time without sacrificing the system that still needs the data. It works by these following assumption:

- There are two sets of redis server, the **source** and the **destination**
- Suppose **source** redis has two sets of data: the **data to keep** and the **data to move**
- We want to move **data to move** from **source** to **destination**, while ignoring the **data to keep**
- *IMPORTANT*: The system that will request through Remiro is the one that *only* needs the **data to move**

With the assumption in place, we can establish a mechanism in which Remiro will adhere to:

- If retrieval request came through remiro:
  - Remiro will check the data existence on **destination**
    - If exists:
      - Remiro will return the data from **destination** to the requester, *process ends*
    - If not exists, *process continues*
  - Remiro will check the data existence in **source**
    - If exists:
      - Remiro will return the data from **source** to the requester
      - Remiro will copy the data from **source** to **destination**
      - (optional) Remiro will delete the data from **source**
      - *process ends*
    - If not exists, Remiro *returns* any response that it gets from **source**
- If assignment request came through remiro:
  - Remiro will write the data to **destination**
  - (optional) Remiro will delete the data with the same key from **source**
- If any other request came through remiro:
  - Simply proxy the request to the **destination**
