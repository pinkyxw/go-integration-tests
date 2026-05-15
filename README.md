# go-integration-tests

Motor de pruebas de integración data-driven para servicios Go. Los casos de prueba
se describen en archivos YAML; el framework realiza peticiones HTTP reales contra el
servicio bajo prueba y valida respuestas y estado de MongoDB.

## Instalación

```bash
go get github.com/pinkyxw/go-integration-tests@main
```

Usa siempre una revisión explícita (`@main`, `@latest` o un tag `v0.1.0`). Un `go get` sin sufijo intenta resolver `v0.0.0` como tag y falla.

**Opcional (solo desarrollo local):** con el repo clonado al lado del tuyo:

```
replace github.com/pinkyxw/go-integration-tests v0.0.0-… => ../go-integration-tests
```

(sustituye la pseudoversión por la que tengas en `go.mod`, o añade el `require` tras un `go get @main` y luego el `replace`.)
## Uso mínimo

```go
package integration_test

import (
    "testing"
    "github.com/pinkyxw/go-integration-tests/yamlintegration"
)

func TestIntegration(t *testing.T) {
    yamlintegration.Run(t, yamlintegration.Config{
        AppEnv:  "local",
        BaseURL: "http://localhost:8080",
    })
}
```

### Config completo

| Campo         | Tipo                | Descripción |
|---------------|---------------------|-------------|
| `AppEnv`      | `string`            | Entorno actual (ej. `"local"`, `"dev"`). |
| `AllowedEnvs` | `[]string`          | Entornos permitidos. Default `["dev", "local"]`. |
| `BaseURL`     | `string`            | URL base del servicio (ej. `"http://localhost:8080"`). |
| `Database`    | `*mongo.Database`   | Conexión a MongoDB ya establecida. Requerida si los YAML usan `preconditions` o `expected_db_state`. |
| `DefaultVars` | `map[string]string` | Variables de plantilla del proyecto. Se mezclan con las variables de fecha del framework; `DefaultVars` tiene precedencia. |
| `TestsDir`    | `string`            | Ruta a la carpeta YAML. Default: `INTEGRATION_TESTS_DIR` o `./integration_tests`. |

### Variables de entorno

| Variable                  | Descripción |
|---------------------------|-------------|
| `INTEGRATION_TESTS_DIR`   | Override de la carpeta YAML raíz. |
| `VERBOSE=true`            | Log de status code + body por cada paso HTTP. |
| `INTEGRATION_HTTP_TIMEOUT`| Timeout HTTP por petición (ej. `"90s"`). Default `30s`. |

### Variables de fecha automáticas

El framework inyecta estas variables en cada ejecución:

| Variable                    | Valor |
|-----------------------------|-------|
| `TODAY`                     | Fecha actual `YYYY-MM-DD` |
| `FIRST_DAY_CURRENT_MONTH`   | Primer día del mes actual |
| `FIRST_DAY_PREVIOUS_MONTH`  | Primer día del mes anterior |
| `LAST_DAY_PREVIOUS_MONTH`   | Último día del mes anterior |
| `TWO_DAYS_FROM_NOW`         | Hoy + 2 días |
| `FUTURE_DATE`               | `3000-01-01` |
| `PAST_DATE`                 | `2000-01-01` |

Las variables de `DefaultVars` pueden sobreescribir las anteriores si es necesario.
Las variables de entorno (`os.Getenv`) tienen la mayor precedencia.

## Esquema YAML

```yaml
name: "Nombre del caso"
description: "Descripción"

preconditions:
  - collection: users
    action: upsert          # delete | insert | upsert
    filter:
      _id: "507f1f77bcf86cd799439011"
    data:
      _id: "507f1f77bcf86cd799439011"
      name: "Test User"

# Formato multi-paso (encadenado)
steps:
  - execution:
      method: POST
      endpoint: /api/v1/resource
      headers:
        Authorization: "Bearer {{TOKEN}}"
      body:
        key: value
    validate_response:
      expected_status: 200
      expected_body:
        status: "ok"
      array_min_len:
        payload.items: 1
      array_contains:
        payload.items:
          - id: "123"
          - _match_any:
              - type: "A"
              - type: "B"
      array_not_contains:
        payload.items:
          - type: "forbidden"
    capture:
      RESOURCE_ID: payload.id

  - execution:
      method: GET
      endpoint: /api/v1/resource/{{RESOURCE_ID}}
    validate_response:
      expected_status: 200

expected_db_state:
  - collection: resources
    filter:
      _id: "507f1f77bcf86cd799439011"
    expected_data:
      status: "active"
```

Para el formato legacy de un solo paso, usa `execution` y `validate_response` directamente
(sin `steps:`). Ambos formatos son retrocompatibles.

## Variables en YAML

Cualquier valor de tipo string puede usar `{{NOMBRE}}`. La prioridad de resolución es:

1. Variable de entorno `NOMBRE` (os.Getenv)
2. `DefaultVars` del Config
3. Variables de fecha automáticas
4. El placeholder queda intacto si no se resuelve

## Ejecutar tests

```bash
# Todos los casos
go test -v -run TestIntegration .

# Con logging verbose
VERBOSE=true go test -v -run TestIntegration .

# Solo una subcarpeta
INTEGRATION_TESTS_DIR=./integration_tests/payments go test -v -run TestIntegration .
```
