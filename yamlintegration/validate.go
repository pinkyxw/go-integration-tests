package yamlintegration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const matchAnyKey = "_match_any"

// validatePartialBody verifica que expected es un subconjunto de actual (match parcial recursivo).
func validatePartialBody(t *testing.T, prefix string, expected, actual map[string]interface{}) {
	t.Helper()
	for k, ev := range expected {
		keyPath := k
		if prefix != "" {
			keyPath = prefix + "." + k
		}

		av, ok := actual[k]
		if !ok {
			t.Errorf(red+"Campo esperado %s no encontrado en el body"+reset, keyPath)
			continue
		}

		em, okE := ev.(map[string]interface{})
		am, okA := av.(map[string]interface{})
		if okE && okA {
			validatePartialBody(t, keyPath, em, am)
			continue
		}

		ej, _ := json.Marshal(ev)
		aj, _ := json.Marshal(av)
		if !bytes.Equal(ej, aj) {
			t.Errorf(red+"Campo %s: esperado %s, obtenido %s"+reset, keyPath, string(ej), string(aj))
		}
	}
}

// validateResponseArrayRules valida array_min_len, array_contains y array_not_contains
// sobre el JSON de respuesta.
func validateResponseArrayRules(t *testing.T, stepLabel string, vr ValidateResponse, root map[string]interface{}) {
	t.Helper()
	for path, minLen := range vr.ArrayMinLen {
		arr, ok := getSliceAtPath(root, path)
		if !ok {
			t.Errorf(red+"array_min_len%s: path %q no existe o no es un array"+reset, stepLabel, path)
			continue
		}
		if len(arr) < minLen {
			t.Errorf(red+"array_min_len%s: path %q longitud %d < mínimo %d; %s"+reset,
				stepLabel, path, len(arr), minLen, summarizeJSONArray(arr))
		}
	}
	for path, matchers := range vr.ArrayContains {
		arr, ok := getSliceAtPath(root, path)
		if !ok {
			t.Errorf(red+"array_contains%s: path %q no existe o no es un array"+reset, stepLabel, path)
			continue
		}
		for i, matcher := range matchers {
			if !arrayElementMatchesAny(arr, matcher) {
				t.Errorf(red+"array_contains%s: path %q matcher [%d] no encontrado en el array. %s matcher=%v"+reset,
					stepLabel, path, i, summarizeJSONArray(arr), matcher)
			}
		}
	}
	for path, matchers := range vr.ArrayNotContains {
		arr, ok := getSliceAtPath(root, path)
		if !ok {
			t.Errorf(red+"array_not_contains%s: path %q no existe o no es un array"+reset, stepLabel, path)
			continue
		}
		for i, matcher := range matchers {
			if idx := firstArrayIndexMatching(arr, matcher); idx >= 0 {
				t.Errorf(red+"array_not_contains%s: path %q matcher [%d] no debía aparecer pero coincide con el índice %d (matcher=%v). %s"+reset,
					stepLabel, path, i, idx, matcher, summarizeJSONArray(arr))
			}
		}
	}
}

func summarizeJSONArray(arr []interface{}) string {
	if len(arr) == 0 {
		return "array vacío (revisa payload.error/message: respuesta dummy p.ej. si IsLastMonthPaid falla, o banner custom que reemplaza eventos)."
	}
	parts := make([]string, 0, len(arr))
	for _, el := range arr {
		m, ok := toStringKeyedMap(el)
		if !ok {
			parts = append(parts, fmt.Sprintf(".(%T)", el))
			continue
		}
		parts = append(parts, fmt.Sprintf("%v", m["event_type"]))
	}
	return fmt.Sprintf("event_types en orden=[%s]", strings.Join(parts, ", "))
}

func toStringKeyedMap(v interface{}) (map[string]interface{}, bool) {
	switch m := v.(type) {
	case map[string]interface{}:
		return m, true
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(m))
		for k, val := range m {
			ks, ok := k.(string)
			if !ok {
				return nil, false
			}
			out[ks] = val
		}
		return out, true
	default:
		return nil, false
	}
}

func getSliceAtPath(root map[string]interface{}, path string) ([]interface{}, bool) {
	v, ok := getValueAtPath(root, path)
	if !ok {
		return nil, false
	}
	arr, ok := v.([]interface{})
	return arr, ok
}

func getValueAtPath(root map[string]interface{}, path string) (interface{}, bool) {
	cur := interface{}(root)
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func arrayElementMatchesAny(arr []interface{}, matcher interface{}) bool {
	for _, el := range arr {
		if elementMatchesMatcher(el, matcher) {
			return true
		}
	}
	return false
}

func firstArrayIndexMatching(arr []interface{}, matcher interface{}) int {
	for i, el := range arr {
		if elementMatchesMatcher(el, matcher) {
			return i
		}
	}
	return -1
}

func elementMatchesMatcher(el interface{}, matcher interface{}) bool {
	am, ok := toStringKeyedMap(el)
	if !ok {
		return false
	}
	mm, ok := toStringKeyedMap(matcher)
	if !ok {
		return false
	}
	return partialSubsetMatch(am, mm)
}

// partialSubsetMatch comprueba que actual cumple todas las claves de expected.
// Si expected es { "_match_any": [...] }, basta que actual cumpla alguna alternativa (OR).
func partialSubsetMatch(actual, expected map[string]interface{}) bool {
	if len(expected) == 1 {
		if raw, ok := expected[matchAnyKey]; ok {
			opts, ok := raw.([]interface{})
			if !ok {
				return false
			}
			for _, opt := range opts {
				om, ok := toStringKeyedMap(opt)
				if ok && partialSubsetMatch(actual, om) {
					return true
				}
			}
			return false
		}
	}
	for k, ev := range expected {
		av, ok := getFieldFromMap(actual, k)
		if !ok {
			return false
		}
		em, okE := toStringKeyedMap(ev)
		am, okA := toStringKeyedMap(av)
		if okE && okA {
			if !partialSubsetMatch(am, em) {
				return false
			}
			continue
		}
		ej, _ := json.Marshal(ev)
		aj, _ := json.Marshal(av)
		if !bytes.Equal(ej, aj) {
			return false
		}
	}
	return true
}

// getFieldFromMap obtiene un campo con alias opcionales (ej. event_type / eventType / EventType).
func getFieldFromMap(m map[string]interface{}, key string) (interface{}, bool) {
	if v, ok := m[key]; ok {
		return v, true
	}
	switch key {
	case "event_type":
		if v, ok := m["eventType"]; ok {
			return v, true
		}
		if v, ok := m["EventType"]; ok {
			return v, true
		}
	}
	return nil, false
}
