# CloudBalancer - Roadmap de implementación por versiones

**Objetivo:** construir desde cero un HTTP Load Balancer en Go con calidad de proyecto de portfolio para un perfil junior orientado a Cloud / DevOps / SRE.

**Regla principal:** NO se empieza una versión nueva hasta que la versión actual cumpla todos sus criterios de aceptación, todos sus tests pasen y se haya creado el tag Git correspondiente.


QUIERO QUE SIGAS BUENAS PRACTICAS DE PROGRAMACION BUENOS NOBMRES DE VARIABLES FUNCIONES QUE SEAN ESPECIFICOS Y CLAROS.
DESPUES DE CADA CAMBIO SE SUGERIRA EL NOMBRE DEL COMMIT AL CAMBIO HECHO TIENE Q ESTAR EN UNA RAMA Y NO QUIERO QUE PONGAS COMENTARIOS EL CODIGO DEBER SER LEIBLE SIN COMENTARIOS 
Fecha del plan: 2026-08-07

---

# 0. Contrato de trabajo para Codex / agente de programación

Este documento debe tratarse como una especificación de implementación.

## Reglas obligatorias

1. Implementar las versiones estrictamente en orden.
2. No anticipar funcionalidades de versiones posteriores salvo que sean necesarias para evitar una mala arquitectura.
3. Al terminar cada versión:
   - ejecutar todos los tests;
   - ejecutar `go test -race ./...`;
   - ejecutar `go vet ./...`;
   - ejecutar `gofmt` sobre los archivos Go modificados;
   - comprobar manualmente el criterio funcional de esa versión;
   - actualizar el README si la funcionalidad visible ha cambiado;
   - hacer commit;
   - crear tag Git.
4. Si un test falla, NO continuar.
5. Si `go test -race` detecta una race condition, NO continuar.
6. No silenciar errores para conseguir que los tests pasen.
7. No eliminar tests salvo que el requisito haya cambiado explícitamente.
8. Cada bug descubierto debe reproducirse primero con un test siempre que sea razonable.
9. Preferir la biblioteca estándar de Go. Añadir dependencias externas solo cuando aporten valor claro.
10. Mantener las responsabilidades separadas: selección de backend, proxy HTTP, health checking, configuración y observabilidad no deben mezclarse en un único archivo.
11. Toda API pública interna importante debe tener tests.
12. Cada versión debe poder ejecutarse de forma funcional antes de avanzar.

## Quality Gate universal

Una versión solo se considera `DONE` cuando TODOS estos comandos terminan correctamente:

```bash
gofmt -w ./cmd ./internal ./test 2>/dev/null || true
go vet ./...
go test ./...
go test -race ./...
```

Cuando existan tests de integración:

```bash
go test -tags=integration ./test/integration/...
```

Cuando exista Docker:

```bash
docker compose config
docker compose build
```

Cuando exista Makefile, estos comandos se encapsularán en:

```bash
make fmt
make vet
make test
make test-race
make verify
```

---

# 1. Stack técnico decidido

## Lenguaje

**Go**.

Motivos:

- excelente biblioteca estándar HTTP;
- `net/http` y `httputil.ReverseProxy` permiten trabajar cerca del comportamiento real del proxy;
- goroutines y sincronización permiten demostrar concurrencia;
- binarios pequeños y fáciles de desplegar;
- encaja bien con infraestructura cloud-native;
- añade variedad técnica frente a proyectos Java/Spring.

## Dependencias previstas

No añadir todas desde el principio.

Se introducirán cuando corresponda:

- Standard Library:
  - `net/http`
  - `net/http/httputil`
  - `net/url`
  - `context`
  - `sync`
  - `sync/atomic` si aporta claridad
  - `log/slog`
  - `os/signal`
  - `time`
- YAML:
  - `gopkg.in/yaml.v3`
- Prometheus:
  - `github.com/prometheus/client_golang`

## Herramientas

- Git
- GitHub
- Docker
- Docker Compose
- Prometheus
- Grafana
- GitHub Actions
- Azure para el despliegue cloud final
- Kubernetes y Helm como ampliación posterior

---

# 2. Arquitectura objetivo

La arquitectura final aproximada será:

```text
                        +---------------------+
                        |       Client        |
                        +----------+----------+
                                   |
                                   v
                        +---------------------+
                        |    CloudBalancer    |
                        |---------------------|
                        | Reverse Proxy       |
                        | Load Balancing      |
                        | Health Checks       |
                        | Retries/Timeouts    |
                        | Metrics             |
                        | Structured Logs     |
                        +----+----------+-----+
                             |          |
                  +----------+          +----------+
                  v                                v
            +-----------+                    +-----------+
            | Backend A |       ...          | Backend N |
            +-----------+                    +-----------+

                        +---------------------+
                        |     Prometheus      |
                        +----------+----------+
                                   |
                                   v
                        +---------------------+
                        |       Grafana       |
                        +---------------------+
```

## Estructura objetivo del repositorio

No es obligatorio crear todos los directorios el primer día. Se irán creando cuando la versión los necesite.

