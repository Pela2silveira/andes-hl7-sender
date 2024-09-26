package main

import (
	"fmt"
	"context"
	"strings"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
    "github.com/spf13/viper"
)

const (
	configFile		  = "config.yaml"
)

var config Config
 

func loadConfig() (error) {

    // Configurar Viper
    viper.SetConfigFile(configFile)
	//viper.SetConfigName("config") // Nombre del archivo sin extensión
    //viper.SetConfigType("yaml")   // Tipo de archivo
    viper.AddConfigPath(".")      // Ruta donde buscar el archivo de configuración
    // Leer archivo de configuración
    if err := viper.ReadInConfig(); err != nil {
        return fmt.Errorf("Error leyendo archivo de configuración: %v", err)
    }
    // // Desempaquetar la configuración en la estructura
    // if err := viper.Unmarshal(&config); err != nil {
    //     return fmt.Errorf("Error desempaquetando configuración: %v", err)
    // }
	config.MongodbURI = viper.GetString("mongodbURI")
	config.RabbitmqURI = viper.GetString("rabbitmqURI")
	config.MongodbDatabase = viper.GetString("mongodbDatabase")
	config.MongodbCollection = viper.GetString("mongodbCollection")
	consumerQueueNames := viper.GetString("consumerQueueNames")
	config.ConsumerQueueNames = strings.Split(consumerQueueNames, ",")

	ctx := context.Background()
	// Conectar a MongoDB
	// credential := options.Credential{
	// 	Username: "hl7v2consumer",
	// 	Password: "zEr6l2U8Um",
	// }
	clientOptions := options.Client().ApplyURI(config.MongodbURI) //.SetAuth(credential)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("Error conectando a MongoDB: %v", err)
	}

	defer client.Disconnect(ctx)

	collection := client.Database(config.MongodbDatabase).Collection(config.MongodbCollection)

	for _, queueName := range config.ConsumerQueueNames {
		var hl7Config HL7Config
		filter := bson.M{"queueName": queueName}

		// Buscar la configuración para la cola actual
		err := collection.FindOne(ctx, filter).Decode(&hl7Config)
		if err != nil {
			return fmt.Errorf("Error recuperando la configuración para la cola %s: %v", queueName, err)
		}

		// Agregar la configuración encontrada al slice de configuraciones
		config.ConsumerHL7Configs = append(config.ConsumerHL7Configs, hl7Config)
	}	

	return nil
}