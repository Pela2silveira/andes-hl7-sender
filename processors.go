package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func consumeRecords(cfg HL7Config) {

	ctx := context.Background() // Contexto para operaciones de MongoDB

	// Aquí 'cfg.ID' contendrá el _id del documento HL7Config si fue cargado desde la DB
	if cfg.ID.IsZero() {
		log.Printf("WARNING: HL7Config no tiene un '_id' válido. No se podrá actualizar el array MsgQueue para esta configuración. Config: %+v\n", cfg)
		return // O manejar este caso como error fatal si siempre se espera un ID
	}

	for _, hisRecord := range cfg.MsgQueue { // hisRecord es un elemento del array MsgQueue
		hl7Message := generateHL7(cfg.Mapping, hisRecord)
		successfullySent := false // Bandera para saber si se envió correctamente a al menos un destino

		// Lógica de envío HL7 a destinos... (Se mantiene igual)
		for _, dest := range cfg.HL7Destinations {
			log.Printf("INFO: Enviando HL7 a %s:%s\n", dest.IPAddress, dest.Port)
			address := net.JoinHostPort(dest.IPAddress, dest.Port)

			conn, err := net.Dial("tcp", address)
			if err != nil {
				log.Printf("ERROR: Error al conectar con el servidor %s: %v\n", address, err)
				continue
			}
			defer conn.Close() // Cierra la conexión cuando esta iteración del bucle interno termine

			response, err := sendHL7Message(conn, hl7Message)
			if err != nil {
				log.Printf("ERROR: Error enviando mensaje HL7 a %s: %v\n", address, err)
				continue
			}
			log.Println("INFO: Respuesta de envío:", response)

			response, err = receiveHL7Message(conn)
			if err != nil {
				log.Printf("ERROR: Error recibiendo respuesta HL7 de %s: %v\n", address, err)
				continue
			}

			writeHL7ToFile(response, "hl7_response.txt")

			log.Printf("INFO: Respuesta completa del servidor:\n")
			lines := strings.Split(response, "\r")
			isMSA_AA := false
			for _, line := range lines {
				if line != "" {
					log.Println(line)
					if strings.HasPrefix(line, "MSA|AA") {
						isMSA_AA = true
						successfullySent = true // Al menos un destino aceptó el mensaje
					}
				}
			}

			isERR := false
			for _, line := range lines {
				if line != "" {
					log.Println(line)
					if strings.HasPrefix(line, "ERR|") {
						isERR = true
						successfullySent = false // Al menos un destino aceptó el mensaje
					}
				}
			}

			if isMSA_AA && !isERR {
				log.Println("INFO: Mensaje aceptado por el servidor")
			} else {
				log.Println("WARNING: Mensaje rechazado por el servidor")
			}
		}

		// Si el mensaje fue enviado correctamente a al menos un destino, elimínalo del array MsgQueue en la DB
		if successfullySent {
			// 1. Obtener el 'id' del HealthRecord a eliminar del array
			//healthRecordID, ok := ["id"]
			//idPath :=
			healthRecordID, err := jsonpath.Get("id", hisRecord)
			if err != nil {
				log.Printf("ERROR: El HealthRecord no contiene una clave 'id' válida de tipo primitive.ObjectID para eliminación del array: %+v\n", hisRecord)
				continue // No podemos eliminar del array si no tenemos el ID del elemento
			}

			// 2. Construir el filtro para encontrar el documento HL7Config principal
			// Usamos el _id del HL7Config
			filter := bson.M{"_id": cfg.ID}

			// 3. Construir la operación de actualización usando $pull
			// $pull remueve todos los elementos de un array que coinciden con una condición.
			// Aquí, removemos el HealthRecord del array "msgQueue" donde su "id" sea igual a healthRecordID.
			update := bson.M{
				"$pull": bson.M{
					"msgQueue": bson.M{
						"id": healthRecordID, // 'id' es la clave dentro del HealthRecord anidado
					},
				},
			}

			// 4. Ejecutar la operación de actualización
			if mongoCollection != nil {
				result, err := mongoCollection.UpdateOne(ctx, filter, update)
				if err != nil {
					log.Printf("ERROR: Error actualizando documento HL7Config para eliminar mensaje (HL7Config ID: %s, MsgQueue ID: %s): %v\n", cfg.ID.Hex(), healthRecordID, err)
				} else {
					log.Printf("INFO: Mensaje con ID %s eliminado del array MsgQueue del HL7Config %s. Documentos modificados: %d\n", healthRecordID, cfg.ID.Hex(), result.ModifiedCount)
				}
			} else {
				log.Println("WARNING: La conexión a la colección de MongoDB no está disponible para actualizar el documento HL7Config.")
			}
		}
	}
}

