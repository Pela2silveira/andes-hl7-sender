package main

import (
	"context"
	"log"
	"sync"
	"time"
)

func main() {
	err := initConfigAndMongoDB()
	if err != nil {
		log.Fatalf("FATAL: Error fatal al inicializar configuración y MongoDB: %v", err)
	}
	// Asegurar que el cliente de MongoDB se desconecte al salir de main
	defer func() {
		if mongoClient != nil {
			log.Println("INFO: Desconectando de MongoDB...")
			mongoClient.Disconnect(context.Background())
		}
	}()

	// Crea un contexto para un apagado elegante de goroutines de larga duración (productores)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Asegura que se llame a cancel cuando main salga

	// Carga la configuración inicial para iniciar los productores
	err = loadConfig()
	if err != nil {
		log.Fatalf("FATAL: Error fatal al cargar configuración inicial para productores: %v", err)
	}

	// Inicia las goroutines de Productores (Listeners) de larga duración UNA SOLA VEZ
	// Estas goroutines deberían ejecutarse indefinidamente hasta que la aplicación se apague.
	var producerLongRunningWG sync.WaitGroup
	for _, cfg := range config.ProducerHL7Configs {
		producerLongRunningWG.Add(1)
		go func(cfg HL7Config) {
			defer producerLongRunningWG.Done() // Este Done() se llama cuando la goroutine anónima sale
			produceRecords(ctx, cfg)           // produceRecords en sí NO llama a wg.Done()
		}(cfg)
	}

	// Bucle principal para tareas periódicas: recargar la configuración e iniciar consumidores por lotes
	for {
		// Recarga periódicamente la configuración para los consumidores
		err := loadConfig()
		if err != nil {
			log.Printf("ERROR: Error cargando configuración para consumidores: %v. Reintentando en 5 segundos...\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("INFO: Configuración cargada y lista. Iniciando procesamiento de mensajes de consumo...\n")

		var consumerWG sync.WaitGroup // Este WaitGroup está correctamente dentro del alcance de esta iteración del bucle interno

		// Procesa las configuraciones HL7 del Consumidor (procesamiento por lotes)
		for _, cfg := range config.ConsumerHL7Configs {
			consumerWG.Add(1)
			go func(cfg HL7Config) {
				// printHL7ConfigLog(cfg)
				defer consumerWG.Done() // Este Done() se llama cuando la goroutine anónima sale
				consumeRecords(cfg)     // consumeRecords en sí NO llama a wg.Done()
			}(cfg)
		}

		consumerWG.Wait() // Espera a que todas las goroutines de procesamiento por lotes terminen

		log.Println("INFO: Todos los mensajes batch procesados. Re-evaluando en 30 segundos...")
		// Introduce un retardo antes de volver a leer la configuración e iniciar nuevos consumidores por lotes
		time.Sleep(30 * time.Second)
	}

	// En una aplicación real, es posible que desees esperar aquí a producerLongRunningWG
	// antes de que main salga realmente, especialmente si manejas señales del sistema operativo para el apagado.
	// producerLongRunningWG.Wait()
}

// func printHL7ConfigLog(cfg HL7Config) {
// 	jsonBytes, err := json.MarshalIndent(cfg, "", "  ")
// 	if err != nil {
// 		log.Printf("Error marshaling HL7Config: %v", err)
// 		return
// 	}
// 	log.Println("HL7Config content:\n", string(jsonBytes))
// }
