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

type Mapping struct {
    Format     string     `bson:"format"`
    Delimiters Delimiters `bson:"delimiters"`
    Mappings   []Segment  `bson:"mappings"` // Cambiado a una lista
}

type Delimiters struct {
    FieldSeparator        string `bson:"fieldSeparator"`
    ComponentSeparator    string `bson:"componentSeparator"`
    SubcomponentSeparator string `bson:"subcomponentSeparator"`
    EscapeCharacter       string `bson:"escapeCharacter"`
    RepetitionCharacter   string `bson:"repetitionCharacter"`
    SegmentSeparator      string `bson:"segmentSeparator"`
}

type Segment struct {
    Segment string  `bson:"segment"` // Este campo debe existir
    Values  []Field `bson:"values"`
}

type Field struct {
    Field     string `bson:"field"`
    Component []int  `bson:"component"`
    Default   string `bson:"default,omitempty"`
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

