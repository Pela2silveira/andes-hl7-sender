package main

import (
	"bufio"
	"fmt"
	"io/ioutil"

	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

const (
	// Delimitadores MLLP
	MLLPStartBlock = byte(0x0b) // <VT>
	MLLPEndBlock1  = byte(0x1c) // <FS>
	MLLPEndBlock2  = byte(0x0d) // <CR>
	MLLPEmpty      = byte(0x00) // <CR>
)

func sendHL7Message(conn net.Conn, message string) (string, error) {
	// Construir el mensaje MLLP
	var mllpMessage strings.Builder
	mllpMessage.WriteByte(MLLPStartBlock) // Delimitador de inicio de MLLP
	mllpMessage.WriteString(message)      // Mensaje HL7
	mllpMessage.WriteByte(MLLPEndBlock1)  // Primer delimitador de fin de MLLP
	mllpMessage.WriteByte(MLLPEndBlock2)  // Segundo delimitador de fin de MLLP

	// Enviar el mensaje MLLP a través de la conexión TCP
	_, err := conn.Write([]byte(mllpMessage.String()))
	if err != nil {
		return "", fmt.Errorf("error al enviar el mensaje HL7: %v", err)
	} else {
		return "Mensaje Enviado", nil
	}

}

// receiveHL7Message recibe un mensaje HL7 de una conexión de red.
func receiveHL7Message(conn net.Conn) (string, error) {
	reader := bufio.NewReader(conn)

	// Establecer un tiempo límite para la lectura
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Leer datos de la conexión hasta el carácter de fin de bloque MLLP
	data, err := reader.ReadString('\x1c') // \x1c es el carácter de fin de bloque en MLLP
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("WARNING: Tiempo de espera agotado al recibir el mensaje HL7: %v", err)
			return "", err
		}
		log.Printf("ERROR: Error al recibir el carácter de fin de bloque: %v", err)
		return "", err
	}

	// Remover los caracteres de inicio y fin de mensaje MLLP
	data = strings.TrimSuffix(data, "\x1c")
	data = strings.TrimPrefix(data, "\x0b")
	response := string(data)
	return response, nil
}

// writeHL7ToFile escribe el mensaje HL7 en un archivo.
func writeHL7ToFile(hl7Message, fileName string) error {
	// Crea o abre el archivo para escribir
	file, err := os.Create(fileName)
	if err != nil {
		log.Printf("ERROR: Error al crear o abrir el archivo %s: %v", fileName, err)
		return err
	}
	defer file.Close()

	// Escribe el mensaje HL7 en el archivo
	_, err = file.WriteString(hl7Message)
	if err != nil {
		log.Printf("ERROR: Error al escribir el mensaje HL7 en el archivo %s: %v", fileName, err)
		return err
	}

	log.Printf("INFO: Mensaje HL7 escrito exitosamente en %s", fileName)
	return nil
}

