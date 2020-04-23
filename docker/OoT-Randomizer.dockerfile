FROM python:3.8-buster

WORKDIR /opt/oot-randomizer
ENTRYPOINT ["/opt/oot-randomizer/OoTRandomizer.py"]

COPY ./ /opt/oot-randomizer

RUN set -eux; \
    mkdir /opt/oot-randomizer/Logs; \
    chmod a+rwX /opt/oot-randomizer/Logs; \
    python -m compileall /opt/oot-randomizer