```text
cloudbalancer/
|-- cmd/
|   |-- cloudbalancer/
|   |   `-- main.go
|   `-- demo-backend/
|       `-- main.go
|
|-- internal/
|   |-- backend/
|   |   |-- backend.go
|   |   `-- backend_test.go
|   |-- balancer/
|   |   |-- balancer.go
|   |   |-- round_robin.go
|   |   |-- weighted_round_robin.go
|   |   |-- least_connections.go
|   |   `-- *_test.go
|   |-- proxy/
|   |   |-- proxy.go
|   |   `-- proxy_test.go
|   |-- health/
|   |   |-- checker.go
|   |   `-- checker_test.go
|   |-- config/
|   |   |-- config.go
|   |   `-- config_test.go
|   |-- metrics/
|   |   |-- metrics.go
|   |   `-- metrics_test.go
|   `-- logging/
|       `-- logging.go
|
|-- test/
|   `-- integration/
|       |-- proxy_test.go
|       |-- failover_test.go
|       `-- distribution_test.go
|
|-- deploy/
|   |-- docker-compose.yml
|   |-- prometheus/
|   |   `-- prometheus.yml
|   |-- grafana/
|   |   `-- provisioning/
|   `-- kubernetes/
|
|-- configs/
|   |-- config.example.yaml
|   `-- config.local.yaml
|
|-- scripts/
|   |-- smoke-test.sh
|   `-- load-test.sh
|
|-- .github/
|   `-- workflows/
|       `-- ci.yml
|
|-- Dockerfile
|-- Makefile
|-- go.mod
|-- go.sum
|-- README.md
|-- LICENSE
`-- .gitignore
```

---

# 3. Convenciones desde el inicio

## Naming

Nombre del proyecto:

```text
CloudBalancer
```

Nombre del binario:

```text
cloudbalancer
```

## Puertos para desarrollo

```text
Load Balancer: 8080
Backend 1:      8081
Backend 2:      8082
Backend 3:      8083
Prometheus:     9090
Grafana:        3000
```

## Endpoints reservados del balanceador

No implementar todos inicialmente.

```text
/healthz     -> health del propio load balancer
/readyz      -> readiness del load balancer
/metrics     -> métricas Prometheus
```

El tráfico normal deberá proxificarse al backend.

## Formato de commits sugerido

```text
feat: add single backend reverse proxy
feat: implement round robin strategy
test: add round robin concurrency tests
fix: avoid selecting unhealthy backend
chore: add docker compose demo
```

## Tags

```text
v0.0.0
v0.1.0
v0.2.0
...
v1.0.0
```

---

# VERSION 0.0.0 - Repositorio y disciplina de calidad

## Objetivo

Crear un proyecto Go mínimo, reproducible y con un pipeline local de calidad antes de implementar networking.

## Tareas

### 0.0.1 Crear repositorio

Crear:

```text
cloudbalancer/
```

Inicializar:

```bash
git init
go mod init github.com/<usuario>/cloudbalancer
```

No inventar el usuario de GitHub si todavía no se conoce. Sustituirlo cuando se cree el repositorio remoto.

### 0.0.2 Crear `.gitignore`

Debe ignorar al menos:

```text
bin/
coverage.out
coverage.html
.env
configs/config.local.yaml
.DS_Store
.idea/
.vscode/
```

No ignorar configuraciones de ejemplo.

### 0.0.3 Crear `cmd/cloudbalancer/main.go`

Inicialmente solo debe arrancar y escribir un mensaje corto mediante `log/slog` o terminar limpiamente.

No implementar todavía el proxy.

### 0.0.4 Crear Makefile

Targets mínimos:

```make
fmt
vet
test
test-race
coverage
build
verify
run
```

`verify` debe ejecutar como mínimo:

```text
fmt -> vet -> test -> test-race -> build
```

### 0.0.5 Crear README inicial

Debe incluir:

- nombre del proyecto;
- objetivo en 2-3 líneas;
- stack inicial;
- estado: `Work in progress`;
- cómo compilar;
- cómo ejecutar tests.

### 0.0.6 Build local

El comando:

```bash
go build -o bin/cloudbalancer ./cmd/cloudbalancer
```

debe generar el ejecutable.

## Tests obligatorios

Todavía no se requiere lógica de dominio, pero debe existir al menos un test trivial que garantice que la infraestructura de tests funciona o una pequeña función testeable extraída de `main`.

## Criterio de aceptación

```bash
make verify
```

debe finalizar con exit code 0.

## GATE 0.0.0

NO continuar hasta que:

- [ ] `go build ./...` funciona.
- [ ] `go test ./...` funciona.
- [ ] `go test -race ./...` funciona.
- [ ] `go vet ./...` funciona.
- [ ] Makefile funciona.
- [ ] README existe.
- [ ] Git working tree está limpio.
- [ ] Se crea tag `v0.0.0`.

---

# VERSION 0.1.0 - Reverse proxy hacia UN único backend

## Objetivo

Aprender y demostrar el comportamiento básico de un reverse proxy HTTP sin introducir todavía balanceo.

Arquitectura:

```text
Client -> CloudBalancer :8080 -> Backend :8081
```

## Diseño

Crear paquete:

```text
internal/proxy
```

La lógica del proxy no debe estar dentro de `main.go`.

## Tareas

### 0.1.1 Crear backend de prueba

Crear:

```text
cmd/demo-backend/main.go
```

El backend debe:

- escuchar en un puerto configurable mediante variable de entorno;
- responder JSON;
- incluir un identificador de instancia configurable.

Respuesta aproximada:

```json
{
  "instance": "backend-1",
  "path": "/hello"
}
```

No es necesario que este sea el backend definitivo de producción. Es una herramienta de demo/integración.

### 0.1.2 Crear tipo Proxy

En:

```text
internal/proxy/proxy.go
```

Responsabilidad:

- aceptar una URL destino;
- construir un `httputil.ReverseProxy`;
- implementar `http.Handler` o exponer `ServeHTTP`.

No implementar algoritmos de selección todavía.

### 0.1.3 Manejo de URL

Validar:

- URL vacía;
- esquema no soportado;
- URL inválida.

La creación del proxy debe devolver error si la configuración no es válida.

### 0.1.4 Arranque desde main

`main.go` deberá:

1. leer backend desde una variable de entorno temporal, por ejemplo `BACKEND_URL`;
2. crear proxy;
3. levantar servidor HTTP en `:8080`;
4. registrar inicio con `slog`.

Ejemplo:

```bash
BACKEND_URL=http://localhost:8081 go run ./cmd/cloudbalancer
```

### 0.1.5 Manejo de errores del proxy

Configurar `ErrorHandler` del reverse proxy.

Cuando el backend no esté accesible:

- devolver HTTP `502 Bad Gateway`;
- no provocar panic;
- registrar el error.

## Tests unitarios obligatorios

Usar `httptest.Server`.

Crear al menos:

### Test 1

```text
TestProxy_ForwardsRequestToBackend
```

Comprobar:

- status recibido del backend;
- body recibido;
- path preservado.

### Test 2

```text
TestProxy_ForwardsQueryString
```

Petición:

```text
/search?q=cloud
```

El backend debe recibir la misma query.

### Test 3

```text
TestProxy_PreservesMethod
```

Probar al menos POST.

### Test 4

```text
TestProxy_ReturnsBadGatewayWhenBackendUnavailable
```

Backend cerrado -> proxy devuelve 502.

### Test 5

```text
TestNewProxy_RejectsInvalidURL
```

## Test manual obligatorio

Terminal 1:

```bash
PORT=8081 INSTANCE_ID=backend-1 go run ./cmd/demo-backend
```

Terminal 2:

```bash
BACKEND_URL=http://localhost:8081 go run ./cmd/cloudbalancer
```

Terminal 3:

```bash
curl -i http://localhost:8080/hello
```

Debe verse que la respuesta procede de `backend-1`.

## GATE 0.1.0

- [ ] Todas las peticiones pasan a un backend.
- [ ] Path y query se preservan.
- [ ] POST funciona.
- [ ] Backend caído devuelve 502.
- [ ] No hay panic.
- [ ] Tests unitarios creados.
- [ ] `make verify` pasa.
- [ ] README actualizado con demo de un backend.
- [ ] Tag `v0.1.0` creado.

---

# VERSION 0.2.0 - Modelo de Backend + Round Robin

## Objetivo

Convertir el proxy de destino fijo en un load balancer con múltiples backends y algoritmo Round Robin.

Arquitectura:

```text
                     +-> backend-1
Client -> LB --------+-> backend-2
                     +-> backend-3