// generateHL7 genera un mensaje HL7 a partir de un registro de salud y un mapeo.
func generateHL7(mapping Mapping, hisRecord HealthRecord) string {
	var hl7Message strings.Builder
	var lastFieldIndex int
	var lastComponentIndex int
	var lastrepetitionIndex int

	// Segmento MSH
	hl7Message.WriteString("MSH")
	hl7Message.WriteString(mapping.Delimiters.FieldSeparator)
	hl7Message.WriteString(mapping.Delimiters.ComponentSeparator)
	hl7Message.WriteString(mapping.Delimiters.RepetitionCharacter)
	hl7Message.WriteString(mapping.Delimiters.EscapeCharacter)
	hl7Message.WriteString(mapping.Delimiters.SubcomponentSeparator)

	for _, segment := range mapping.Mappings {
		if segment.Segment != "msh" {
			hl7Message.WriteString(strings.ToUpper(segment.Segment))
		}

		lastFieldIndex = 0
		lastComponentIndex = 1
		lastrepetitionIndex = 0

		hl7Message.WriteString(mapping.Delimiters.FieldSeparator)

		for _, field := range segment.Values {

			var componentValue string
			var repetitionIndex int

			if field.Field == "date_time" {
				// Insertar la fecha y hora actual en el formato especificado
				componentValue = time.Now().Format("20060102150405")
			} else {
				// Dividir el campo para obtener la ruta jerárquica (ej. "paciente.apellido")
				// value, exists := getValueFromPath(hisRecord, field.Field)
				value, err := jsonpath.Get(field.Field, hisRecord)
				if err != nil {
					// Log the error for debugging, but don't return an error as per the original function signature
					// Instead, return false for "exists" and a nil value.
					log.Printf("WARNING: Error al obtener el valor de jsonpath '%s': %v", field.Field, err)
					componentValue = field.Default // Usar el valor predeterminado si no se encuentra
				} else {
					componentValue = fmt.Sprintf("%v", value) // Convertir a string
				}
			}

			// Manejo de la separación de componentes y campos
			fieldIndex := field.Component[0]
			componentIndex := field.Component[1]
			if len(field.Component) > 2 {
				repetitionIndex = field.Component[2]
			} else {
				repetitionIndex = 0
			}

			if fieldIndex > lastFieldIndex {
				for i := lastFieldIndex; i < fieldIndex; i++ {
					hl7Message.WriteString(mapping.Delimiters.FieldSeparator)
				}
				lastFieldIndex = fieldIndex
				lastComponentIndex = 1
				lastrepetitionIndex = 0
			}

			if componentIndex > lastComponentIndex {
				for i := lastComponentIndex; i < componentIndex; i++ {
					hl7Message.WriteString(mapping.Delimiters.ComponentSeparator)
				}
				lastComponentIndex = componentIndex
			}

			// posiblemente haya casos mas complejos no handleados
			if repetitionIndex > lastrepetitionIndex {
				hl7Message.WriteString(mapping.Delimiters.RepetitionCharacter)
				lastComponentIndex = 1
				lastrepetitionIndex = repetitionIndex
			}

			hl7Message.WriteString(componentValue)
		}
		hl7Message.WriteString("\r")
	}
	writeHL7ToFile(hl7Message.String(), "hl7_parsed_message.txt")
	log.Println("INFO: Mensaje HL7 generado:")
	lines := strings.Split(hl7Message.String(), "\r")
	for _, line := range lines {
		if line != "" { // Verifica que no sea una línea vacía
			log.Println(line)
		}
	}
	return hl7Message.String()
}

func parseHL7Message(hl7Message string, mapping Mapping, queueName string) map[string]interface{} {
	result := make(map[string]interface{})
	segments := strings.Split(hl7Message, "\r")
	delimiter := mapping.Delimiters.FieldSeparator
	componentDelimiter := mapping.Delimiters.ComponentSeparator

	var obxText string
	var encapsulatedData strings.Builder
	var isEncapsulated bool
	decoder := charmap.ISO8859_1.NewDecoder()

	for _, segment := range segments {
		parts := strings.Split(segment, delimiter)
		if len(parts) == 0 {
			continue
		}
		segmentName := strings.ToLower(parts[0])

		if segmentName == "obx" {
			// Detectamos si es el primer OBX de tipo ED (Encapsulated Data)
			if !isEncapsulated && len(parts) > 2 && strings.ToUpper(parts[2]) == "ED" {
				isEncapsulated = true
			}

			// Si es encapsulado, concatenamos para reconstruir el documento
			if isEncapsulated {
				if len(parts) > 5 {
					pdfStream := strings.Split(parts[5], componentDelimiter)
					if len(pdfStream) > 3 {
						encapsulatedData.WriteString(pdfStream[4])
					}
				}
			} else {
				// Si no es encapsulado, asumimos texto plano
				obxText += strings.Join(parts[5:], " ") + "\n"
			}
			continue
		}

		for _, segmentMapping := range mapping.Mappings {
			if segmentMapping.Segment == segmentName {
				parseSegment(parts, segmentMapping, result, mapping.Delimiters.ComponentSeparator)
			}
		}
	}

	if isEncapsulated {
		// Guardamos el contenido como base64 en el JSON((
		saveBase64PDF(encapsulatedData.String(), "tmp.pdf")
		result["informe_base64"] = encapsulatedData.String()
		result["informe_tipo"] = "pdf"
	} else {
		// Decodificamos si es texto plano en ISO-8859-1
		obxUTF8, err := ioutil.ReadAll(transform.NewReader(strings.NewReader(obxText), decoder))
		if err == nil {
			result["informe"] = string(obxUTF8)
		}
	}
	result["queueName"] = queueName

	log.Println("INFO: Mensaje HL7 analizado a JSON:", result)
	return result
}

