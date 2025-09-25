package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	configFile = "config.yaml"
)

// Variables globales para la configuración y la conexión a MongoDB
var (
	config          Config
	mongoClient     *mongo.Client
	mongoCollection *mongo.Collection // La colección principal donde se guardan las HL7Configs
)

// initConfigAndMongoDB inicializa Viper y la conexión a MongoDB
func initConfigAndMongoDB() error {
	// Configurar Viper
	viper.SetConfigFile(configFile)
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error leyendo archivo de configuración: %v", err)
	}

	config.MongodbURI = viper.GetString("mongodbURI")
	config.MongodbDatabase = viper.GetString("mongodbDatabase")
	config.MongodbCollection = viper.GetString("mongodbCollection")

	// Conectar a MongoDB una sola vez al inicio de la aplicación
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // Asegura que el contexto se cancele

	clientOptions := options.Client().ApplyURI(config.MongodbURI)
	var err error
	mongoClient, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("error conectando a MongoDB: %v", err)
	}

	// Hacer un ping para verificar la conexión
	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		mongoClient.Disconnect(context.Background()) // Desconectar si el ping falla
		return fmt.Errorf("error haciendo ping a MongoDB: %v", err)
	}

	// Asignar la colección globalmente
	mongoCollection = mongoClient.Database(config.MongodbDatabase).Collection(config.MongodbCollection)

	log.Println("INFO: Conexión a MongoDB establecida exitosamente.")
	return nil
}

// loadConfig ahora solo carga los datos de configuración HL7 desde la DB
// Asume que mongoCollection ya está inicializado y activo.
func loadConfig() error {
	if mongoCollection == nil {
		return fmt.Errorf("la colección de MongoDB no está inicializada. Llamar a initConfigAndMongoDB primero")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Limpiar configuraciones anteriores antes de cargar nuevas
	config.ConsumerHL7Configs = nil
	config.ProducerHL7Configs = nil

	filter := bson.M{} // Filtro para obtener todas las configuraciones HL7
	cursor, err := mongoCollection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("error recuperando las configuraciones HL7 de la DB: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var hl7Config HL7Config
		if err := cursor.Decode(&hl7Config); err != nil {
			return fmt.Errorf("error decodificando configuración HL7 de la DB: %v", err)
		}
		if hl7Config.Direccion == "inbound" {
			config.ProducerHL7Configs = append(config.ProducerHL7Configs, hl7Config)
		} else if hl7Config.Direccion == "outbound" {
			config.ConsumerHL7Configs = append(config.ConsumerHL7Configs, hl7Config)
		}
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("error en el cursor al cargar configuraciones HL7: %v", err)
	}

	log.Println("INFO: Configuraciones HL7 cargadas desde la base de datos.")
	return nil
}
