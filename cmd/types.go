package cmd

import "context"

type Command struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	R       []byte `json:"r"`
	S       []byte `json:"s"`
}

type KeyPair struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

type CryptoService interface {
	GenKeys(ctx context.Context) (KeyPair)
}
