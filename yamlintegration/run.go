package yamlintegration

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// Run ejecuta la suite de integración YAML data-driven. Asume que el servicio bajo prueba
// ya está corriendo en cfg.BaseURL y que, si hay precondiciones Mongo, cfg.Database está conectado.
//
// Variables de entorno que afectan el comportamiento:
//   - INTEGRATION_TESTS_DIR: carpeta raíz de YAML (override de cfg.TestsDir)
//   - VERBOSE=true:           log de status + body por cada paso HTTP
//   - INTEGRATION_HTTP_TIMEOUT: timeout por petición (ej. "90s")
func Run(t *testing.T, cfg Config) {
	t.Helper()

	allowedEnvs := cfg.AllowedEnvs
	if len(allowedEnvs) == 0 {
		allowedEnvs = []string{"dev", "local"}
	}
	env := strings.ToLower(cfg.AppEnv)
	allowed := false
	for _, a := range allowedEnvs {
		if env == strings.ToLower(a) {
			allowed = true
			break
		}
	}
	if !allowed {
		t.Fatalf(red+"🚨 BLOQUEADO: Test solo permitido en %v. ENV actual: %s"+reset, allowedEnvs, cfg.AppEnv)
	}

	t.Logf(cyan+"🚀 Ejecutando tests contra: %s"+reset, cfg.BaseURL)

	testsDir := cfg.TestsDir
	if testsDir == "" {
		testsDir = os.Getenv("INTEGRATION_TESTS_DIR")
	}
	if testsDir == "" {
		testsDir = "./integration_tests"
	}

	testCases, err := LoadTestCases(testsDir)
	if err != nil {
		t.Fatalf(red+"Error cargando tests: %v"+reset, err)
	}
	t.Logf(cyan+"📂 Casos YAML desde: %s"+reset, testsDir)
	verbose := strings.EqualFold(os.Getenv("VERBOSE"), "true")

	// Dates first, then caller's custom vars override (so BFF can override FUTURE_DATE etc. if needed)
	baseVars := mergeStringMaps(DateVariables(time.Now()), cfg.DefaultVars)

	for _, tc := range testCases {
		tc := resolveTestCaseVars(tc, baseVars)
		t.Run(tc.Name, func(t *testing.T) {
			t.Logf(gray+"📝 Descripción: %s"+reset, tc.Description)

			if cfg.Database != nil {
				if err := SetupDBPreconditions(t, cfg.Database, tc.Preconditions); err != nil {
					t.Fatalf(red+"Precondiciones fallidas: %v"+reset, err)
				}
			} else if len(tc.Preconditions) > 0 {
				t.Fatalf(red+"El caso tiene precondiciones pero Config.Database es nil"+reset)
			}

			runtimeVars := make(map[string]string)
			steps := tc.integrationSteps()

			for i, step := range steps {
				stepLabel := ""
				if len(steps) > 1 {
					stepLabel = fmt.Sprintf(" [paso %d/%d]", i+1, len(steps))
				}

				mergedVars := mergeStringMaps(baseVars, runtimeVars)
				resolvedStep := resolveTestStep(step, mergedVars)

				resp, err := ExecuteHTTPRequest(cfg.BaseURL, resolvedStep.Execution.Method, resolvedStep.Execution.Endpoint, resolvedStep.Execution.Headers, resolvedStep.Execution.Body)
				if err != nil {
					t.Fatalf(red+"Error en petición HTTP%s: %v. ¿Está el servidor corriendo en %s?"+reset, stepLabel, err, cfg.BaseURL)
				}
				if verbose {
					t.Logf(cyan+"HTTP%s %s %s -> %d | body: %s"+reset, stepLabel, resolvedStep.Execution.Method, resolvedStep.Execution.Endpoint, resp.StatusCode, string(resp.Body))
				}

				if resp.StatusCode != resolvedStep.ValidateResponse.ExpectedStatus {
					msg := fmt.Sprintf("Status esperado %d, obtenido %d%s.", resolvedStep.ValidateResponse.ExpectedStatus, resp.StatusCode, stepLabel)
					if verbose {
						msg = fmt.Sprintf("%s Body: %s", msg, string(resp.Body))
					}
					t.Error(red + msg + reset)
				}

				if resolvedStep.ValidateResponse.ExpectedBody != nil ||
					len(resolvedStep.ValidateResponse.ArrayContains) > 0 ||
					len(resolvedStep.ValidateResponse.ArrayNotContains) > 0 ||
					len(resolvedStep.ValidateResponse.ArrayMinLen) > 0 {
					var actualBody map[string]interface{}
					if err := json.Unmarshal(resp.Body, &actualBody); err != nil {
						msg := fmt.Sprintf("Error parseando respuesta%s: %v.", stepLabel, err)
						if verbose {
							msg = fmt.Sprintf("%s Body: %s", msg, string(resp.Body))
						}
						t.Error(red + msg + reset)
					} else {
						if resolvedStep.ValidateResponse.ExpectedBody != nil {
							validatePartialBody(t, "", resolvedStep.ValidateResponse.ExpectedBody, actualBody)
						}
						validateResponseArrayRules(t, stepLabel, resolvedStep.ValidateResponse, actualBody)
					}
				}

				if resp.StatusCode == resolvedStep.ValidateResponse.ExpectedStatus && len(step.Capture) > 0 {
					applyCaptureFromResponse(resp.Body, step.Capture, runtimeVars)
				}
			}

			if cfg.Database != nil && len(tc.ExpectedDBState) > 0 {
				if err := ValidateDBState(cfg.Database, tc.ExpectedDBState); err != nil {
					t.Errorf(red+"Validación DB fallida: %v"+reset, err)
				}
			}
		})
	}
}

// applyCaptureFromResponse extrae valores del JSON de respuesta y los guarda en runtimeVars
// para usarlos en pasos siguientes via {{NOMBRE}}.
func applyCaptureFromResponse(body []byte, capture map[string]string, runtimeVars map[string]string) {
	if len(capture) == 0 {
		return
	}
	var root map[string]interface{}
	if err := json.Unmarshal(body, &root); err != nil {
		return
	}
	for varName, path := range capture {
		if v, ok := getByJSONPath(root, path); ok {
			runtimeVars[varName] = v
		}
	}
}

func getByJSONPath(root map[string]interface{}, path string) (string, bool) {
	var cur interface{} = root
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			continue
		}
		m, ok := cur.(map[string]interface{})
		if !ok {
			return "", false
		}
		cur, ok = m[part]
		if !ok {
			return "", false
		}
	}
	return interfaceToString(cur)
}

func interfaceToString(v interface{}) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t)), true
		}
		return fmt.Sprintf("%g", t), true
	case bool:
		return fmt.Sprintf("%t", t), true
	case nil:
		return "", false
	default:
		return fmt.Sprintf("%v", t), true
	}
}