// saveBase64PDF decodifica base64 y guarda el archivo como PDF
func saveBase64PDF(base64Content, outputPath string) error {
	// Decodificar base64
	data, err := base64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		log.Fatal("❌ Error de base64:", err)
	}
	err = ioutil.WriteFile("/tmp/test.pdf", data, 0644)
	if err != nil {
		log.Fatal("❌ Error escribiendo PDF:", err)
	}

	log.Println("PDF guardado en:", outputPath)
	return nil
}

// sendToAndes toma el 'record' (que es el resultado del parseo HL7)
// y lo añade al array 'msgQueue' del documento HL7Config correspondiente.
func sendToAndes(ctx context.Context, cfg HL7Config, recordToInsert map[string]interface{}) {
	log.Printf("INFO: [sendToAndes] Preparando para agregar registro al MsgQueue de HL7Config ID: %s\n", cfg.ID.Hex())

	if cfg.ID.IsZero() {
		log.Printf("ERROR: [sendToAndes] HL7Config no tiene un '_id' válido. No se puede actualizar MsgQueue. Config: %+v\n", cfg)
		return
	}

	// Es CRUCIAL que cada registro en MsgQueue tenga un identificador único
	// si luego `consumeRecords` va a intentar eliminarlo usando ese 'id'.
	// Si 'parseHL7Message' no añade un 'id' de tipo primitive.ObjectID,
	// deberíamos añadirlo aquí.
	if _, ok := recordToInsert["id"]; !ok {
		recordToInsert["id"] = primitive.NewObjectID()
		log.Printf("INFO: [sendToAndes] Nuevo primitive.ObjectID generado y añadido al registro: %s\n", recordToInsert["id"].(primitive.ObjectID).Hex())
	} else if _, ok := recordToInsert["id"].(primitive.ObjectID); !ok {
		// Si el id existe pero no es un ObjectID, podría ser un problema para el $pull futuro.
		// Considerar convertirlo o loguear una advertencia.
		// Por ahora, si ya existe un 'id', lo dejamos como está pero avisamos si no es ObjectID.
		log.Printf("WARNING: [sendToAndes] El registro ya tiene una clave 'id', pero no es primitive.ObjectID. Tipo: %T, Valor: %v. Esto podría causar problemas en consumeRecords.\n", recordToInsert["id"], recordToInsert["id"])
	}

	// 1. Construir el filtro para encontrar el documento HL7Config principal
	filter := bson.M{"_id": cfg.ID}

	// 2. Construir la operación de actualización usando $push
	// $push añade el 'recordToInsert' al array "msgQueue".
	update := bson.M{
		"$push": bson.M{
			"msgQueue": recordToInsert,
		},
	}

	// 3. Ejecutar la operación de actualización
	if mongoCollection != nil {
		result, err := mongoCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			log.Printf("ERROR: [sendToAndes] Error actualizando documento HL7Config (ID: %s) para agregar mensaje a MsgQueue: %v\n", cfg.ID.Hex(), err)
		} else {
			if result.ModifiedCount > 0 {
				log.Printf("INFO: [sendToAndes] Mensaje agregado exitosamente al array MsgQueue del HL7Config %s. Documentos modificados: %d\n", cfg.ID.Hex(), result.ModifiedCount)
			} else if result.MatchedCount == 0 {
				log.Printf("WARNING: [sendToAndes] No se encontró el documento HL7Config con ID %s para agregar el mensaje al MsgQueue.\n", cfg.ID.Hex())
			} else {
				log.Printf("WARNING: [sendToAndes] El documento HL7Config con ID %s fue encontrado pero no modificado. Esto es inusual con $push a menos que haya un error concurrente no reportado.\n", cfg.ID.Hex())
			}
		}
	} else {
		log.Println("ERROR: [sendToAndes] La conexión a la colección de MongoDB no está disponible para actualizar el documento HL7Config.")
	}
}

