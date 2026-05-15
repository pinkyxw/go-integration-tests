package yamlintegration

// Precondition describe una operación de base de datos a ejecutar antes del test.
// Acciones soportadas: "delete", "insert", "upsert".
type Precondition struct {
	Collection string                 `yaml:"collection"`
	Action     string                 `yaml:"action"`
	Filter     map[string]interface{} `yaml:"filter"`
	Data       map[string]interface{} `yaml:"data"`
}

// Execution describe la petición HTTP a realizar.
type Execution struct {
	Method   string                 `yaml:"method"`
	Endpoint string                 `yaml:"endpoint"`
	Headers  map[string]string      `yaml:"headers"`
	Body     map[string]interface{} `yaml:"body"`
}

// ValidateResponse describe las validaciones sobre la respuesta HTTP.
//
// ArrayContains: para cada path con puntos (ej. "payload.events"), cada elemento de la lista
// debe coincidir con al menos un elemento del array en la respuesta (match parcial de map).
// Un matcher puede ser un map con la clave reservada "_match_any": lista de maps; basta que
// alguna alternativa coincida con algún elemento del array.
//
// ArrayNotContains: para cada path, ningún elemento del array de respuesta debe cumplir
// el match parcial de cada matcher listado.
//
// ArrayMinLen: longitud mínima del array en cada path.
type ValidateResponse struct {
	ExpectedStatus   int                      `yaml:"expected_status"`
	ExpectedBody     map[string]interface{}   `yaml:"expected_body"`
	ArrayContains    map[string][]interface{} `yaml:"array_contains"`
	ArrayNotContains map[string][]interface{} `yaml:"array_not_contains"`
	ArrayMinLen      map[string]int           `yaml:"array_min_len"`
}

// ExpectedDBState describe el estado esperado de un documento en la base de datos tras el test.
type ExpectedDBState struct {
	Collection   string                 `yaml:"collection"`
	Filter       map[string]interface{} `yaml:"filter"`
	ExpectedData map[string]interface{} `yaml:"expected_data"`
}

// TestStep encadena execution + validate_response en orden dentro de un mismo caso YAML.
// Capture: tras un paso exitoso (status esperado), asigna valores desde el JSON de respuesta
// a variables para {{NOMBRE}} en pasos siguientes. Ej: PAYMENT_ID: payload.payment_id
type TestStep struct {
	Execution        Execution         `yaml:"execution"`
	ValidateResponse ValidateResponse  `yaml:"validate_response"`
	Capture          map[string]string `yaml:"capture"`
}

// TestCase representa un caso de prueba completo cargado desde un archivo YAML.
// Soporta tanto el formato legacy (execution/validate_response directos) como
// el formato multi-paso (steps[]).
type TestCase struct {
	Name             string            `yaml:"name"`
	Description      string            `yaml:"description"`
	Preconditions    []Precondition    `yaml:"preconditions"`
	Steps            []TestStep        `yaml:"steps"`
	Execution        Execution         `yaml:"execution"`
	ValidateResponse ValidateResponse  `yaml:"validate_response"`
	ExpectedDBState  []ExpectedDBState `yaml:"expected_db_state"`
}

// integrationSteps devuelve los pasos HTTP: steps: en YAML o un único paso legacy.
func (tc TestCase) integrationSteps() []TestStep {
	if len(tc.Steps) > 0 {
		return tc.Steps
	}
	return []TestStep{{Execution: tc.Execution, ValidateResponse: tc.ValidateResponse}}
}
