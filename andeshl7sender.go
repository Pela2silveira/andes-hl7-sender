package main

import (
    "fmt"
    "log"
)

// El registro de salud tiene formato libre
type HealthRecord map[string]interface{}


type HL7Destination struct {
	IPAddress string `bson:"ipAddress"`
	Port      int    `bson:"port"`
}

type Config struct {
    RabbitmqURI       string    `json:"rabbitmqURI"` 
	QueueName         string    `json:"queueName"`
	MongodbURI        string    `json:"mongodbURI"`
	MongodbDatabase   string    `json:"mongodbDatabase"`
	MongodbCollection string    `json:"mongodbCollection"`
	HL7Config         HL7Config `bson:"hl7Config"`

} 

type HL7Config struct {
	QueueName       string           `bson:"queueName"`
	HL7Destinations []HL7Destination `bson:"hl7Destinations"`
	Mapping         Mapping          `bson:"mapping"`
}

// El Mapeo debe respetar esta estructura
type Mapping struct {
    Format     string             `json:"format"`
    Delimiters Delimiters         `json:"delimiters"`
    Mappings   map[string]Segment `json:"mappings"`
}

type Delimiters struct {
    FieldSeparator        string `json:"fieldSeparator"`
    ComponentSeparator    string `json:"componentSeparator"`
    SubcomponentSeparator string `json:"subcomponentSeparator"`
    EscapeCharacter       string `json:"escapeCharacter"`
    RepetitionCharacter   string `json:"repetitionCharacter"`
    SegmentSeparator      string `json:"segmentSeparator"`
}

type Segment struct {
    Values []Field `json:"values"`
}

type Field struct {
    Field     string  `json:"field"`
    Component []int   `json:"component"`
    Default   string  `json:"default,omitempty"`
}

func main() {

    err := loadConfig()
    if err != nil {
        log.Fatalf("Error cargando configuración: %v", err)
    }

    fmt.Printf("Configuración cargada: %+v\n", config)

	// Obtener datos del paciente desde RabbitMQ
	err = consumeRecords()
	if err != nil {
		fmt.Printf("Error recuperando registros del HIS: %v\n", err)
		return
	}

}

