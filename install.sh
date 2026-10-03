#!/usr/bin/env bash

set -e

sudo rm -rf /usr/local/wisp
sudo tar -C /usr/local -xzf dist/wisp-*-linux-amd64.tar.gz

echo "Wisp installed successfully."