```

## Regla de diseño

El proxy NO decide qué backend usar.

El algoritmo de balanceo NO realiza HTTP proxying.

Separar ambas responsabilidades.

## Tareas

### 0.2.1 Crear modelo Backend

Archivo:

```text
internal/backend/backend.go
```

Debe representar como mínimo:

```text
ID
URL
Alive
```

`Alive` todavía puede ser simple, pero debe diseñarse pensando en concurrencia.

El estado de salud se usará realmente en la versión siguiente.

### 0.2.2 Crear interfaz de estrategia

Archivo:

```text
internal/balancer/balancer.go
```

Crear una interfaz pequeña. Ejemplo conceptual:

```go
type Balancer interface {
    Next() (*backend.Backend, error)
}
```

El nombre exacto puede cambiar si hay una razón arquitectónica clara.

### 0.2.3 Implementar RoundRobin

Archivo:

```text
internal/balancer/round_robin.go
```

Requisitos:

- recibe lista de backends;
- rota de forma circular;
- thread-safe;
- no utiliza random;
- no depende de HTTP.

Para tres backends, las selecciones deben seguir:

```text
A B C A B C A B C...
```

### 0.2.4 Integrar selector en Proxy

El `Proxy` debe pedir un backend al `Balancer` para cada petición.

Puede usarse un `ReverseProxy` dinámico o un Director/Rewrite adecuado.

No duplicar innecesariamente el reverse proxy por request.

### 0.2.5 Configuración temporal

Hasta la fase YAML, se puede usar una variable como:

```text
BACKEND_URLS=http://localhost:8081,http://localhost:8082,http://localhost:8083
```

## Tests unitarios obligatorios

### Backend

```text
TestBackend_StoresParsedURL
```

### Round Robin

```text
TestRoundRobin_ReturnsBackendsInOrder
TestRoundRobin_WrapsAround
TestRoundRobin_WithSingleBackend
TestRoundRobin_ReturnsErrorWithNoBackends
```

### Concurrencia

```text
TestRoundRobin_ConcurrentAccess
```

Ejecutar muchas llamadas concurrentes a `Next()` y verificar:

- no panic;
- no race detector;
- todas las selecciones pertenecen al pool.

No exigir distribución exacta bajo concurrencia salvo que el diseño lo garantice de manera determinista.

### Integración proxy + RR

Levantar 3 `httptest.Server` con IDs distintos.

Realizar 6 requests y esperar:

```text
A B C A B C
```

Nombre sugerido:

```text
TestProxy_RoundRobinDistribution
```

## Test manual obligatorio

Levantar 3 demo backends.

Ejecutar:

```bash
for i in $(seq 1 9); do curl -s http://localhost:8080/; echo; done
```

Esperar patrón repetido 1 -> 2 -> 3.

## GATE 0.2.0

- [ ] Existen al menos 3 backends configurables.
- [ ] Round Robin funciona deterministicamente en secuencial.
- [ ] Round Robin es thread-safe.
- [ ] `go test -race ./...` pasa.
- [ ] Integración demuestra distribución.
- [ ] Proxy y algoritmo están desacoplados.
- [ ] README actualizado.
- [ ] Tag `v0.2.0` creado.

---

# VERSION 0.3.0 - Active Health Checks y failover

## Objetivo

Evitar enviar tráfico a backends caídos.

Esta es una de las fases más importantes para el valor del proyecto en CV.

## Comportamiento esperado

Estado inicial:

```text
backend-1 healthy
backend-2 healthy
backend-3 healthy
```

Si backend-2 deja de responder:

```text
backend-2 unhealthy
```

El Round Robin debe seleccionar únicamente:

```text
backend-1
backend-3
```

Cuando backend-2 se recupere, debe volver al pool automáticamente.

## Tareas

### 0.3.1 Extender Backend

El Backend debe permitir:

- leer estado healthy;
- cambiar estado healthy;
- hacerlo de manera segura bajo concurrencia.

No exponer mutex directamente.

Funciones aproximadas:

```text
IsAlive()
SetAlive(bool)
```

### 0.3.2 Crear HealthChecker

Archivo:

```text
internal/health/checker.go
```

Responsabilidades:

- comprobar periódicamente cada backend;
- usar un endpoint configurable, inicialmente `/health`;
- aplicar timeout;
- marcar healthy/unhealthy;
- ejecutarse hasta cancelar `context.Context`.

No debe contener lógica de Round Robin.

### 0.3.3 Definir health check

Un backend se considera healthy si:

- hay respuesta dentro del timeout;
- status está en rango 200-299.

Cualquier otra situación:

- error de red;
- timeout;
- HTTP 500;

=> unhealthy.

### 0.3.4 Intervalos de desarrollo

Valores iniciales razonables:

```text
interval: 2s
timeout:  500ms
```

Más adelante serán configurables.

### 0.3.5 Modificar Round Robin

`Next()` debe saltar backends unhealthy.

Caso extremo:

```text
0 backends healthy
```

Debe devolver error explícito, no loop infinito.

### 0.3.6 Respuesta sin backends

Si no existe ningún backend sano:

```text
HTTP 503 Service Unavailable
```

No usar 500.

### 0.3.7 Recuperación automática

Un backend marcado unhealthy debe seguir siendo comprobado.

Si vuelve a responder 2xx, pasa a healthy y puede recibir tráfico de nuevo.

## Tests obligatorios

### Health checker unitario

```text
TestHealthChecker_MarksHealthyBackendAlive
TestHealthChecker_Marks500BackendUnhealthy
TestHealthChecker_MarksUnreachableBackendUnhealthy
TestHealthChecker_MarksTimeoutBackendUnhealthy
TestHealthChecker_RecoversBackend
```

Para timeout usar un `httptest.Server` cuya respuesta tarde más que el timeout configurado.

### Round Robin con health

```text
TestRoundRobin_SkipsUnhealthyBackend
TestRoundRobin_ReturnsErrorWhenAllBackendsUnhealthy
TestRoundRobin_ReincludesRecoveredBackend
```

### Integración de failover

Escenario:

1. levantar A, B y C;
2. verificar que los tres reciben requests;
3. cerrar B;
4. esperar health check;
5. realizar requests;
6. confirmar que ninguna llega a B;
7. levantar B de nuevo o sustituir por servidor recuperado;
8. confirmar reincorporación.

Nombre:

```text
TestFailover_BackendFailureAndRecovery
```

## Test manual obligatorio

Con Docker todavía no es obligatorio.

Usar tres terminales o procesos.

1. Hacer 9 requests.
2. Parar backend-2.
3. Esperar más de un intervalo de health check.
4. Hacer 10 requests.
5. Comprobar que solo responden backend-1 y backend-3.
6. Volver a arrancar backend-2.
7. Esperar health check.
8. Comprobar que vuelve a recibir tráfico.

## GATE 0.3.0

- [ ] Backend caído deja de recibir tráfico.
- [ ] Backend recuperado vuelve automáticamente.
- [ ] Si todos caen se devuelve 503.
- [ ] Health checker es cancelable mediante context.
- [ ] No existen data races.
- [ ] Tests de timeout existen.
- [ ] Test de recuperación existe.
- [ ] `make verify` pasa.
- [ ] Tag `v0.3.0` creado.

---

# VERSION 0.4.0 - Timeouts, errores y resiliencia de request

## Objetivo

Hacer que una request no pueda quedar bloqueada indefinidamente por un backend lento y mejorar el comportamiento ante fallos transitorios.

## Importante

No implementar retries agresivos sin pensar en idempotencia.

## Tareas

### 0.4.1 Configurar transporte HTTP

Crear un `http.Transport` con límites razonables para:

- dial timeout;
- TLS handshake timeout;
- response header timeout;
- idle connections;
- idle connection timeout.

No crear un nuevo transport por request.

### 0.4.2 Request timeout

Definir timeout máximo del proxy para esperar una respuesta del backend.

Cuando expire:

- cancelar request aguas abajo;
- responder error apropiado, normalmente 504 Gateway Timeout;
- registrar backend implicado.

### 0.4.3 Retry limitado

Implementar retry inicialmente SOLO para métodos idempotentes:

```text
GET
HEAD
OPTIONS
```

Opcionalmente PUT/DELETE solo si se documenta la decisión. Para simplificar y evitar problemas, empezar con GET/HEAD/OPTIONS.

### 0.4.4 Número máximo de intentos

Valor inicial:

```text
maxAttempts = 2
```

Esto significa request inicial + como máximo un segundo backend.

No crear loops infinitos.

### 0.4.5 No reintentar POST automáticamente

Un `POST` fallido NO debe enviarse a otro backend por defecto.

Crear test explícito.

### 0.4.6 Evitar retry al mismo backend

Si falla backend A y existe B sano, el segundo intento debe intentar B.

Debe existir mecanismo para excluir los backends ya probados en esa request o estrategia equivalente.

## Tests obligatorios

```text
TestProxy_ReturnsGatewayTimeoutForSlowBackend
TestProxy_RetriesGETOnDifferentBackend
TestProxy_DoesNotRetryPOST
TestProxy_StopsAfterMaxAttempts
TestProxy_Returns503WhenNoBackendAvailable
```

Caso clave:

- A falla;
- B funciona;
- GET termina en 200 desde B.

## GATE 0.4.0

- [ ] Slow backend no bloquea indefinidamente.
- [ ] GET puede hacer retry.
- [ ] POST no se duplica.
- [ ] Hay máximo de intentos.
- [ ] Retry cambia de backend.
- [ ] Tests de error y timeout pasan.
- [ ] `go test -race ./...` pasa.
- [ ] README documenta política de retries.
- [ ] Tag `v0.4.0` creado.

---

# VERSION 0.5.0 - Weighted Round Robin

## Objetivo

Permitir que backends con distinta capacidad reciban proporciones diferentes de tráfico.

Ejemplo:

```text
backend-1 weight=5
backend-2 weight=3
backend-3 weight=2
```

En una ventana suficientemente grande, distribución aproximada:

```text
50% / 30% / 20%
```

## Tareas

### 0.5.1 Extender Backend

Añadir `Weight`.

Validar:

```text
weight >= 1
```

### 0.5.2 Implementar estrategia independiente

Archivo:

```text
internal/balancer/weighted_round_robin.go
```

No modificar RoundRobin hasta convertirlo en Weighted Round Robin.

Ambas estrategias deben coexistir.

### 0.5.3 Elegir algoritmo

Preferencia: Smooth Weighted Round Robin.

Documentar brevemente en código/README:

- peso estático;
- current weight;
- selección del mayor current weight;
- resta del total de pesos al seleccionado.

### 0.5.4 Respetar health

Backend unhealthy no participa temporalmente en la selección.

Al recuperarse vuelve con su peso configurado.

## Tests obligatorios

```text
TestWeightedRoundRobin_EqualWeightsBehavesEvenly
TestWeightedRoundRobin_RespectsWeights
TestWeightedRoundRobin_SkipsUnhealthyBackend
TestWeightedRoundRobin_AllUnhealthyReturnsError
TestWeightedRoundRobin_ConcurrentAccess
```

Para pesos 5/3/2, realizar exactamente 10 o un múltiplo adecuado si el algoritmo garantiza esa distribución en la ventana.

Si el algoritmo no garantiza una ventana exacta, probar tolerancias estadísticas razonables sobre muchas selecciones.

## GATE 0.5.0

- [ ] Round Robin sigue funcionando.
- [ ] Weighted Round Robin funciona.
- [ ] Weighted strategy es thread-safe.
- [ ] Health se respeta.
- [ ] Tests deterministas o estadísticos robustos.
- [ ] `make verify` pasa.
- [ ] Tag `v0.5.0` creado.

---

# VERSION 0.6.0 - Least Connections / Least Active Requests

## Objetivo

Seleccionar el backend con menor número de requests activas.

Para HTTP a nivel aplicación es más preciso llamar al contador `ActiveRequests`, aunque el algoritmo pueda presentarse como Least Connections en el README explicando la aproximación.

## Tareas

### 0.6.1 Extender Backend

Añadir contador concurrente:

```text
activeRequests
```

Operaciones:

```text
IncrementActive()
DecrementActive()
ActiveCount()
```

Debe ser thread-safe.

### 0.6.2 Lifecycle correcto

Antes de enviar request al backend seleccionado:

```text
IncrementActive
```

Al finalizar, SIEMPRE:

```text
DecrementActive
```

Usar `defer` solo si garantiza el punto de decremento correcto respecto al final real de la request.

### 0.6.3 Implementar estrategia

Archivo:

```text
internal/balancer/least_connections.go
```

Seleccionar backend healthy con menor contador.

### 0.6.4 Empates

No elegir siempre el primer backend en caso de empate porque puede sesgar el tráfico.

Solución sugerida:

- mantener cursor Round Robin para desempate.

Documentarlo.

## Tests obligatorios

```text
TestLeastConnections_SelectsBackendWithLowestActiveRequests
TestLeastConnections_SkipsUnhealthyBackend
TestLeastConnections_TieBreaksFairly
TestBackend_ActiveRequestsNeverNegative
TestLeastConnections_ConcurrentAccess
```

### Test de integración con backend lento

- backend A duerme 500 ms;
- backend B responde inmediatamente;
- lanzar requests concurrentes;
- comprobar que nuevas requests tienden a B mientras A está ocupado.

No exigir una distribución exacta si el scheduler introduce variación.

## GATE 0.6.0

- [ ] Contador activo correcto.
- [ ] No queda contador positivo tras finalizar todas las requests.
- [ ] No puede hacerse negativo.
- [ ] Estrategia selecciona menor carga.
- [ ] Desempate no introduce sesgo permanente.
- [ ] Race detector limpio.
- [ ] Tag `v0.6.0` creado.

---

# VERSION 0.7.0 - Configuración YAML validada

## Objetivo

Eliminar configuración ad hoc por variables separadas y definir una configuración reproducible para local, Docker y cloud.

## Archivo objetivo

```text
configs/config.example.yaml
```

Ejemplo conceptual:

```yaml
server:
  listen_address: ":8080"
  request_timeout: 5s

