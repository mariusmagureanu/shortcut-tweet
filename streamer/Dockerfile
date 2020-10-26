FROM alpine
LABEL application=streamer

RUN mkdir /lib64 \
    && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

COPY streamer/bin/streamer /

CMD ["./streamer"]
