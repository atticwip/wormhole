package abi

import (
	"context"
	"github.com/qtumproject/janus/pkg/qtum"
	"log"
	"testing"
)

func Test_ABI(t *testing.T) {

	urlRPC := "http://qtum:testpasswd@0.0.0.0:13889"
	//address := "6f3ffd81c3092577f8155eb5d0ffca275b23c0b2"
	address := "7f9bb7eeb13f313d43ac176a06b2f00beddfe802"

	client, err := qtum.NewClient(false, urlRPC)
	if err != nil {
		t.Error(err)
	}

	m := &qtum.Method{Client: client}

	abiQ, err := NewAbiQtum(address, m)
	if err != nil {
		t.Error(err)
	}

	guardianSetC := make(chan *AbiLogGuardianSetChanged, 2)
	sub, err := abiQ.WatchLogGuardianSetChanged(context.Background(), guardianSetC)
	if err != nil {
		t.Error(err)
		return
	}

	for {
		log.Print("Loop tick")
		select {
		case l := <-guardianSetC:
			log.Print(l)
		case err = <-sub.Err():
			t.Error(err)
			return
		}

	}

	//query := [][]interface{}{{abiQ.abi.Events["LogGuardianSetChanged"].ID}}
	//
	//topics, err := abi.MakeTopics(query...)
	//if err != nil {
	//	t.Error(err)
	//	return
	//}
	//
	//log.Print(topics)
	//return
	//"dfb80683934199683861bf00b64ecdf0984bbaf661bf27983dba382e99297a62"

	//
	//bt, err := abi.Store(3)
	//
	//log.Print(hex.EncodeToString(bt))
	//return

	//out, err := abi.Get()
	//if err != nil {
	//	t.Error(err)
	//}
	//
	//log.Print(out)
	//
	//resp, err := m.SearchLogs(&qtum.SearchLogsRequest{
	//	FromBlock: big.NewInt(950380),
	//	ToBlock:   big.NewInt(950380),
	//	//FromBlock: big.NewInt(908412),
	//	//ToBlock:   big.NewInt(948413),
	//	//Addresses: []string{"6f3ffd81c3092577f8155eb5d0ffca275b23c0b2"},
	//	Topics: nil,
	//})
	//if err != nil {
	//	t.Error(err)
	//}
	//
	//for _, value := range resp {
	//	log.Printf("%+v", value)
	//}

	//
	//ss, err := m.FromHexAddress("d362e096e873eb7907e205fadc6175c6fec7bc44")
	//if err != nil {
	//	t.Error(err)
	//}
	//
	//log.Print(ss)

	//rawtxreq := []interface{}{
	//	nil,
	//	nil,
	//	Filter{
	//		Addresses: []string{"7f9bb7eeb13f313d43ac176a06b2f00beddfe802"},
	//		//Topics:    nil,
	//	},
	//	1,
	//}
	//
	//log.Print(common.HexToAddress("7f9bb7eeb13f313d43ac176a06b2f00beddfe802").Hex())
	//		map[string]interface{}{
	//	"fromBlock": nil, //big.NewInt(816230),
	//	"toBlock":   nil, //big.NewInt(816240),
	//	"filter": Filter{
	//		Addresses: []common.Address{common.HexToAddress("7f9bb7eeb13f313d43ac176a06b2f00beddfe802")},
	//		Topics:    nil,
	//	},
	//}

	//rawtxreq

	//bt, err := json.Marshal(rawtxreq)
	//if err != nil {
	//
	//}
	//
	//log.Print(string(bt))

	//Node tx building used to escape hold node constants in code
	//var rawTx Log
	//if err := client.Request(qtum.MethodWaitForLogs, rawtxreq, &rawTx); err != nil {
	//	t.Error(err)
	//}
	//
	//log.Print(rawTx)
}
