package main

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Identifier struct {
	AssigningAuthority string `bson:"assigning_authority" json:"assigning_authority"`
	IDTypeCode         string `bson:"id_type_code" json:"id_type_code"`
	Identificador      string `bson:"identificador" json:"identificador"`
}

//	type HealthRecord struct {
//		Apellido        string       `bson:"apellido" json:"apellido"`
//		Direccion       string       `bson:"direccion" json:"direccion"`
//		FechaNacimiento string       `bson:"fechaNacimiento" json:"fechaNacimiento"` // Format: YYYYMMDD
//		ID              string       `bson:"id" json:"id"`                           // Consider using primitive.ObjectID if needed
//		Identifiers     []Identifier `bson:"identifiers" json:"identifiers"`
//		Localidad       string       `bson:"localidad" json:"localidad"`
//		Nombre          string       `bson:"nombre" json:"nombre"`
//		Provincia       string       `bson:"provincia" json:"provincia"`
//		Sexo            string       `bson:"sexo" json:"sexo"` // "F", "M", etc.
//		Telefono        string       `bson:"telefono" json:"telefono"`
//	}
type HealthRecord map[string]interface{}

// El registro de salud tiene formato libre
//type HealthRecord map[string]interface{}

type HL7Destination struct {
	IPAddress string `bson:"ipAddress"`
	Port      string `bson:"port"`
}

type Config struct {
	MongodbURI         string      `json:"mongodbURI"`
	MongodbDatabase    string      `json:"mongodbDatabase"`
	MongodbCollection  string      `json:"mongodbCollection"`
	ConsumerHL7Configs []HL7Config `bson:"hl7Config"`
	ProducerHL7Configs []HL7Config `bson:"hl7Config"`
}

type HL7Config struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	QueueName       string             `bson:"queueName"`
	HL7Destinations []HL7Destination   `bson:"hl7Destinations"`
	Mapping         Mapping            `bson:"mapping"`
	Direccion       string             `bson:"direccion"`
	MsgQueue        []HealthRecord     `bson:"msgQueue"`
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
