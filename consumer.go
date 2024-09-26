	package main

	import (
		"fmt"
		"encoding/json"
		"net"
		amqp "github.com/rabbitmq/amqp091-go"
		"strings"
		"time"
		"errors"
		"bytes"
		//"bufio"
		"sync"
	)

	const (
		// MLLP delimiters
		MLLPStartBlock = byte(0x0b) // <VT>
		MLLPEndBlock1  = byte(0x1c) // <FS>
		MLLPEndBlock2  = byte(0x0d) // <CR>
		MLLPEmpty      = byte(0x00) // <CR>
	)

	func cleanMLLP(buffer string) string {
		// Remover caracteres de control específicos del protocolo MLLP
		cleaned := strings.Map(func(r rune) rune {
			// Remover MLLP delimiters (0x0b, 0x1c, 0x0d)
			if byte(r) == MLLPStartBlock || byte(r) == MLLPEndBlock1 || byte(r) == MLLPEndBlock2 || byte(r) == MLLPEmpty {
				return -1
			}
			// Dejar otros caracteres, incluyendo saltos de línea
			return r
		}, buffer)

		return cleaned
	}

	func sendHL7Message(destination string, port int, message string) (string, error) {
		// Establecer conexión TCP al servidor HL7
		//hl7Message := "MSH|^~\\&|SENDING_APPLICATION|SENDING_FACILITY|RECEIVING_APPLICATION|RECEIVING_FACILITY|20110613083617||ADT^A01|934576120110613083617|P|2.3||||\rEVN|A01|20110613083617|||\rPID|1||135769||MOUSE^MICKEY^||19281118|M|||123 Main St.^^Lake Buena Vista^FL^32830||(407)939-1289^^^theMainMouse@disney.com|||||1719|99999999||||||||||||||||||||\rPV1|1|O|||||^^^^^^^^|^^^^^^^^"
		//"MSH|^~\\&|ADT1|MCM|FINGER|MCM|198808181126|SECURITY|ADT^A01|MSG00001|P|2.3.1\rEVN|A01|198808181123\rPID|1||PATID1234^5^M11^ADT1^MR^MCM~123456789^^^USSSA^SS||SMITH^WILLIAM^A^III||19610615|M||C|1200 N ELM STREET^^JERUSALEM^TN^99999?1020|GL|(999)999?1212|(999)999?3333||S||PATID12345001^2^M10^ADT1^AN^A|123456789|987654^NC\rNK1|1|SMITH^OREGANO^K|WI^WIFE||||NK^NEXT OF KIN\rPV1|1|I|2000^2012^01||||004777^CASTRO^FRANK^J.|||SUR||||ADM|A0"
		// MSH|^~\\&|SendingApp|SendingFac|ReceivingApp|ReceivingFac|202404271111||ADT^A01|123456|P|2.5\r"

		address := fmt.Sprintf("%s:%d", destination, port)
		conn, err := net.Dial("tcp", address)
		if err != nil {
			return "", fmt.Errorf("error al conectar con el servidor: %v", err)
		}
		defer conn.Close() // Cerrar la conexión al finalizar

		// Construir el mensaje MLLP
		var mllpMessage strings.Builder
		mllpMessage.WriteByte(MLLPStartBlock)    // Delimitador de inicio de MLLP
		mllpMessage.WriteString(message)      // Mensaje HL7
		mllpMessage.WriteByte(MLLPEndBlock1)     // Primer delimitador de fin de MLLP
		mllpMessage.WriteByte(MLLPEndBlock2)     // Segundo delimitador de fin de MLLP

		// Enviar el mensaje MLLP a través de la conexión TCP
		_, err = conn.Write([]byte(mllpMessage.String()))
		if err != nil {
			return "", fmt.Errorf("error al enviar el mensaje HL7: %v", err)
		}

		// Esperar y leer la respuesta del servidor
		conn.SetReadDeadline(time.Now().Add(5 * time.Second)) // Timeout de lectura
		response := make([]byte, 1024) // Tamaño del buffer
		_, err = conn.Read(response)
		if err != nil {
			return "", fmt.Errorf("error al recibir respuesta: %v", err)
		} else {
			containsMSH := bytes.Contains(response, []byte("MSH"))
			containsMSA := bytes.Contains(response, []byte("MSA"))
			cleanedBuffer := cleanMLLP(string(response))
			writeHL7ToFile(string(cleanedBuffer), "hl7_response.txt")
			if containsMSA && containsMSH {
				return cleanedBuffer, nil
			} else {
				return cleanedBuffer, errors.New("Error del servidor")
			}
		}
	}
	func consumeRecords(wg *sync.WaitGroup, cfg HL7Config) {
		defer wg.Done()
		// Connect to RabbitMQ server
		conn, err := amqp.Dial(config.RabbitmqURI)
		if err != nil {
			fmt.Errorf("Failed to connect to RabbitMQ: %v", err)
			return
		}
		defer conn.Close()

		// Open a channel
		ch, err := conn.Channel()
		if err != nil {
			fmt.Errorf("Failed to open a channel: %v", err)
			return
		}
		defer ch.Close()

		// Declare the queue
		q, err := ch.QueueDeclare(
			cfg.QueueName, // name
			true,     // durable
			false,     // delete when unused
			false,     // exclusive
			false,     // no-wait
			nil,       // arguments
		)
		if err != nil {
			fmt.Errorf("Failed to declare a queue: %v", err)
			return
		}

		// Consume messages
		msgs, err := ch.Consume(
			q.Name, // queue
			"",     // consumer
			true,   // auto-ack
			false,  // exclusive
			false,  // no-local
			false,  // no-wait
			nil,    // args
		)
		if err != nil {
			fmt.Errorf("Failed to register a consumer: %v", err)
			return
		}

		var hisRecord  HealthRecord
		//var hl7Message *bytes.Buffer

		for d := range msgs {
			//fmt.Println(d)
			err = json.Unmarshal(d.Body, &hisRecord)
			if err != nil {
				fmt.Printf("Error deserializando mensaje: %v\n", err)
				continue // Saltar este mensaje si hay un error
			}
			// 	return string(d.Body), nil // Process the message
			hl7Message := generateHL7(cfg.Mapping, hisRecord)

			for _, dest := range cfg.HL7Destinations {
				fmt.Printf("Enviando HL7 a %s:%d\n", dest.IPAddress, dest.Port)
				response, err := sendHL7Message(dest.IPAddress, dest.Port, hl7Message)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
				} else {
					fmt.Printf("Respuesta del servidor: %s\n", response)
				}
			}
		}

		return
	}
