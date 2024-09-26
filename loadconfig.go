package main

import (
	"fmt"
	"context"
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
	config.QueueName = viper.GetString("queueName")
	
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

    filter := bson.M{"queueName": config.QueueName}

	err = collection.FindOne(ctx, filter).Decode(&config.HL7Config)
	if err != nil {
		return fmt.Errorf("Error recuperando la configuración: %v", err)
	}

	return nil
}