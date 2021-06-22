#!/bin/sh
# This script configures the devnet for test transfers with hardcoded addresses.
set -x

import_key () {

    echo Running command - "$@"
    i=0
    until [ ! $i -lt 10 ]
    do
      res=$(curl --data-binary "{\"jsonrpc\": \"1.0\", \"id\":\"curltest\", \"method\": \"importprivkey\", \"params\": [\"${1}\", \"address1\", false] }" -H 'content-type: text/plain;' http://qtum:testpasswd@127.0.0.1:3889)
      err=$(echo $res | jq '.error.code')
      if [ $err = 'null' ]; then
         echo Command finished successfully
         return
      fi
      sleep 1
      echo Retrying
      i=`expr $i + 1`
    done

  echo Giving up running command - "$@"
}

#import private keys and then prefund them

import_key "cMbgxCJrTYUqgcmiC1berh5DFrtY1KeU4PXZ6NZxgenniF1mXCRk" # addr=qUbxboqjBRp96j3La8D1RYkyqx5uQbJPoW hdkeypath=m/88'/0'/1'
import_key "cRcG1jizfBzHxfwu68aMjhy78CpnzD9gJYZ5ggDbzfYD3EQfGUDZ" # addr=qLn9vqbr2Gx3TsVR9QyTVB5mrMoh4x43Uf hdkeypath=m/88'/0'/2'
import_key "cV79qBoCSA2NDrJz8S3T7J8f3zgkGfg4ua4hRRXfhbnq5VhXkukT" # addr=qTCCy8qy7pW94EApdoBjYc1vQ2w68UnXPi
import_key "cV93kaaV8hvNqZ711s2z9jVWLYEtwwsVpyFeEZCP6otiZgrCTiEW" # addr=qWMi6ne9mDQFatRGejxdDYVUV9rQVkAFGp
import_key "cVPHpTvmv3UjQsZfsMRrW5RrGCyTSAZ3MWs1f8R1VeKJSYxy5uac" # addr=qLcshhsRS6HKeTKRYFdpXnGVZxw96QQcfm
import_key "cTs5NqY4Ko9o6FESHGBDEG77qqz9me7cyYCoinHcWEiqMZgLC6XY" # addr=qW28njWueNpBXYWj2KDmtFG2gbLeALeHfV

echo Finished importing accounts
echo Seeding accounts

# address1
echo Seeding qUbxboqjBRp96j3La8D1RYkyqx5uQbJPoW
curl --data-binary '{"jsonrpc": "1.0", "id":"curltest", "method": "generatetoaddress", "params": [1000, "qUbxboqjBRp96j3La8D1RYkyqx5uQbJPoW"] }' -H 'content-type: text/plain;' http://qtum:testpasswd@127.0.0.1:3889

# address2
echo Seeding qLn9vqbr2Gx3TsVR9QyTVB5mrMoh4x43Uf
curl --data-binary '{"jsonrpc": "1.0", "id":"curltest", "method": "generatetoaddress", "params": [1000, "qLn9vqbr2Gx3TsVR9QyTVB5mrMoh4x43Uf"] }' -H 'content-type: text/plain;' http://qtum:testpasswd@127.0.0.1:3889

# address3
echo Seeding qTCCy8qy7pW94EApdoBjYc1vQ2w68UnXPi
curl --data-binary '{"jsonrpc": "1.0", "id":"curltest", "method": "generatetoaddress", "params": [500, "qTCCy8qy7pW94EApdoBjYc1vQ2w68UnXPi"] }' -H 'content-type: text/plain;' http://qtum:testpasswd@127.0.0.1:3889

# address4
echo Seeding qWMi6ne9mDQFatRGejxdDYVUV9rQVkAFGp
curl --data-binary '{"jsonrpc": "1.0", "id":"curltest", "method": "generatetoaddress", "params": [250, "qWMi6ne9mDQFatRGejxdDYVUV9rQVkAFGp"] }' -H 'content-type: text/plain;' http://qtum:testpasswd@127.0.0.1:3889

# address5
echo Seeding qLcshhsRS6HKeTKRYFdpXnGVZxw96QQcfm
curl --data-binary '{"jsonrpc": "1.0", "id":"curltest", "method": "generatetoaddress", "params": [100, "qLcshhsRS6HKeTKRYFdpXnGVZxw96QQcfm"] }' -H 'content-type: text/plain;' http://qtum:testpasswd@127.0.0.1:3889

# address6
echo Seeding qW28njWueNpBXYWj2KDmtFG2gbLeALeHfV
curl --data-binary '{"jsonrpc": "1.0", "id":"curltest", "method": "generatetoaddress", "params": [100, "qW28njWueNpBXYWj2KDmtFG2gbLeALeHfV"] }' -H 'content-type: text/plain;' http://qtum:testpasswd@127.0.0.1:3889

echo Finished importing and seeding accounts

echo Start migration

npm run migrate

nc -l -p 2000

sleep infinity