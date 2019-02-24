FROM alpine
HEALTHCHECK --retries=3 --timeout=1s CMD /bin/sh -c 'echo "hello"'
