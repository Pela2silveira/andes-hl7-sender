package main

import (
	"fmt"
	"encoding/json"
	"net"
	amqp "github.com/rabbitmq/amqp091-go"
	"strings"
	"time"
	"bufio"

)

const (
	// MLLP delimiters
    MLLPStartBlock = byte(0x0b) // <VT>
    MLLPEndBlock1  = byte(0x1c) // <FS>
    MLLPEndBlock2  = byte(0x0d) // <CR>
)

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
    response, err := bufio.NewReader(conn).ReadString(MLLPEndBlock2)
    if err != nil {
        return "", fmt.Errorf("error al recibir respuesta: %v", err)
    }

    // Remover delimitadores MLLP de la respuesta
    response = strings.TrimPrefix(response, string(MLLPStartBlock))
    response = strings.TrimSuffix(response, string(MLLPEndBlock1)+string(MLLPEndBlock2))

    // Verificar si la respuesta es un ACK
    if strings.Contains(response, "MSA|AA") {
        return "ACK recibido", nil
    } else {
        return fmt.Sprintf("Respuesta no ACK recibida: %s", response), nil
    }
}

func consumeRecords() (error) {
	// Connect to RabbitMQ server
	conn, err := amqp.Dial(config.RabbitmqURI)
	if err != nil {
		return fmt.Errorf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Open a channel
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declare the queue
	q, err := ch.QueueDeclare(
		config.QueueName, // name
		true,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("Failed to declare a queue: %v", err)
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
		return fmt.Errorf("Failed to register a consumer: %v", err)
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
		hl7Message := generateHL7(config.HL7Config.Mapping, hisRecord)

		for _, dest := range config.HL7Config.HL7Destinations {
			fmt.Printf("Enviando HL7 a %s:%d\n", dest.IPAddress, dest.Port)
			response, err := sendHL7Message(dest.IPAddress, dest.Port, hl7Message)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Respuesta del servidor: %s\n", response)
			}
		}
	}

	return fmt.Errorf("No messages in the queue")
}