balancer:
  strategy: round_robin

health_check:
  enabled: true
  interval: 2s
  timeout: 500ms
  path: /health

retries:
  max_attempts: 2

backends:
  - id: backend-1
    url: http://localhost:8081
    weight: 5
  - id: backend-2
    url: http://localhost:8082
    weight: 3
  - id: backend-3
    url: http://localhost:8083
    weight: 2
```

## Tareas

### 0.7.1 Crear paquete config

```text
internal/config/config.go
```

Funciones aproximadas:

```text
Load(path string) (Config, error)
Validate(Config) error
```

### 0.7.2 Validaciones

Rechazar:

- cero backends;
- IDs duplicados;
- URL inválida;
- weight <= 0;
- estrategia desconocida;
- health interval <= 0;
- health timeout <= 0;
- max attempts < 1.

### 0.7.3 Selección de estrategia

Valores permitidos:

```text
round_robin
weighted_round_robin
least_connections
```

Crear factory separada si mejora la arquitectura.

### 0.7.4 CLI mínima

Soportar:

```bash
cloudbalancer --config ./configs/config.local.yaml
```

No crear un framework CLI pesado salvo necesidad.

Puede usarse `flag` de la standard library.

### 0.7.5 Defaults

Definir defaults explícitos para valores no críticos.

No aplicar defaults silenciosos a errores graves como backend URL vacío.

## Tests obligatorios

```text
TestConfig_LoadValidYAML
TestConfig_RejectsUnknownStrategy
TestConfig_RejectsDuplicateBackendID
TestConfig_RejectsInvalidBackendURL
TestConfig_RejectsInvalidWeight
TestConfig_RejectsNoBackends
TestConfig_AppliesDefaults
```

Añadir fixtures YAML dentro del paquete de test o usar strings temporales.

## GATE 0.7.0

- [ ] App arranca exclusivamente con archivo YAML + defaults documentados.
- [ ] Config inválida falla rápido al arrancar.
- [ ] No inicia parcialmente con configuración corrupta.
- [ ] Las tres estrategias se pueden elegir desde YAML.
- [ ] Tests de validación completos.
- [ ] `config.example.yaml` actualizado.
- [ ] Tag `v0.7.0` creado.

---

# VERSION 0.8.0 - Observabilidad: logs estructurados + Prometheus

## Objetivo

Hacer observable el sistema como servicio de infraestructura real.

## Parte A - Logging

Usar `log/slog`.

### Campos recomendados

Para request:

```text
method
path
backend_id
status
latency_ms
attempt
```

Para health check:

```text
backend_id
old_state
new_state
reason
```

Evitar imprimir un log de health check exitoso cada 2 segundos si genera ruido.

Registrar especialmente transiciones:

```text
healthy -> unhealthy
unhealthy -> healthy
```

## Parte B - Prometheus

### Endpoint

```text
GET /metrics
```

### Métricas mínimas

Counter:

```text
cloudbalancer_requests_total
```

Labels sugeridas:

```text
method
backend
status
```

Counter:

```text
cloudbalancer_backend_errors_total
```

Gauge:

```text
cloudbalancer_backend_healthy
```

Valor:

```text
1 healthy
0 unhealthy
```

Gauge:

```text
cloudbalancer_backend_active_requests
```

Histogram:

```text
cloudbalancer_backend_request_duration_seconds
```

Counter:

```text
cloudbalancer_retries_total
```

### Cardinalidad

NO usar path completo como label Prometheus si puede contener IDs u otros valores de cardinalidad no acotada.

Documentar esta decisión.

## Parte C - Health del propio load balancer

Implementar:

```text
GET /healthz
```

Debe indicar que el proceso está vivo.

Implementar:

```text
GET /readyz
```

Debe devolver ready solo si existe al menos un backend healthy, salvo que se documente otra política.

## Tests obligatorios

```text
TestMetrics_RequestCounterIncrements
TestMetrics_BackendErrorCounterIncrements
TestMetrics_HealthyGaugeChanges
TestMetrics_RetryCounterIncrements
TestHealthz_Returns200
TestReadyz_Returns200WhenBackendAvailable
TestReadyz_Returns503WhenNoBackendAvailable
```

No comprobar texto completo de `/metrics`; comprobar métricas concretas mediante librerías de test Prometheus cuando sea posible.

## GATE 0.8.0

- [ ] Logs son estructurados.
- [ ] Se identifica backend seleccionado.
- [ ] `/metrics` funciona.
- [ ] `/healthz` funciona.
- [ ] `/readyz` refleja disponibilidad.
- [ ] Métricas no introducen data races.
- [ ] Cardinalidad de labels está controlada.
- [ ] Tag `v0.8.0` creado.

---

# VERSION 0.9.0 - Lifecycle de producción y graceful shutdown

## Objetivo

Cerrar el load balancer sin cortar requests activas de forma abrupta.

## Tareas

### 0.9.1 Señales

Escuchar:

```text
SIGINT
SIGTERM
```

Usar `signal.NotifyContext` si encaja.

### 0.9.2 Shutdown ordenado

Al recibir señal:

1. dejar de aceptar nuevas conexiones;
2. cancelar health checker;
3. permitir finalizar requests en curso;
4. aplicar timeout máximo de shutdown;
5. cerrar transport idle connections;
6. terminar proceso.

### 0.9.3 Logging de lifecycle

Registrar:

```text
server started
shutdown initiated
shutdown completed
```

No es necesario loggear ruido innecesario.

## Tests obligatorios

La lógica de shutdown debe estar suficientemente desacoplada de `main` como para poder probarla.

Crear al menos:

```text
TestServer_GracefulShutdownLetsActiveRequestFinish
TestHealthChecker_StopsWhenContextCancelled
```

Si un test con señales del sistema resulta frágil, probar el método de shutdown directamente y mantener un smoke test manual de señales.

## Test manual

1. iniciar backend lento;
2. iniciar una request que tarde 2 s;
3. enviar SIGTERM al load balancer durante la request;
4. la request debe completar si está dentro del timeout de shutdown;
5. el proceso debe terminar después.

## GATE 0.9.0

- [ ] SIGTERM no mata inmediatamente requests sanas activas.
- [ ] Health checker termina.
- [ ] No quedan goroutines obvias bloqueadas.
- [ ] Race detector limpio.
- [ ] Tag `v0.9.0` creado.

---

# VERSION 1.0.0 - Docker Compose demo reproducible

## Objetivo

Crear una demo que cualquier reclutador o entrevistador pueda ejecutar con un único comando.

```bash
docker compose up --build
```

## Servicios

```text
cloudbalancer
backend-1
backend-2
backend-3
prometheus
grafana
```

## Tareas

### 1.0.1 Dockerfile del load balancer

Usar multi-stage build.

Características:

- build stage con Go;
- runtime stage reducido;
- ejecutar como usuario no root si es práctico;
- copiar solo binario y configuración necesaria;
- `EXPOSE 8080`.

### 1.0.2 Demo backend container

Puede usarse la misma imagen/binario con variables:

```text
INSTANCE_ID
PORT
```

### 1.0.3 Docker Compose

Los backends deben resolverse por DNS interno:

```text
http://backend-1:8080
http://backend-2:8080
http://backend-3:8080
```

No usar `localhost` entre contenedores.

### 1.0.4 Prometheus

Crear:

```text
deploy/prometheus/prometheus.yml
```

Scrapear:

```text
cloudbalancer:8080/metrics
```

### 1.0.5 Grafana

Provisionar datasource Prometheus automáticamente.

Idealmente provisionar un dashboard mínimo con:

- requests por segundo;
- errores por backend;
- latencia;
- backends healthy;
- active requests.

### 1.0.6 Healthchecks Docker

Añadir healthcheck al menos para CloudBalancer usando `/healthz`.

### 1.0.7 Script smoke test

Crear:

```text
scripts/smoke-test.sh
```

Debe:

1. enviar varias requests;
2. verificar HTTP 200;
3. verificar que aparecen varios IDs de backend;
4. verificar `/metrics`;
5. verificar `/healthz`.

### 1.0.8 Demo de caída

Documentar exactamente:

```bash
docker compose stop backend-2
```

Esperar health check y volver a enviar tráfico.

Después:

```bash
docker compose start backend-2
```

Mostrar recuperación.

## Tests obligatorios

Además de todos los tests Go:

```bash
docker compose build
docker compose up -d
./scripts/smoke-test.sh
docker compose down
```

## GATE 1.0.0

- [ ] `docker compose up --build` levanta todo.
- [ ] 3 backends reciben tráfico.
- [ ] Stop de uno no tumba el servicio.
- [ ] Start del backend lo reincorpora.
- [ ] Prometheus scrapea el LB.
- [ ] Grafana tiene datasource configurado.
- [ ] Smoke test automatizado pasa.
- [ ] README tiene instrucciones exactas de demo.
- [ ] Tag `v1.0.0` creado.

**A partir de aquí el proyecto ya es presentable en CV.**

---

# VERSION 1.1.0 - Integration tests end-to-end y escenarios de fallo

## Objetivo

Demostrar que el sistema completo funciona, no solo sus unidades.

## Tareas

Crear suite en:

```text
test/integration
```

Usar procesos/httptest o Docker según el caso. Mantener los tests rápidos siempre que sea posible.

## Escenarios obligatorios

### E2E 1 - Round Robin

- 3 backends;
- 9 requests;
- reparto esperado 3/3/3 en ejecución secuencial.

### E2E 2 - Backend failure

- 3 healthy;
- cae B;
- health check lo detecta;
- requests continúan por A/C.

### E2E 3 - Recovery

- B vuelve;
- health checker lo detecta;
- B recibe tráfico.

### E2E 4 - All backends down

Esperar:

```text
503 Service Unavailable
```

### E2E 5 - Slow backend

- A muy lento;
- timeout configurado;
- respuesta no queda colgada.

### E2E 6 - Retry GET

- A falla;
- B healthy;
- GET termina correctamente desde B.

### E2E 7 - POST no duplicado

Instrumentar backend para contar requests.

Confirmar que el POST solo se intentó una vez.

### E2E 8 - Concurrent load

Enviar cientos o miles de requests concurrentes, en función del tiempo de test.

Validar:

- sin panic;
- respuestas dentro de expectativas;
- race detector limpio en suite compatible.

## GATE 1.1.0

- [ ] Todos los escenarios E2E pasan.
- [ ] Tests no son flaky tras varias ejecuciones.
- [ ] No dependen de sleeps arbitrarios largos cuando puede usarse polling con deadline.
- [ ] CI podrá ejecutarlos.
- [ ] Tag `v1.1.0` creado.

---

# VERSION 1.2.0 - Benchmarks y load testing

## Objetivo

Añadir evidencia cuantitativa y aprender a medir throughput/latencia.

No optimizar prematuramente. Primero medir.

## Parte A - Go Benchmarks

Crear benchmarks para estrategias:

```text
BenchmarkRoundRobinNext
BenchmarkWeightedRoundRobinNext
BenchmarkLeastConnectionsNext
```

Ejecutar:

```bash
go test -bench=. -benchmem ./internal/balancer/...
```

Guardar resultados representativos en documentación, indicando máquina/entorno.

## Parte B - Load testing HTTP

Elegir una herramienta común, por ejemplo:

- `hey`, o
- `wrk`, o
- `k6`.

Preferencia para portfolio Cloud: `k6` si se quiere un escenario versionable; `hey` si se busca simplicidad.

Crear script:

```text
scripts/load-test.sh
```

Escenarios:

### Scenario A

```text
1000 requests
concurrency 20
3 healthy backends
```

### Scenario B

Misma carga con un backend detenido.

Comparar:

- success rate;
- p50 latency;
- p95 latency;
- p99 latency;
- requests/sec.

## Parte C - Observación

Mientras corre la prueba:

- Prometheus recibe métricas;
- Grafana muestra distribución y latencia.

## GATE 1.2.0

- [ ] Benchmarks Go reproducibles.
- [ ] Load test script existe.
- [ ] Resultados documentados sin exageraciones.
- [ ] Se incluyen condiciones del benchmark.
- [ ] No afirmar que supera Nginx/Envoy sin una comparación experimental justa.
- [ ] Tag `v1.2.0` creado.

---

# VERSION 1.3.0 - CI con GitHub Actions

## Objetivo

Cada push/PR debe verificar automáticamente la calidad del proyecto.

## Workflow

Archivo:

```text
.github/workflows/ci.yml
```

## Jobs mínimos

### Job 1 - quality

```text
checkout
setup-go
gofmt check
go vet
go test
go test -race
go build
```

### Job 2 - integration

Ejecutar integration tests.

Puede depender de `quality`.

### Job 3 - docker

```text
docker build
```

Como mínimo confirmar que la imagen se construye.

## Coverage

Generar:

```bash
go test -coverprofile=coverage.out ./...
```

No fijar inicialmente un umbral artificial muy alto.

Objetivo orientativo:

```text
>= 70% en paquetes de lógica crítica
```

Pero priorizar calidad de tests sobre porcentaje global.

## Branch protection recomendada

En GitHub, cuando el repo esté remoto:

- requerir CI verde antes de merge a main;
- trabajar con ramas `feature/...` para fases grandes si se desea.

## Badge README

Añadir badge CI.

Coverage badge solo si se configura una solución fiable.

## GATE 1.3.0

- [ ] Push ejecuta CI.
- [ ] Un test roto rompe CI.
- [ ] Race detector se ejecuta.
- [ ] Docker image build se verifica.
- [ ] README muestra CI.
- [ ] Tag `v1.3.0` creado.

---

# VERSION 1.4.0 - Imagen de contenedor publicable y seguridad básica

## Objetivo

Preparar el artefacto para despliegue cloud.

## Tareas

### 1.4.1 Imagen mínima

Revisar:

- multi-stage;
- usuario no root;
- no incluir código fuente innecesario en runtime;
- no incluir secretos;
- `.dockerignore`.

### 1.4.2 Metadata

Usar build args/ldflags para incluir opcionalmente:

```text
version
commit SHA
build date
```

Crear endpoint o flag:

```bash
cloudbalancer --version
```

### 1.4.3 Scan

Añadir en CI un scanner de imagen conocido si resulta estable, por ejemplo Trivy.

No bloquear inicialmente por vulnerabilidades de severidad baja si crea ruido. Documentar política.

### 1.4.4 Registry

Publicar imagen en uno de:

- GitHub Container Registry, o
- Docker Hub.

Preferencia: GHCR para integrarlo con GitHub Actions.

Etiquetas:

```text
latest
v1.4.0
<git-sha>
```

## GATE 1.4.0

- [ ] Imagen corre como non-root.
- [ ] No contiene secrets.
- [ ] `--version` funciona.
- [ ] CI construye la imagen.
- [ ] Imagen versionada publicada.
- [ ] Tag `v1.4.0` creado.

---

# VERSION 1.5.0 - Despliegue en Azure

## Objetivo

Tener una URL pública y demostrar experiencia real de despliegue cloud.

## Enfoque recomendado

Desplegar inicialmente CloudBalancer y backends como contenedores en Azure.

El servicio concreto puede elegirse según coste y disponibilidad de la cuenta, pero debe permitir:

- ejecutar contenedores;
- networking entre LB y backends;
- logs;
- variables/secrets;
- health probes.

Para un portfolio estudiantil, Azure Container Apps suele ser una opción razonable si encaja con los límites de la cuenta.

## Tareas

### 1.5.1 Separar config local/cloud

Nunca hardcodear hostname local.

Config cloud debe poder apuntar a endpoints internos/reales.

### 1.5.2 Health probes

Usar:

```text
/healthz
/readyz
```

### 1.5.3 Secret management

Si no hay secretos reales, mantener el proyecto sin necesidad de ellos.

Si aparecen tokens/credentials:

- nunca commit;
- usar mecanismo de secrets del proveedor.

### 1.5.4 Logging

Confirmar que logs estructurados aparecen en la plataforma.

### 1.5.5 Deployment automatizado

Añadir workflow separado solo después de que el deploy manual sea estable.

Flujo recomendado:

```text
push tag -> test -> build image -> push image -> deploy
```

Nunca automatizar una secuencia de despliegue que todavía no se ha probado manualmente.

### 1.5.6 Smoke test post-deploy

Después de deploy:

```text
GET /healthz
GET /readyz
GET /metrics (si no se expone públicamente, verificar internamente)
GET /demo
```

### 1.5.7 Documentación

README debe indicar:

- arquitectura cloud;
- qué servicio Azure se usó;
- cómo se despliega;
- decisiones de networking;
- limitaciones de la demo.

## GATE 1.5.0

- [ ] URL pública responde.
- [ ] Tráfico llega a múltiples backends.
- [ ] Health probes funcionan.
- [ ] Logs accesibles.
- [ ] CI/CD solo despliega tras tests verdes.
- [ ] No hay credentials en Git.
- [ ] README contiene arquitectura cloud.
- [ ] Tag `v1.5.0` creado.

**Este es el objetivo recomendado para poner el proyecto con fuerza en el CV.**

---

# VERSION 1.6.0 - Kubernetes - ampliación Cloud/SRE

## Objetivo

Aprender la relación entre tu load balancer y las primitivas reales de Kubernetes.

Esta fase es una ampliación. No debe retrasar que publiques el proyecto terminado hasta v1.5.0.

## Tareas

### 1.6.1 Manifests

Crear:

```text
deploy/kubernetes/
```

Manifests mínimos:

```text
Deployment cloudbalancer
Deployment demo-backend
Service cloudbalancer
Service demo-backend
ConfigMap
```

### 1.6.2 Replicas

Levantar varias réplicas del demo backend.

Analizar una cuestión importante:

> Si CloudBalancer apunta a un Kubernetes Service, Kubernetes ya realiza balanceo por debajo. Para demostrar TU algoritmo, CloudBalancer debe conocer endpoints individuales o desplegar backends de manera que pueda distinguirlos.

Documentar esta diferencia en el README.

### 1.6.3 Probes

```text
livenessProbe -> /healthz
readinessProbe -> /readyz
```

### 1.6.4 ConfigMap

Mover configuración no sensible a ConfigMap.

### 1.6.5 Resource requests/limits

Definir valores razonables, no enormes.

### 1.6.6 Graceful termination

Verificar interacción con:

```text
terminationGracePeriodSeconds
SIGTERM
graceful shutdown
```

### 1.6.7 Helm - opcional dentro de la fase

Solo después de que manifests simples funcionen.

Crear chart con:

- image repository/tag;
- replicas;
- config;
- service port;
- resources.

## Tests / validación

En cluster local:

- kind, k3d o minikube.

Comprobar:

```bash
kubectl get pods
kubectl get svc
kubectl logs ...
```

Eliminar un backend pod y observar recuperación/reprogramación.

## GATE 1.6.0

- [ ] Manifests aplican sin errores.
- [ ] Probes funcionan.
- [ ] ConfigMap funciona.
- [ ] SIGTERM + Kubernetes no corta requests activas indebidamente.
- [ ] Se documenta la diferencia entre LB propio y Kubernetes Service.
- [ ] Tag `v1.6.0` creado.

---

# VERSION 1.7.0 - Consistent Hashing - ampliación de sistemas distribuidos

## Objetivo

Añadir una estrategia que permita afinidad determinista por una clave.

Ejemplos de clave:

- cookie;
- header;
- client identifier explícito.

Evitar usar IP como única opción porque puede comportarse mal detrás de proxies/NAT.

## Tareas

### 1.7.1 Interfaz

La interfaz actual `Next()` probablemente no tiene contexto de request suficiente.

Antes de cambiarla, diseñar cuidadosamente una interfaz compatible con estrategias request-aware.

Ejemplo conceptual:

```go
Select(*http.Request, SelectionContext) (*backend.Backend, error)
```

No hacer el cambio hasta tener tests de regresión para Round Robin, Weighted y Least Connections.

### 1.7.2 Ring

Implementar consistent hashing con virtual nodes.

Configurable:

```text
virtual_nodes
hash_key_source
```

### 1.7.3 Propiedad importante

Al añadir/eliminar un backend, solo una fracción de claves debe remapearse.

## Tests obligatorios

```text
TestConsistentHash_SameKeyReturnsSameBackend
TestConsistentHash_DifferentKeysDistribute
TestConsistentHash_BackendRemovalRemapsSubset
TestConsistentHash_SkipsUnhealthyBackend
```

## GATE 1.7.0

- [ ] Misma key es estable.
- [ ] Distribución razonable.
- [ ] Backend unhealthy no se selecciona.
- [ ] Regresión de estrategias anteriores = 0.
- [ ] Tag `v1.7.0` creado.

---

# 4. Estrategia de testing completa

## Pirámide

### Unit tests

Deben cubrir principalmente:

```text
backend state
balancing algorithms
config parsing/validation
health state transitions
retry policy
metrics behavior
```

### Integration tests

Deben cubrir:

```text
real HTTP proxying
backend failure
recovery
timeouts
retry
multi-backend distribution
```

### Smoke tests

Deben comprobar que el sistema desplegado funciona.

### Load tests

Deben medir, no sustituir tests funcionales.

---

# 5. Regla para testear bugs

Cada vez que aparezca un bug:

```text
1. reproducir bug;
2. escribir test que falle;
3. confirmar que falla por el motivo correcto;
4. implementar fix;
5. confirmar que test pasa;
6. ejecutar suite completa;
7. ejecutar race detector;
8. commit del fix.
```

Ejemplo:

Bug:

```text
Round Robin entra en loop infinito cuando todos están unhealthy.
```

Primero crear:

```text
TestRoundRobin_ReturnsErrorWhenAllBackendsUnhealthy
```

Solo después arreglarlo.

---

# 6. Reglas de concurrencia

Este proyecto se ejecutará con múltiples requests concurrentes. Por tanto:

1. Todo estado compartido debe analizarse explícitamente.
2. Usar mutex/atomic cuando proceda.
3. No introducir locks globales grandes si pueden evitarse.
4. Evitar mantener locks mientras se hacen requests HTTP.
5. Nunca hacer un health check HTTP manteniendo bloqueado el mutex del pool.
6. Ejecutar race detector en cada versión.
7. Añadir tests concurrentes especialmente para:
   - selector;
   - health state;
   - active requests;
   - metrics integration.

---

# 7. Reglas HTTP importantes

No considerar terminado el proyecto si rompe semántica HTTP básica.

Comprobar progresivamente:

- methods;
- path;
- query string;
- request body;
- response status;
- response body;
- headers relevantes;
- context cancellation;
- streaming si finalmente se soporta;
- `X-Forwarded-For` / forwarding headers según comportamiento elegido.

Documentar qué realiza automáticamente `httputil.ReverseProxy` y qué lógica es propia.

No afirmar en el CV que has implementado un proxy TCP de capa 4: este proyecto es inicialmente un **HTTP reverse proxy / Layer 7 load balancer**.

---

# 8. README final obligatorio

Cuando llegue a v1.5.0, el README debe ser tratado como parte del proyecto, no como decoración.

Estructura recomendada:

```text
# CloudBalancer

