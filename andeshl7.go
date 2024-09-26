package main

import (
    "fmt"
    "log"
    "sync"
)

// El registro de salud tiene formato libre
type HealthRecord map[string]interface{}


type HL7Destination struct {
	IPAddress string `bson:"ipAddress"`
	Port      int    `bson:"port"`
}

type Config struct {
    RabbitmqURI        string      `json:"rabbitmqURI"` 
	ConsumerQueueNames []string    `json:"consumerQueueNames"`
	MongodbURI         string      `json:"mongodbURI"`
	MongodbDatabase    string      `json:"mongodbDatabase"`
	MongodbCollection  string      `json:"mongodbCollection"`
	ConsumerHL7Configs []HL7Config `bson:"hl7Config"`

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
    var wg sync.WaitGroup

    for _, cfg := range config.ConsumerHL7Configs {
        wg.Add(1) // Incrementa el contador del WaitGroup
        go consumeRecords(&wg, cfg)
    }
    // wg.Add(2) // Indicar que hay una gorutina más
	// // Obtener datos del paciente desde RabbitMQ
	// go consumeRecords(&wg)
    // go produceRecords(&wg)

    wg.Wait() // Esperar a que todas las gorutinas terminen

}

