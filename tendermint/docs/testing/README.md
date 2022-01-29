---
title: Testing the consensus algorithm
description: Testing the core consensus algorithm by intercepting messages and using a testing strategy
---

# Setup

The configuration template file located at [`networks/local/localnode/config-template.toml`](networks/local/localnode/config-template.toml) is changed for the local testnet setup. The following config valiable are changed
- `proxy_app` is set to `kvstore`
- `consensus/create-empty-blocks=false`
- `p2p/test-intercept=true`
- `p2p/controller-master-addr` should be set to the address the scheduler is run on

## Build

1. Run `make build-linux` to create the binary. Rerun this after making any changes to the codebase. 
2. Run `make localnet-start` to spin up the local cluster. However the config files in `build/node{ID}/config/config.toml` will have to be changed. The following config variables will have to be changed appropriately:
    - `p2p/controller-listen-addr` should be set to `0.0.0.0:8080`
    - `p2p/controller-extern-addr` should be set to `127.0.0.1:NODE_PORT`
where `NODE_PORT` can be fetched from `docker-compose.yml` for that particular node. To make the changes stop the running cluster, make the changes and rerun `make localnet-start`