## Overview
## Why I built it
## Features
## Architecture
## Load balancing algorithms
## Health checking and failover
## Retry policy
## Observability
## Quick start
## Docker demo
## Failure/recovery demo
## Configuration
## Metrics
## Testing
## Benchmarks
## CI/CD
## Cloud deployment
## Design decisions
## Limitations
## Future work
```

## Features finales mínimas para marcar con check

```text
[x] HTTP reverse proxy
[x] Round Robin
[x] Weighted Round Robin
[x] Least Active Requests / Least Connections
[x] Active health checks
[x] Automatic failover
[x] Backend recovery
[x] Request timeout
[x] Safe retry policy
[x] Graceful shutdown
[x] YAML configuration
[x] Prometheus metrics
[x] Structured logging
[x] Docker Compose demo
[x] Unit tests
[x] Integration tests
[x] Race detector
[x] CI
[x] Container registry
[x] Cloud deployment
```

Consistent Hashing y Kubernetes pueden aparecer como extras si están terminados.

---

# 9. Qué NO hacer

Para mantener el proyecto creíble y terminable:

## No implementar todavía

- HTTP/3 propio;
- TLS termination avanzada desde cero;
- custom TCP stack;
- service mesh;
- distributed control plane;
- Raft;
- auto-scaling propio;
- GUI de administración;
- base de datos solo para guardar configuración;
- dynamic service discovery complejo antes del MVP.

Estas funcionalidades aumentan mucho el alcance y reducen la probabilidad de terminar una versión sólida.

## No usar Nginx/Envoy como núcleo

Puede compararse contra ellos, pero el objetivo del proyecto es que la lógica de balanceo sea propia.

## No hacer microservicios innecesarios

CloudBalancer debe ser un binario principal bien estructurado.

---

# 10. Checklist de revisión de cada Pull Request

Antes de mergear cualquier fase:

## Correctness

- [ ] Cumple exactamente la historia de la versión.
- [ ] Error paths están contemplados.
- [ ] No existen panics esperables por input/configuración.

## Tests

- [ ] Tests nuevos para funcionalidad nueva.
- [ ] Test del happy path.
- [ ] Test de al menos un error path.
- [ ] Tests existentes siguen pasando.
- [ ] Race detector pasa.

## Code quality

- [ ] `gofmt` limpio.
- [ ] `go vet` limpio.
- [ ] Funciones con responsabilidad clara.
- [ ] No hay duplicación importante.
- [ ] No hay dependencia innecesaria.

## Operations

- [ ] Logs útiles.
- [ ] Errores no contienen secretos.
- [ ] Shutdown/lifecycle no se ha roto.

## Documentation

- [ ] README actualizado si cambia comportamiento observable.
- [ ] Config example actualizado si cambia configuración.

---

# 11. Definition of Done final del proyecto

El proyecto puede considerarse completamente preparado para CV cuando se cumpla, como mínimo, hasta `v1.5.0`.

## Funcionalidad

- [ ] Proxy HTTP funcional.
- [ ] 3 algoritmos de balanceo.
- [ ] Health checking.
- [ ] Failover.
- [ ] Recovery.
- [ ] Timeouts.
- [ ] Retries seguros.
- [ ] Graceful shutdown.

## Calidad

- [ ] Unit tests.
- [ ] Integration tests.
- [ ] `go test -race` limpio.
- [ ] `go vet` limpio.
- [ ] CI verde.

## Observabilidad

- [ ] Prometheus.
- [ ] Grafana.
- [ ] Logs estructurados.
- [ ] Health/readiness endpoints.

## DevOps

- [ ] Dockerfile.
- [ ] Docker Compose.
- [ ] Registry.
- [ ] CI/CD.
- [ ] Cloud deployment.

## Portfolio

- [ ] README profesional.
- [ ] Diagrama de arquitectura.
- [ ] Instrucciones de demo reproducibles.
- [ ] Captura de Grafana.
- [ ] Ejemplo de failover.
- [ ] URL pública o explicación si se apaga por coste.
- [ ] Releases/tags claros.

---

# 12. Orden exacto de ejecución resumido

Codex debe seguir este orden y no saltar pasos:

```text
v0.0.0  Repo + Makefile + quality baseline
   |
   v
