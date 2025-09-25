## Documento de Integración entre ANDES y MOSAIC

### 1. Introducción
Este documento describe los lineamientos, objetivos y aspectos técnicos implementados en la integración entre los sistemas ANDES y MOSAIC. El objetivo principal de esta integración es mejorar el flujo de información clínica y administrativa entre ambas plataformas, asegurando interoperabilidad, trazabilidad y eficiencia. Aunque la integración es extensible, en este caso su uso estará enfocado exclusivamente a la gestión de prestaciones de radioterapia.

### 2. Objetivos de la integración
- Facilitar el intercambio de información entre ANDES y MOSAIC.
- Garantizar la consistencia de datos entre ambos sistemas.
- Minimizar duplicación de datos y errores manuales.
- Mejorar la continuidad de atención del paciente.

### 3. Descripción general de los sistemas
**ANDES:** Plataforma de gestión de salud que centraliza la historia clínica electrónica, turnos, administración de pacientes, entre otros módulos. ANDES cuenta con múltiples componentes, pero para esta integración se trabajará específicamente con los siguientes:
- **MPI (Master Patient Index):** Gestión de identidades y datos demográficos del paciente.
- **RUP (Registro Único de Prestaciones):** Registro de las prestaciones clínicas realizadas.
- **CITAS:** Gestión de turnos y agenda de atención.
- **HUDS (Historia Unificada Digital de Salud):** Consolidación de documentos clínicos accesibles.

**MOSAIC:** Sistema especializado en la gestión oncológica, utilizado para seguimiento de pacientes con cáncer, administración de tratamientos y estudios relacionados. Aunque el sistema es extensible, en este caso su uso estará enfocado exclusivamente a la gestión de prestaciones de radioterapia.

### 4. Escenarios de integración implementados
- Envío de datos de paciente desde ANDES hacia MOSAIC, utilizando mensajes HL7v2 ADT^A04.
- Disponibilidad de documentos generados en MOSAIC como CDA en ANDES, accesibles a través de la HUDS.

### 5. Consideraciones técnicas iniciales
- El protocolo de intercambio de información es **HL7 versión 2 (HL7v2)**.
- Los mensajes utilizados son exclusivamente **ADT** (para eventos relacionados a pacientes) y **MDM** (para gestión de documentos médicos o documentos relacionados).

#### Flujo ADT^A04
- En el caso de los mensajes **ADT**, se trabaja únicamente con el evento **ADT^A04**, siendo ANDES el iniciador de la comunicación ante un evento específico.
- Este evento está vinculado a una **prestación**, un **efector** y una **operación**, lo que constituye una terna estándar de configuración de disparadores en ANDES. 
  - **Prestación:** Consulta de radioterapia
  - **Efector:** Hospital Castro Rendón
  - **Operación:** Asignación de turno en el módulo CITAS
- La conjunción de estas tres variables genera el envío de un mensaje ADT^A04 con los datos del paciente y el nuevo identificador generado por ANDES.
- Este mensaje se emite **siempre** que se cumplan las condiciones mencionadas, actualizando los datos del paciente si alguno ha sido modificado.
- El sistema **creador del paciente es ANDES**, es decir, toda nueva identificación de paciente se origina desde este sistema.

#### Flujo MDM
- En el caso de los mensajes **MDM**, el proceso está configurado exclusivamente en la pestaña **Documentos** del sistema MOSAIC.
- En dicha pestaña, se puede configurar el **tipo de documento** que será objeto de envío.
- El envío del mensaje MDM se dispara únicamente cuando el documento es **aprobado por un profesional** dentro del sistema MOSAIC.
- El mensaje MDM se transmite hacia una **interfaz** que recibe el contenido del documento en **formato de texto plano** o **pdf**.
- Durante la transmisión, si el formato original es texto plano la **interfaz será responsable de embellecer el contenido** de convertirlo en un formato más legible.
- Posteriormente, el documento transformado es almacenado temporalmente en una **memoria intermedia**.
- Una **tarea programada** toma estos documentos y los convierte a formato **CDA** (Clinical Document Architecture), haciéndolos accesibles a través de la **HUDS** (Historia Única Digital de Salud) en ANDES.

#### Identificación del paciente
- Se define utiliza el DNI como identificador alternativo, debido a que el protocolo HL7v2 admite un máximo de 15 caracteres.
- El identificador original de ANDES excede ese límite, por lo que se envía la primera sección del mismo en un campo alternativo.s

