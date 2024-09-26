package main

import (
    "fmt"
//    "log"
    "os"
    "strings"
	"time"
)

// Function to write the HL7 message to a file
func writeHL7ToFile(hl7Message, fileName string) error {
	// Create or open the file for writing
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

// 	// Write the HL7 message to the file
	_, err = file.WriteString(hl7Message)
	if err != nil {
		return err
	}

	return nil
}

// Helper function to retrieve values from the Record based on the hierarchical path
func getValueFromPath(record HealthRecord, path []string) (interface{}, bool) {
    if len(path) == 0 {
        return nil, false // Si el path está vacío, no hay nada que buscar
    }

    // Tratar el primer elemento del path
    firstKey := path[0]
    var current interface{} = record

    if firstMap, ok := current.(HealthRecord); ok {
        if value, exists := firstMap[firstKey]; exists {
            current = value
        } else {
            //fmt.Printf("Key '%s' does not exist in first map.\n", firstKey)
            return nil, false // Si la clave no existe, retornar false
        }
    } else {
        fmt.Printf("Current is not a map at key: %s\n", firstKey)
        return nil, false // Si current no es un map[string]interface{}, retornar false
    }

    for _, key := range path[1:] {
        if currentMap, ok := current.(map[string]interface{}); ok {
            if value, exists := currentMap[key]; exists {
                current = value
            } else {
                fmt.Printf("Key '%s' does not exist in current map.\n", key)
                return nil, false // Si la clave no existe, retornar false
            }
        } else {
            fmt.Printf("Current is not a map at key: %s\n", key)
            return nil, false // Si current no es un map[string]interface{}, retornar false
        }
    }

	return current, true
}

// Function to generate the HL7 message
func generateHL7(mapping Mapping, hisRecord HealthRecord) string {//*bytes.Buffer {

	var hl7Message         strings.Builder
//    var hl7Message         bytes.Buffer
	var lastFieldIndex     int
	var lastComponentIndex int
    
    // MSH segment
    hl7Message.WriteString("MSH")
    hl7Message.WriteString(mapping.Delimiters.FieldSeparator)
    hl7Message.WriteString(mapping.Delimiters.ComponentSeparator)
    hl7Message.WriteString(mapping.Delimiters.SubcomponentSeparator)
    hl7Message.WriteString(mapping.Delimiters.EscapeCharacter)
    hl7Message.WriteString(mapping.Delimiters.RepetitionCharacter)

    for _, segment := range mapping.Mappings {
        if segment.Segment != "msh" {
            hl7Message.WriteString(strings.ToUpper(segment.Segment))
        }

        lastFieldIndex = 0
        lastComponentIndex = 1

        hl7Message.WriteString(mapping.Delimiters.FieldSeparator)

        for _, field := range segment.Values {
            var componentValue string
            if field.Field == "date_time" {
                // Insertar la fecha y hora actual en el formato especificado
                componentValue = time.Now().Format("20060102150405")
            } else {
                // Split the field to get the hierarchical path (e.g., "paciente.apellido")
                fieldPath := strings.Split(field.Field, ".")
                // Try to get the value based on the hierarchical path
                value, exists := getValueFromPath(hisRecord, fieldPath)
                if exists {
                    componentValue = fmt.Sprintf("%v", value) // Convert to string
                } else {
                    componentValue = field.Default // Use default if the value is not found
                }
            }

            // Handling component and field separation
            fieldIndex := field.Component[0]
            componentIndex := field.Component[1]

            if fieldIndex > lastFieldIndex {
                for fieldIndex > lastFieldIndex {
                    hl7Message.WriteString(mapping.Delimiters.FieldSeparator)
                    lastFieldIndex++
                }
                lastComponentIndex = 1
            }

            for componentIndex > lastComponentIndex {
                hl7Message.WriteString(mapping.Delimiters.ComponentSeparator)
                lastComponentIndex++
            }

            hl7Message.WriteString(componentValue)
        }
        hl7Message.WriteString("\r")
    }
	//return &hl7Message
    writeHL7ToFile(hl7Message.String(), "hl7_message.txt")
    lines := strings.Split(hl7Message.String(), "\r")
    for _, line := range lines {
        if line != "" { // Verifica que no sea una línea vacía
            fmt.Println(line)
        }
    }    
    return hl7Message.String()
}
