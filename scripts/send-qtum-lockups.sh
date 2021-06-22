#!/usr/bin/env bash
set -e

kubectl exec -it -c tests qtum-devnet-0 -- npx truffle exec src/send-lockups.js
