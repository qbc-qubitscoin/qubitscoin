package main

import "github.com/qbc-qubitscoin/qubitscoin/internal/contracts/multisig"

//export init_multisig
func init_multisig(payloadPtr, payloadLen uint32) int32 {
	return multisig.InitMultisig(payloadPtr, payloadLen)
}

//export execute_transfer
func execute_transfer(payloadPtr, payloadLen uint32) int32 {
	return multisig.ExecuteTransfer(payloadPtr, payloadLen)
}

func main() {}
