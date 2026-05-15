package yamlintegration

import (
	"os"
	"regexp"
	"strings"
)

// testVariablePattern coincide con {{NOMBRE}} donde NOMBRE es [A-Z0-9_]+.
var testVariablePattern = regexp.MustCompile(`\{\{\s*([A-Z0-9_]+)\s*\}\}`)

// mergeStringMaps combina base y overlay en un nuevo mapa; overlay tiene precedencia.
func mergeStringMaps(base, overlay map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

// replaceTestVariables sustituye {{KEY}} en value buscando primero en os.Getenv(KEY)
// y luego en vars. Si no se encuentra, deja el placeholder intacto.
func replaceTestVariables(value string, vars map[string]string) string {
	if value == "" {
		return value
	}
	return testVariablePattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := testVariablePattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		key := strings.TrimSpace(parts[1])
		if envValue := os.Getenv(key); envValue != "" {
			return envValue
		}
		if defaultValue, ok := vars[key]; ok {
			return defaultValue
		}
		return match
	})
}

func replaceVariablesInExecution(exec Execution, vars map[string]string) Execution {
	exec.Method = replaceTestVariables(exec.Method, vars)
	exec.Endpoint = replaceTestVariables(exec.Endpoint, vars)
	exec.Headers = replaceVariablesInStringMap(exec.Headers, vars)
	exec.Body = replaceVariablesInMap(exec.Body, vars)
	return exec
}

func replaceVariablesInStringMap(values map[string]string, vars map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	replaced := make(map[string]string, len(values))
	for key, value := range values {
		replaced[replaceTestVariables(key, vars)] = replaceTestVariables(value, vars)
	}
	return replaced
}

func replaceVariablesInMap(values map[string]interface{}, vars map[string]string) map[string]interface{} {
	if values == nil {
		return nil
	}
	replaced := make(map[string]interface{}, len(values))
	for key, value := range values {
		replaced[replaceTestVariables(key, vars)] = replaceVariablesInAny(value, vars)
	}
	return replaced
}

func replaceVariablesInAny(value interface{}, vars map[string]string) interface{} {
	switch typed := value.(type) {
	case string:
		return replaceTestVariables(typed, vars)
	case map[string]interface{}:
		return replaceVariablesInMap(typed, vars)
	case []interface{}:
		replaced := make([]interface{}, len(typed))
		for i, item := range typed {
			replaced[i] = replaceVariablesInAny(item, vars)
		}
		return replaced
	default:
		return value
	}
}

func replaceVariablesInPathMatchers(m map[string][]interface{}, vars map[string]string) map[string][]interface{} {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string][]interface{}, len(m))
	for path, matchers := range m {
		newMatchers := make([]interface{}, len(matchers))
		for i, matcher := range matchers {
			newMatchers[i] = replaceVariablesInAny(matcher, vars)
		}
		out[replaceTestVariables(path, vars)] = newMatchers
	}
	return out
}

func replaceVariablesInPathIntMap(m map[string]int, vars map[string]string) map[string]int {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]int, len(m))
	for path, n := range m {
		out[replaceTestVariables(path, vars)] = n
	}
	return out
}

// resolveTestCaseVars aplica sustitución de variables sobre los campos estáticos
// del TestCase (nombre, descripción, precondiciones, expected_db_state).
// Los pasos steps: se resuelven en runtime para soportar variables capturadas ({{PAYMENT_ID}}, etc.).
func resolveTestCaseVars(tc TestCase, vars map[string]string) TestCase {
	tc.Name = replaceTestVariables(tc.Name, vars)
	tc.Description = replaceTestVariables(tc.Description, vars)

	for i := range tc.Preconditions {
		tc.Preconditions[i].Collection = replaceTestVariables(tc.Preconditions[i].Collection, vars)
		tc.Preconditions[i].Action = replaceTestVariables(tc.Preconditions[i].Action, vars)
		tc.Preconditions[i].Filter = replaceVariablesInMap(tc.Preconditions[i].Filter, vars)
		tc.Preconditions[i].Data = replaceVariablesInMap(tc.Preconditions[i].Data, vars)
	}

	tc.Execution = replaceVariablesInExecution(tc.Execution, vars)
	tc.ValidateResponse.ExpectedBody = replaceVariablesInMap(tc.ValidateResponse.ExpectedBody, vars)
	tc.ValidateResponse.ArrayContains = replaceVariablesInPathMatchers(tc.ValidateResponse.ArrayContains, vars)
	tc.ValidateResponse.ArrayNotContains = replaceVariablesInPathMatchers(tc.ValidateResponse.ArrayNotContains, vars)
	tc.ValidateResponse.ArrayMinLen = replaceVariablesInPathIntMap(tc.ValidateResponse.ArrayMinLen, vars)

	for i := range tc.ExpectedDBState {
		tc.ExpectedDBState[i].Collection = replaceTestVariables(tc.ExpectedDBState[i].Collection, vars)
		tc.ExpectedDBState[i].Filter = replaceVariablesInMap(tc.ExpectedDBState[i].Filter, vars)
		tc.ExpectedDBState[i].ExpectedData = replaceVariablesInMap(tc.ExpectedDBState[i].ExpectedData, vars)
	}

	return tc
}

// resolveTestStep aplica sustitución de variables sobre un paso individual (usado en runtime
// para poder incorporar variables capturadas de pasos anteriores).
func resolveTestStep(step TestStep, vars map[string]string) TestStep {
	out := step
	out.Execution = replaceVariablesInExecution(step.Execution, vars)
	out.ValidateResponse.ExpectedBody = replaceVariablesInMap(step.ValidateResponse.ExpectedBody, vars)
	out.ValidateResponse.ArrayContains = replaceVariablesInPathMatchers(step.ValidateResponse.ArrayContains, vars)
	out.ValidateResponse.ArrayNotContains = replaceVariablesInPathMatchers(step.ValidateResponse.ArrayNotContains, vars)
	out.ValidateResponse.ArrayMinLen = replaceVariablesInPathIntMap(step.ValidateResponse.ArrayMinLen, vars)
	return out
}