v0.1.0  Reverse proxy -> 1 backend
   |
   v
v0.2.0  Multiple backends + Round Robin
   |
   v
v0.3.0  Health checks + failover + recovery
   |
   v
v0.4.0  Timeouts + safe retries
   |
   v
v0.5.0  Weighted Round Robin
   |
   v
v0.6.0  Least Connections / Active Requests
   |
   v
v0.7.0  YAML configuration + validation
   |
   v
v0.8.0  Prometheus + logging + health/readiness
   |
   v
v0.9.0  Graceful shutdown
   |
   v
v1.0.0  Docker Compose + Prometheus + Grafana demo
   |
   v
v1.1.0  Full integration/failure test suite
   |
   v
v1.2.0  Benchmarks + load tests
   |
   v
v1.3.0  GitHub Actions CI
   |
   v
v1.4.0  Production image + registry
   |
   v
v1.5.0  Azure deployment + CI/CD
   |
   +----> PROJECT READY FOR CV
   |
   v
v1.6.0  Kubernetes/Helm [optional advanced]
   |
   v
v1.7.0  Consistent Hashing [optional advanced]
```

---

# 13. Primera instrucción que darle a Codex

Copiar este fichero al repositorio como:

```text
ROADMAP.md
```

Después darle a Codex una instrucción de este estilo:

```text
Lee ROADMAP.md completo.

