FROM python:3.5-alpine

RUN apk add gcc musl-dev

RUN pip install  --no-cache-dir rdbtools python-lzf

ENTRYPOINT [ "/usr/local/bin/rdb" ]