// produceRecords inicia un servidor MLLP para recibir mensajes HL7 entrantes.
func produceRecords(ctx context.Context, cfg HL7Config) {
	// Crear el servidor y escuchar en el puerto configurado
	address := fmt.Sprintf("%s:%s", cfg.HL7Destinations[0].IPAddress, cfg.HL7Destinations[0].Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Printf("ERROR: Error al crear el servidor para el productor '%s': %v\n", cfg.QueueName, err)
		return
	}
	defer listener.Close()

	log.Printf("INFO: Servidor MLLP del productor '%s' escuchando en %s\n", cfg.QueueName, address)

	// Canal para señalar errores de aceptación que podrían requerir un reinicio o un manejo más severo
	acceptErrChan := make(chan error, 1)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Verificar si el error se debe a que el listener está cerrado
				select {
				case <-ctx.Done():
					log.Printf("INFO: Productor '%s': Listener detenido debido a la cancelación del contexto.\n", cfg.QueueName)
					return // Salir del bucle de aceptación si el contexto ha terminado
				default:
					log.Printf("ERROR: Productor '%s': Error al aceptar conexión: %v. Reintentando...\n", cfg.QueueName, err)
					acceptErrChan <- err        // Enviar error al bucle principal para un posible manejo
					time.Sleep(1 * time.Second) // Pequeño retraso para evitar un bucle ocupado en errores persistentes
					continue
				}
			}

			// Manejar cada conexión en una nueva goroutine para concurrencia
			go func(c net.Conn) {
				defer c.Close() // Cerrar la conexión cuando termine su goroutine manejadora
				log.Printf("INFO: Productor '%s': Conexión aceptada desde %s\n", cfg.QueueName, c.RemoteAddr())

				response, err := receiveHL7Message(c)
				if err != nil {
					log.Printf("ERROR: Productor '%s': Error al recibir mensaje HL7 de %s: %v\n", cfg.QueueName, c.RemoteAddr(), err)
					return
				}
				writeHL7ToFile(response, "hl7_received.txt")
				log.Printf("INFO: Productor '%s': Recibido de %s: %s\n", cfg.QueueName, c.RemoteAddr(), response)

				result := parseHL7Message(response, cfg.Mapping, cfg.QueueName)

				jsonResult, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					log.Printf("ERROR: Productor '%s': Error al convertir a JSON para %s: %v\n", cfg.QueueName, c.RemoteAddr(), err)
					return
				}
				log.Printf("INFO: Productor '%s': Analizado para %s: %s\n", cfg.QueueName, c.RemoteAddr(), string(jsonResult))

				sendToAndes(ctx, cfg, result)

				_, err = sendHL7Message(c, buildHL7ACK())
				if err != nil {
					log.Printf("ERROR: Productor '%s': Error al enviar ACK a %s: %v\n", cfg.QueueName, c.RemoteAddr(), err)
					return
				}
				log.Printf("INFO: Productor '%s': ACK enviado a %s\n", cfg.QueueName, c.RemoteAddr())

			}(conn)
		}
	}()

	// Bloquear hasta que el contexto sea cancelado
	<-ctx.Done()
	log.Printf("INFO: Productor '%s': Cerrando goroutine del listener.\n", cfg.QueueName)
}
