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
	"log"
	"os/exec"
	"bytes"
	"encoding/json"
	"strings"
	"encoding/base64"
	"math/big"
	"os"
)

// listenCmd represents the listen command
var listenCmd = &cobra.Command{
	Use:   "listen",
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

		_, err := amqp.Dial(amqpUrl)

		return err
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("listen called")

		channel, _ := cmd.Flags().GetString("channel")
		amqpUrl, _ := cmd.Flags().GetString("amqp")
		key, _ := cmd.Flags().GetString("key")
		hostName, err := os.Hostname()
		failOnError(err, "could not retrieve hostname")

		publicKey, err := base64.StdEncoding.DecodeString(key)
		failOnError(err, "could not decode public key")

		connRead, chRead, qRead := GetActiveChannel(amqpUrl, fmt.Sprintf("%s_i", channel))
		defer connRead.Close()
		defer chRead.Close()

		connWrite, chWrite, qWrite := GetActiveChannel(amqpUrl, fmt.Sprintf("%s_o", channel))
		defer connWrite.Close()
		defer chWrite.Close()

		msgs, err := chRead.Consume(
			qRead.Name, // queue
			"",         // consumer
			true,       // auto-ack
			false,      // exclusive
			false,      // no-local
			false,      // no-wait
			nil,        // args
		)
		failOnError(err, "Failed to register a consumer")

		forever := make(chan bool)

		go func() {
			defer func() {
				if err := recover(); err != nil {
					log.Println("work failed:", err)
				}
			}()
			for d := range msgs {
				log.Printf("Received a message: %s", d.Body)

				var command Command
				var commandName string
				var commandArgs []string
				err := json.Unmarshal(d.Body, &command)
				failOnError(err, "could not unmarshal")
				switch command.Type {
				case "CMD":

					// first verify we can trust this command
					r := big.Int{}
					r.SetBytes(command.R)
					s := big.Int{}
					s.SetBytes(command.S)
					verification := Verify([]byte(command.Command), publicKey, &r, &s)

					if !verification {
						log.Fatalln("command verification failed")
					}

					commands := strings.SplitAfter(command.Command, " ")
					switch len(commands) {
					case 0:
						continue
					case 1:
						commandName = strings.Trim(commands[0], " ")
						commandArgs = []string{}
					default:
						commandName = strings.Trim(commands[0], " ")
						commandArgs = commands[1:]
					}
					cmd := exec.Command(commandName, commandArgs...)

					var out bytes.Buffer
					cmd.Stdout = &out
					err = cmd.Run()

					if err != nil {
						log.Println(err)
						err = chWrite.Publish(
							"",          // exchange
							qWrite.Name, // routing key
							false,       // mandatory
							false,       // immediate
							amqp.Publishing{
								ContentType: "text/plain",
								Body:        []byte(err.Error()),
							})

						failOnError(err, "could not send back")
					} else {
						log.Println(out.String())
						err = chWrite.Publish(
							"",          // exchange
							qWrite.Name, // routing key
							false,       // mandatory
							false,       // immediate
							amqp.Publishing{
								ContentType: "text/plain",
								Body:        []byte("OK"),
							})

						failOnError(err, "could not send back")
					}
				case "PING":
					err = chWrite.Publish(
						"",          // exchange
						qWrite.Name, // routing key
						false,       // mandatory
						false,       // immediate
						amqp.Publishing{
							ContentType: "text/plain",
							Body:        []byte(hostName),
						})

					failOnError(err, "could not send back")
				}
			}
		}()

		log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
		<-forever
	},
}

func GetActiveChannel(amqpUrl string, channelName string) (*amqp.Connection, *amqp.Channel, amqp.Queue) {
	conn, err := amqp.Dial(amqpUrl)
	failOnError(err, "failed to dial")
	ch, err := conn.Channel()
	failOnError(err, "failed to open channel")
	q, err := ch.QueueDeclare(
		channelName,
		false,
		true,
		false,
		false,
		nil,
	)
	failOnError(err, "could not declare queue")

	return conn, ch, q
}

func init() {
	rootCmd.AddCommand(listenCmd)

	listenCmd.Flags().StringP("amqp", "a", "amqp://localhost:5672", "The rabbitmq connection server")
	listenCmd.Flags().StringP("channel", "c", "open", "The connection channel")
	listenCmd.Flags().StringP("key", "k", "", "the ECDSA public key")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listenCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listenCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
