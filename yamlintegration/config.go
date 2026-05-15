package yamlintegration

import "go.mongodb.org/mongo-driver/v2/mongo"

// Config parametriza una ejecución del runner. El caller (p.ej. el BFF) es responsable
// de conectar a Mongo y resolver las URLs antes de llamar a Run.
type Config struct {
	// AppEnv es el valor del entorno actual (ej. "dev", "local", "staging").
	// Se compara contra AllowedEnvs para bloquear ejecuciones accidentales en producción.
	AppEnv string

	// AllowedEnvs lista los entornos donde el test puede correr.
	// Si está vacío se usa el default ["dev", "local"].
	AllowedEnvs []string

	// BaseURL es la URL base del servicio bajo prueba (ej. "http://localhost:8080").
	BaseURL string

	// Database es la conexión a MongoDB ya establecida. Si es nil y algún caso
	// tiene precondiciones o expected_db_state, el runner falla el test en cuestión.
	Database *mongo.Database

	// DefaultVars son variables de plantilla adicionales específicas del proyecto
	// (ej. constantes de cliente, IDs fijos). El runner las mezcla con las variables
	// de fecha genéricas que calcula internamente; DefaultVars tiene precedencia.
	DefaultVars map[string]string

	// TestsDir es la ruta a la carpeta raíz con los archivos YAML.
	// Si está vacío se usa INTEGRATION_TESTS_DIR o "./integration_tests".
	TestsDir string
}
