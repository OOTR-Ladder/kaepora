FROM python:3.8-alpine

WORKDIR /opt/oot-randomizer
ENTRYPOINT ["/opt/oot-randomizer/OoTRandomizer.py"]

COPY ./ /opt/oot-randomizer