Implementa EXCLUSIVAMENTE la VERSION 0.0.0.
No empieces ninguna tarea de v0.1.0 ni posteriores.

Sigue todas las reglas del apartado "Contrato de trabajo para Codex".
Antes de terminar, ejecuta el Quality Gate completo de v0.0.0.
Si algo falla, arréglalo antes de considerar la versión terminada.

Al final indícame:
1. archivos creados/modificados;
2. decisiones técnicas tomadas;
3. tests/comandos ejecutados y resultado;
4. checklist del GATE 0.0.0;
5. cualquier problema pendiente.

No avances automáticamente a v0.1.0.
```

Para la siguiente fase:

```text
Lee ROADMAP.md y revisa el estado actual del repositorio.

Comprueba primero que el tag/requisitos de v0.0.0 están completos.
Si no lo están, no avances.

Si v0.0.0 está realmente terminada, implementa EXCLUSIVAMENTE v0.1.0.
Ejecuta todos sus tests y Quality Gates.
No avances a v0.2.0.
```

Repetir el mismo patrón en cada versión.

---

# 14. Qué deberías saber explicar tú en una entrevista

No basta con que Codex genere el proyecto. Antes de poner una feature en el CV, debes poder explicar:

## Reverse proxy

- qué diferencia hay entre proxy y reverse proxy;
- qué ocurre con una request desde cliente hasta backend;
- por qué es Layer 7.

## Round Robin

- complejidad de selección;
- ventajas;
- limitación cuando las requests tienen costes muy distintos.

## Weighted Round Robin

- por qué existe;
- cuándo un backend necesita más peso.

## Least Connections

- por qué puede funcionar mejor con requests largas;
- por qué en tu implementación probablemente cuentas requests activas y no sockets TCP reales.

## Health checking

- diferencia entre active y passive health checking;
- qué significa false positive/false negative;
- relación entre interval, timeout y tiempo de detección.

## Failover

- qué ocurre cuando un backend cae;
- qué ocurre cuando todos caen;
- por qué 503 es apropiado cuando no hay capacidad disponible.

## Retries

- por qué un retry de POST puede duplicar efectos;
- qué es idempotencia;
- por qué necesitas máximo de intentos.

## Concurrencia

- qué estado es compartido;
- qué protegiste con mutex/atomic;
- qué detecta `go test -race`.

## Observabilidad

- diferencia entre logs, metrics y traces;
- Counter vs Gauge vs Histogram;
- qué es p95/p99;
- por qué evitar labels Prometheus de cardinalidad alta.

## Graceful shutdown

- por qué SIGTERM importa en Docker/Kubernetes;
- qué ocurre con requests activas durante un deploy.

## Cloud

- dónde se ejecuta el contenedor;
- cómo descubre backends;
- qué health probes existen;
- cómo funciona CI/CD.

Si no puedes explicar una funcionalidad sin mirar el código, dedicar una sesión a estudiarla antes de añadirla al CV.

---

# 15. Resultado esperado para el CV

Cuando se complete como mínimo v1.5.0, una descripción razonable podría ser:

```text
CloudBalancer - Go, Docker, Prometheus, Grafana, GitHub Actions, Azure

Designed and implemented a concurrent Layer 7 HTTP load balancer in Go with
Round Robin, Weighted Round Robin and Least-Active-Requests strategies,
automatic health checks and failover, safe retries, graceful shutdown and
Prometheus observability. Containerized the system with Docker Compose,
implemented unit/integration testing with race detection, and automated CI/CD
and cloud deployment on Azure.
```

No copiar esta descripción al CV hasta que las funcionalidades mencionadas estén realmente implementadas.

---

# 16. Prioridades si falta tiempo

Si hay que reducir alcance, NO recortar tests ni calidad. Recortar features avanzadas.

Orden de prioridad:

```text
IMPRESCINDIBLE
1. Reverse proxy
2. Round Robin
3. Health checks
4. Failover/recovery
5. Timeouts
6. Tests + race detector
7. Metrics
8. Docker Compose
9. CI
10. Cloud deployment

MUY RECOMENDABLE
11. Weighted Round Robin
12. Least Connections
13. Grafana
14. Load testing

EXTRA
15. Kubernetes
16. Helm
17. Consistent Hashing
```

Un proyecto más pequeño, bien testeado y desplegado vale más para un CV que uno enorme con features incompletas.

