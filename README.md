# Emergency Ops

Sistema de despacho de ambulancias con 2 microservicios (`dispatch` → `triage`, dependencia
HTTP) y acceso a base de datos, instrumentado de punta a punta con OpenTelemetry y desplegado
en AWS con Terraform + CI/CD.

Este repo contiene el **código de los servicios**. La infraestructura vive en
[emergency-ops-infrastructure](https://github.com/cristiancave/emergency-ops-infrastructure).

## Arquitectura

```
                          ┌──────────────┐
   Internet ──────────────▶     ALB      │
                          └──────┬───────┘
                    ┌────────────┼────────────────┐
                    ▼            ▼                 ▼ :3000
             /dispatch*     /triage*           Grafana
                    │            │                 │
                    ▼            ▼                 │
          ┌──────────────┐ ┌──────────────┐        │
          │   dispatch   │─▶│   triage    │        │
          │  (ECS/Go)    │HTTP│  (ECS/Go) │        │
          └──────┬───────┘ └──────┬───────┘        │
                 │  OTLP           │  OTLP           │ scrape
                 │  (traces)       │  (traces)       │
                 └────────┬────────┘                │
                          ▼                          │
                  ┌───────────────┐                  │
                  │ OTel Collector│──── X-Ray (traces)
                  │  (ECS/ADOT)   │
                  └───────┬───────┘                  │
                          │ :8888 (self metrics)      │
                          ▼                          │
                  ┌───────────────┐   scrape   ┌──────────┐
                  │  Prometheus   │◀───────────│ dispatch │
                  │   (ECS)       │◀───────────│ /triage  │
                  └───────┬───────┘   :8080/8081└──────────┘
                          └──────────────────────────┘
                                     │
                              triage │
                                     ▼
                            RDS PostgreSQL
```

Todo el tráfico interno (Collector, Prometheus, service-to-service metrics scraping) se
resuelve por DNS privado vía **AWS Cloud Map** (`*.emergency-ops.local`), no por IPs de tarea
que cambian en cada deploy.

## Los dos servicios

| Servicio | Rol | Persistencia | Puerto |
|---|---|---|---|
| `dispatch-service` | Recibe el reporte, clasifica vía `triage`, asigna la ambulancia más cercana | En memoria (fleet de ambulancias) | 8080 |
| `triage-service` | Clasifica prioridad (`RED`/`YELLOW`/`GREEN`) según síntomas/edad | PostgreSQL (RDS) | 8081 |

## Endpoints (ambiente dev)

Base: `http://emergency-ops-alb-268713301.us-east-1.elb.amazonaws.com`

| Qué | Cómo |
|---|---|
| Crear despacho | `POST /dispatch` |
| Consultar despacho | `GET /dispatch/{id}` |
| Clasificar emergencia | `POST /triage` |
| Consultar clasificación | `GET /triage/{report_id}` |
| Grafana | `:3000` — user `admin`, password: `aws secretsmanager get-secret-value --secret-id emergency-ops-grafana-admin-password --query SecretString --output text` |

No hay HTTPS configurado en dev (solo puerto 80/3000). `/health` de cada servicio no es
alcanzable vía ALB (sus reglas de path solo enrutan `/dispatch*` y `/triage*`); el healthcheck
del ALB pega directo al contenedor.

Ejemplo de `POST /dispatch` — **ojo**: `incident_latitude`/`incident_longitude` son campos
**planos** en el nivel raíz del JSON, no un objeto anidado (ver "Limitaciones conocidas" más
abajo, un error justamente en este punto quedó documentado ahí):

```json
{
  "report_id": "RPT-1",
  "patient_age": 45,
  "symptoms": ["dolor torácico"],
  "description": "texto libre",
  "incident_latitude": 40.4168,
  "incident_longitude": -3.7038
}
```

## Observabilidad (las 4 fases)

### Fase 1 — Instrumentación (código, `pkg/`)

- `pkg/telemetry`: `TracerProvider` (OTLP/gRPC → Collector) + `MeterProvider` (Prometheus pull), propagación W3C Trace Context.
- `pkg/logger`: logs JSON con `trace_id`/`span_id` extraídos del contexto activo.
- `pkg/httpclient`: cliente HTTP instrumentado (usado por `dispatch` para llamar a `triage`) — así una traza cruza ambos servicios como una sola traza, no dos desconectadas.
- Auto-instrumentación HTTP (`otelhttp`) en ambos servidores y en el cliente dispatch→triage.
- Auto-instrumentación DB (`otelsql`) en `triage` — cada query de Postgres es un span hijo, más métricas del pool de conexiones.
- Spans custom en la lógica de negocio crítica: `findBestAmbulance` (dispatch), `calculatePriority` (triage).
- `/metrics` en formato Prometheus en ambos servicios.

### Fase 2 — OTel Collector

Desplegado en ECS Fargate con la distribución ADOT (`aws-otel-collector`), config en SSM
Parameter Store (inyectada como env var, leída con el provider `env:`). Pipeline:
`receiver OTLP (gRPC+HTTP)` → `processors memory_limiter+resource+batch` → `exporters
awsxray (trazas) + prometheus (métricas)`.

### Fase 3 — Backends y visualización

- **Trazas**: AWS X-Ray.
- **Métricas**: Prometheus (self-hosted en ECS) + Grafana, dashboard *"Emergency Ops - SLIs"*
  con 6 paneles: request rate, error rate %, p99 latency, conexiones DB abiertas (saturación),
  CPU (vía datasource nativo de CloudWatch), y errores del Collector.
- **Logs**: JSON estructurado con `trace_id`/`span_id`, enviado a CloudWatch Logs vía el driver
  `awslogs` de ECS — se correlaciona con trazas usando `trace_id` como pivot.

### Fase 4 — Overhead de la instrumentación

Ver [docs/OTEL_OVERHEAD_BENCHMARK.md](docs/OTEL_OVERHEAD_BENCHMARK.md) para la metodología y
resultados completos de las dos pruebas: carga ligera (20 VUs) y carga realista sostenida (80
VUs por 5 min, con límites de CPU/memoria por contenedor). Resumen: overhead de ~6% en p99 bajo
carga ligera, ~18% bajo carga sostenida — memoria +23-30% relativo (pocos MiB absolutos en
ambos casos), CPU sin cambio neto bajo carga alta por saturación del límite del contenedor.
Scripts de carga en
[benchmark/k6-triage-load-light.js](benchmark/k6-triage-load-light.js) y
[benchmark/k6-triage-load-realistic.js](benchmark/k6-triage-load-realistic.js).

## Probar el sistema desplegado / ver los datos en vivo

- **Generar tráfico de prueba** (mezcla de requests válidas e inválidas, para que los
  dashboards no muestren solo el camino feliz):
  [`emergency-ops-infrastructure/scripts/generate-demo-traffic.ps1`](https://github.com/cristiancave/emergency-ops-infrastructure/blob/main/scripts/generate-demo-traffic.ps1)
- **Grafana**: ver tabla de endpoints arriba.
- **Trazas**: consola de X-Ray →
  `https://console.aws.amazon.com/xray/home?region=us-east-1#/traces` (ajustá el rango de
  tiempo arriba a la derecha).
- **Logs correlacionados**: CloudWatch Logs Insights sobre `/ecs/emergency-ops-dispatch` o
  `/ecs/emergency-ops-triage`, buscando por el campo `trace_id` para pivotear a la traza
  correspondiente en X-Ray.
- **Operar el ambiente en AWS** (pausar/reanudar servicios para no generar costo, conectarse a
  la RDS sin exponerla, destruir infra): ver el README de
  [emergency-ops-infrastructure](https://github.com/cristiancave/emergency-ops-infrastructure).

## Desarrollo local

```bash
cd services/triage
docker compose up -d          # Postgres + triage-service
cd ../dispatch
go run ./cmd/server            # usa TRIAGE_SERVICE_URL=http://localhost:8081 por default
```

Sin `DATABASE_URL`, `triage` cae automáticamente al repositorio en memoria — no hace falta
Postgres para desarrollar o correr tests.

## CI/CD

Push a `main` en este repo dispara (`.github/workflows/build-push.yml`):
1. `go test ./...` en ambos servicios.
2. Build + push de las imágenes Docker a ECR (contexto = raíz del repo, porque ambos servicios
   dependen del módulo local `emergencyops/pkg` vía `go.work`).

Push a `main` en `emergency-ops-infrastructure` dispara `terraform plan` + `apply` contra el
ambiente `dev`. Autenticación AWS vía OIDC (IAM Role, sin access keys estáticas en GitHub).

## Documentación adicional

- [Diagrama de arquitectura](https://github.com/cristiancave/emergency-ops-infrastructure/blob/main/docs/architecture.drawio)
  (`.drawio`, abrir en [app.diagrams.net](https://app.diagrams.net))
- [Reporte técnico completo](https://github.com/cristiancave/emergency-ops-infrastructure/blob/main/docs/Reporte_Tecnico_Emergency_Ops.docx)
  (arquitectura, decisiones de diseño, análisis de overhead)

## Limitaciones conocidas (no bloqueantes)

- `dispatch` no libera ambulancias tras el ETA — bajo carga sostenida el pool se agota (visible
  en el benchmark de Fase 4, que por eso mide contra `triage`).
- ~~`incident_latitude`/`incident_longitude` a veces devuelven `0`~~ — **resuelto**: no era un
  bug de la app. El DTO de `POST /dispatch` espera campos planos `incident_latitude`/
  `incident_longitude`, no un objeto anidado `incident_location`; varios payloads de prueba
  usados durante el desarrollo (incluido en este repo antes del fix) mandaban el objeto anidado,
  que Go ignora silenciosamente por default, dejando lat/long en `0`. Ver
  `emergency-ops-infrastructure/scripts/generate-demo-traffic.ps1` para el payload correcto.
- Sin HTTPS ni WAF en el ALB (aceptable para `dev`, no para producción).