// // parseHL7Message analiza un mensaje HL7 y lo convierte en un mapa JSON.
// func parseHL7Message(hl7Message string, mapping Mapping, queueName string) map[string]interface{} {
// 	result := make(map[string]interface{})

// 	segments := strings.Split(hl7Message, "\r")
// 	delimiter := mapping.Delimiters.FieldSeparator

// 	obxText := ""
// 	decoder := charmap.ISO8859_1.NewDecoder()

// 	for _, segment := range segments {
// 		parts := strings.Split(segment, delimiter)
// 		if len(parts) == 0 {
// 			continue
// 		}
// 		segmentName := strings.ToLower(parts[0])

// 		// Si es un segmento OBX, concatena el texto de cada campo en obxText
// 		if segmentName == "obx" {
// 			obxText += strings.Join(parts[5:], " ") + "\n" // El campo de texto OBX suele estar en la posición 5
// 			continue
// 		}

// 		for _, segmentMapping := range mapping.Mappings {
// 			if segmentMapping.Segment == segmentName {
// 				parseSegment(parts, segmentMapping, result, mapping.Delimiters.ComponentSeparator)
// 			}
// 		}
// 	}
// 	obxUTF8, err := ioutil.ReadAll(transform.NewReader(strings.NewReader(obxText), decoder))

// 	if err == nil {
// 		result["informe"] = string(obxUTF8)
// 		result["queueName"] = queueName
// 	}

// 	log.Println("INFO: Mensaje HL7 analizado a JSON:", result)

// 	return result
// }

// parseSegment es una función auxiliar para analizar segmentos individuales
// y poblar el JSON resultante.
func parseSegment(parts []string, segmentMapping Segment, result map[string]interface{}, componentSeparator string) {
	for _, field := range segmentMapping.Values {
		fieldIndex := field.Component[0]
		componentIndex := field.Component[1]

		if fieldIndex < len(parts) {
			components := strings.Split(parts[fieldIndex], componentSeparator)

			if componentIndex < len(components) && components[componentIndex] != "" {
				result[field.Field] = components[componentIndex]
			} else if field.Default != "" {
				result[field.Field] = field.Default
			}
		} else if field.Default != "" {
			result[field.Field] = field.Default
		}
	}
}

// TODO: esto debería mejorarse. Este ACK debería construirse usando la configuración de la DB
const ack_fmt = "MSH|^~\\&|ANDES|HPN|MOSAIQ|HPN|%s||ACK|%s|P|2.5\nMSA|AA|%s"

// buildHL7ACK construye un mensaje HL7 ACK.
func buildHL7ACK() string {
	timestamp := time.Now().Format("20060102150405") // Formato AAAAMMDDHHMMSS

	// Generar un messageID usando el timestamp + 3 dígitos aleatorios
	randomNumber := rand.Intn(1000) // Número aleatorio entre 0 y 999
	messageID := fmt.Sprintf("%s%03d", timestamp, randomNumber)

	ackMessage := fmt.Sprintf(
		ack_fmt,
		timestamp, messageID, messageID,
	)
	return ackMessage
}
