FROM node:20-bookworm-slim

RUN apt-get update \
 && apt-get install -y --no-install-recommends strace ca-certificates \
 && rm -rf /var/lib/apt/lists/*

# Pre-fetch the baseline package at BUILD time so analysis runs can be fully
# offline (--network=none). lodash has zero dependencies, which keeps the
# baseline trace as quiet as possible.
RUN mkdir -p /fixtures \
 && npm pack lodash@4.17.21 --pack-destination /fixtures --silent

# Decoy credentials. Fake values, so synthetic test fixtures have something
# realistic to "steal" without touching anything real.
RUN mkdir -p /root/.ssh /root/.aws \
 && echo "FAKE-SSH-KEY-DO-NOT-USE"      > /root/.ssh/id_rsa \
 && echo "[default]"                    > /root/.aws/credentials \
 && echo "aws_secret_access_key = FAKE" >> /root/.aws/credentials \
 && echo "# decoy"                      > /root/.bashrc

ENV NPM_TOKEN=npm_FAKE_TOKEN_FOR_TESTING
ENV AWS_SECRET_ACCESS_KEY=FAKE_SECRET

# Empty project so npm has somewhere to install into without warnings.
RUN mkdir -p /work && cd /work && npm init -y --silent

WORKDIR /work