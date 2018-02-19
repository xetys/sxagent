// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"io"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"crypto/rand"
	"encoding/base64"
	"math/big"
	"context"
	"github.com/go-kit/kit/endpoint"
	"net/http"

	httptransport "github.com/go-kit/kit/transport/http"
	"encoding/json"
	"log"
)

type cryptoService struct{}

func (cryptoService) GenKeys(ctx context.Context) (KeyPair) {

	priv, pub := genPPKeys(rand.Reader)

	return KeyPair{
		PublicKey:  base64.StdEncoding.EncodeToString(pub),
		PrivateKey: base64.StdEncoding.EncodeToString(priv),
	}
}

func makeGenKeysEndpoint(svc CryptoService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		keyPair := svc.GenKeys(ctx)

		return keyPair, nil
	}
}

// cryptoCmd represents the crypto command
var cryptoCmd = &cobra.Command{
	Use:   "crypto",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		serve, _ := cmd.Flags().GetBool("serve")
		if serve {
			serveCryptoService()
		} else {
			priv, pub := genPPKeys(rand.Reader)
			fmt.Printf("private key %s, public key %s", base64.StdEncoding.EncodeToString(priv), base64.StdEncoding.EncodeToString(pub))
		}
	},
}

func genPPKeys(random io.Reader) (private_key_bytes, public_key_bytes []byte) {
	private_key, _ := ecdsa.GenerateKey(elliptic.P224(), random)
	private_key_bytes, _ = x509.MarshalECPrivateKey(private_key)
	public_key_bytes, _ = x509.MarshalPKIXPublicKey(&private_key.PublicKey)
	return private_key_bytes, public_key_bytes
}

func Sign(hash []byte, private_key_bytes []byte) (r, s *big.Int, err error) {
	zero := big.NewInt(0)
	private_key, err := x509.ParseECPrivateKey(private_key_bytes)
	if err != nil {
		return zero, zero, err
	}

	r, s, err = ecdsa.Sign(rand.Reader, private_key, hash)
	if err != nil {
		return zero, zero, err
	}
	return r, s, nil
}

func Verify(hash []byte, public_key_bytes []byte, r *big.Int, s *big.Int) (result bool) {
	public_key, err := x509.ParsePKIXPublicKey(public_key_bytes)
	if err != nil {
		return false
	}

	switch public_key := public_key.(type) {
	case *ecdsa.PublicKey:
		return ecdsa.Verify(public_key, hash, r, s)
	default:
		return false
	}
}

func serveCryptoService() {

	svc := cryptoService{}

	cryptoServiceHandle := httptransport.NewServer(
		makeGenKeysEndpoint(svc),
		decodeRequest,
		encodeResponse,
	)

	http.Handle("/gen-key-pair", cryptoServiceHandle)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func decodeRequest(context context.Context, request2 *http.Request) (request interface{}, err error) {
	return nil, nil
}
func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Add("Content-type", "application/json")
	return json.NewEncoder(w).Encode(response)
}

func init() {
	rootCmd.AddCommand(cryptoCmd)

	cryptoCmd.Flags().BoolP("serve", "s", false, "Serve an simple endpoint for key generation")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// cryptoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// cryptoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
