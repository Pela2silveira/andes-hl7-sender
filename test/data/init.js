// init-mongo.js

// Connect to the admin database to authenticate
db = db.getSiblingDB('admin');

// Create and switch to the database 'hospital'
db = db.getSiblingDB('hospital');
db.config.insert({
    "queueName": "adt04mosaiq",
    "hl7Destinations": [{
            "ipAddress": "localhost",
            "port": 1337 //1437        
        }
    ],
    "mapping": {
        "format": "hl7-2.5",
        "delimiters": {
            "fieldSeparator": "|",
            "componentSeparator": "^",
            "subcomponentSeparator": "&",
            "escapeCharacter": "\\",
            "repetitionCharacter": "~",
            "segmentSeparator": "\r"
        },
        "mappings": [
            {
                "segment": "msh",
                "values": [
                    {
                        "field": "sending_application",
                        "component": [0, 1],
                        "default": "Andes"
                    },
                    {
                        "field": "sending_facility",
                        "component": [1, 1],
                        "default": "HPN"
                    },
                    {
                        "field": "receiving_application",
                        "component": [2, 1],
                        "default": "MOSAIQ"
                    },
                    {
                        "field": "receiving_facility",
                        "component": [3, 1],
                        "default": "HPN"
                    },
                    {
                        "field": "date_time",
                        "component": [4, 1]
                    },
                    {
                        "field": "message_type",
                        "component": [6, 1],
                        "default": "ADT"
                    },
                    {
                        "field": "message_type_ref",
                        "component": [6, 2],
                        "default": "A04"
                    },
                    {
                        "field": "date_time",
                        "component": [7, 1]
                    },
                    {
                        "field": "processing_id",
                        "component": [8, 1],
                        "default": "P"
                    },
                    {
                        "field": "version_id",
                        "component": [9, 1],
                        "default": "2.5"
                    },
                    {
                        "field": "accept_type",
                        "component": [12, 1],
                        "default": "AL"
                    },
                    {
                        "field": "app_type",
                        "component": [13, 1],
                        "default": "NE"
                    },
                    {
                        "field": "country_code",
                        "component": [14, 1],
                        "default": "AR"
                    },
                    {
                        "field": "character_set",
                        "component": [15, 1],
                        "default": "UTF-8"
                    }
                ]
            },
            {
                "segment": "evn",
                "values": [
                    {
                        "field": "evn.id",
                        "component": [0, 1],
                        "default": "A04"
                    },
                    {
                        "field": "date_time",
                        "component": [1, 1]
                    },
                    {
                        "field": "date_time",
                        "component": [5, 1]
                    }
                ]
            },
            {
                "segment": "pid",
                "values": [
                    {
                        "field": "set_id",
                        "component": [0, 1],
                        "default": "1"
                    },
                    {
                        "field": "documento",
                        "component": [2, 1]
                    },
                    {
                        "field": "paciente.assigning_authority",
                        "component": [2, 4],
                        "default": "ANDES"

                    },
                    {
                        "field": "apellido",
                        "component": [4, 1]
                    },
                    {
                        "field": "nombre",
                        "component": [4, 2]
                    },
                    {
                        "field": "fechaNacimiento",
                        "component": [6, 1]
                    },
                    {
                        "field": "sexo",
                        "component": [7, 1]
                    },
                    {
                        "field": "telefono",
                        "component": [12, 1],
                        "default": ""
                    }
                ]
            },
            {
                "segment": "pv1",
                "values": [
                    {
                        "field": "set_id",
                        "component": [0, 1],
                        "default": "1"
                    },
                    {
                        "field": "patient_class",
                        "component": [2, 1],
                        "default": "O"
                    },
                    {
                        "field": "patient_type",
                        "component": [4, 1],
                        "default": "S"

                    }
                ]
            }

        ]
    }
})

// Insert data into the 'patients' collection
db.patients.insertMany([
    {
        "patient_id": "12345",
        "nombre": "John",
        "apellido": "Doe",
        "age": 30,
        "dni": "3148934921",
        "fechaNacimiento": "19870904",
        "sexo": "M",
        "diagnosis": "Flu",
        "treatment": "Rest and hydration"
    },
    {
        "patient_id": "67890",
        "dni": "3030238923",
        "fechaNacimiento": "19800704",
        "nombre": "Jane", 
        "apellido": "Smith",
        "age": 25,
        "sexo": "F",
        "diagnosis": "Migraine",
        "treatment": "Painkillers",
        "contacto":
        {
            "valor": "1121212121"
        }
    }
]);

db.createUser({
    user: "appuser",
    pwd: "appsecret",
    roles: [{ role: "readWrite", db: "hospital" }]
  })
  