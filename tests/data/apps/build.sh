#!/usr/bin/env bash
set -e
#
# Package some example docker compose as gzip and zip packages
#

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
pushd "$SCRIPT_DIR" >/dev/null ||:

(cd app1 && tar czvf ../app1.tar.gz docker-compose.yaml Dockerfile static/*)
(cd app2 && zip ../app2.zip docker-compose.yaml Dockerfile static/*)

popd ||:
