package qtum

import (
	"context"
	"encoding/base64"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"log"
	"testing"
)

func Test_SenderTest(t *testing.T) {
	//payload := "AQAAAAABAIjk+oquxkVaQUu2CJAbOAa427a+LqTO70k+TWT48ndnGF2tiIdeIaf01XRVa0st27wnXwyC0J4Niwzu0GYtzP4AYMckHRAAAD47BAIAAAAAAAAAAAAAAAB5JiIwcFR9LRWy715zg+VBwzj/6QAAAAAAAAAAAAAAAJD4v2pHnzIOrQdEEaSw55ROqMnBBAAAAAAAAAAAAAAAAP0j8NVCc/oF9YitjqZGhF0HyqQHCQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB"
	//payload := "AQAAAAABANdobDyDZbv6WVzSC2EgdGgLGYokUySAbyvKYIC8McF3TGZfjl/GghYZV2+xsJWULvyckuFRTLbSfBKkXeS+4dYBYMsY7hDgmp2tBAIAAAAAAAAAAAAAAAB5JiIwcFR9LRWy715zg+VBwzj/6QAAAAAAAAAAAAAAAJD4v2pHnzIOrQdEEaSw55ROqMnBBAAAAAAAAAAAAAAAALTT4oeThEpPW9gzXaobzxjmm4I1CQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB"

	payload := "AQAAAAABAOu3o/H3oOHdLBMszscmeuffYiM4a4GNteh8V9c6FRGlW/mfppubMBfnQjJ+sRhWhtMge1A/NXKjhOMYGsKhiuYBAAASUhAAAJW/AgQAAAAAAAAAAAAAAACQ+L9qR58yDq0HRBGksOeUTqjJwQAAAAAAAAAAAAAAAHkmIjBwVH0tFbLvXnOD5UHDOP/pAgAAAAAAAAAAAAAAAM/rhp9pQx5CzbVKT08QXBnAgKYBCQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB"
	//vaa  := vaa.VAA{}
	bt, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Error(err)
		return
	}

	va, err := vaa.Unmarshal(bt)
	if err != nil {
		t.Error(err)
		return
	}

	log.Printf("%+v", va.Payload.(*vaa.BodyTransfer))
	log.Print(va.Payload.(*vaa.BodyTransfer).SourceChain)
	log.Print(va.Payload.(*vaa.BodyTransfer).TargetChain)

	log.Print(va.Payload.(*vaa.BodyTransfer).Asset.Chain)
	log.Print(va.Payload.(*vaa.BodyTransfer).Asset.Address)
	log.Print(va.Payload.(*vaa.BodyTransfer).Asset.Decimals)
	return

	//client, err := qtum.NewClient(false, "http://qtum:testpasswd@0.0.0.0:3889")

	//m := &qtum.Method{Client: client}

	//log.Print(m.GetBlockChainInfo())
	return
	//wstr, err := hex.DecodeString("39623062393731666236653338303839653032386663383833663136376336646134323534653732353461316432373364383263636431376465633538326135")
	//if err != nil {
	//	t.Error(err)
	//}
	//
	//log.Print(string(wstr))
	//return

	//"qUbxboqjBRp96j3La8D1RYkyqx5uQbJPoW"
	resp, err := SubmitVAA(context.Background(), "http://qtum:testpasswd@0.0.0.0:13889", "" /*"6f3ffd81c3092577f8155eb5d0ffca275b23c0b2"*/, "7f9bb7eeb13f313d43ac176a06b2f00beddfe802", "cMbgxCJrTYUqgcmiC1berh5DFrtY1KeU4PXZ6NZxgenniF1mXCRk", &vaa.VAA{Payload: &vaa.BodyTransfer{}})
	if err != nil {
		t.Error(err)
	}

	log.Print(resp)
}
