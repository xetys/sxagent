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
	"github.com/streadway/amqp"
	"errors"
	"encoding/base64"
	"encoding/json"
)

// sendCmd represents the send command
var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {

		channel, _ := cmd.Flags().GetString("channel")
		if channel == "" {
			return errors.New("empty channel")
		}

		amqpUrl, _ := cmd.Flags().GetString("amqp")
		if amqpUrl == "" {
			return errors.New("empty amqp url")
		}

		key, _ := cmd.Flags().GetString("key")
		if key == "" {
			return errors.New("empty key")
		}

		conn, err := amqp.Dial(amqpUrl)
		defer conn.Close()

		return err
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("send called")
		channel, _ := cmd.Flags().GetString("channel")
		amqpUrl, _ := cmd.Flags().GetString("amqp")
		commandStr, _ := cmd.Flags().GetString("command")
		key, _ := cmd.Flags().GetString("key")

		privateKey, err := base64.StdEncoding.DecodeString(key)
		failOnError(err, "could not decode private key")

		r, s, err := Sign([]byte(commandStr), privateKey)


		command := Command{Type: "CMD", Command: commandStr, R: r.Bytes(), S: s.Bytes()}
		commandJson, err := json.Marshal(command)
		failOnError(err, "could not marshal")

		connRead, chRead, qRead := GetActiveChannel(amqpUrl, fmt.Sprintf("%s_i", channel))
		defer connRead.Close()
		defer chRead.Close()

		connWrite, chWrite, qWrite := GetActiveChannel(amqpUrl, fmt.Sprintf("%s_o", channel))
		defer connWrite.Close()
		defer chWrite.Close()
		err = chRead.Publish(
			"",         // exchange
			qRead.Name, // routing key
			false,      // mandatory
			false,      // immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        commandJson,
			})
		failOnError(err, "Failed to publish a message")

		msgs, err := chWrite.Consume(
			qWrite.Name, // queue
			"",          // consumer
			true,        // auto-ack
			false,       // exclusive
			false,       // no-local
			false,       // no-wait
			nil,         // args
		)

		failOnError(err, "Failed to register a consumer")
		for d := range msgs {
			fmt.Printf("%s", d.Body)
			break
		}
	},
}

func init() {
	rootCmd.AddCommand(sendCmd)
	sendCmd.Flags().StringP("amqp", "a", "amqp://localhost:5672", "The rabbitmq connection server")
	sendCmd.Flags().StringP("channel", "c", "open", "The connection channel")
	sendCmd.Flags().StringP("command", "x", "whoami", "send a command over pipe")
	sendCmd.Flags().StringP("key", "k", "", "the ECDSA private key")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sendCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sendCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
