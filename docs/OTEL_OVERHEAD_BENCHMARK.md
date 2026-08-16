# Fase 4 — Análisis de overhead de instrumentación OTel

## Metodología

Se comparó `triage-service` en dos versiones, construidas desde el mismo repo en distintos
commits:

- **Baseline** (`91a151f`): HTTP + acceso a Postgres, **sin** OpenTelemetry.
- **Instrumentado** (`HEAD`): igual, más OTel SDK completo — auto-instrumentación HTTP
  (`otelhttp`), auto-instrumentación DB (`otelsql`), un span custom (`calculatePriority`),
  logs JSON con `trace_id`/`span_id`, y exportación OTLP real a un Collector (Jaeger local),
  no a un endpoint caído — así el overhead medido incluye el costo real de exportar, no
  reintentos fallidos.

Se eligió `triage-service` (no `dispatch`) porque es el servicio con DB real y span custom;
además `dispatch` tiene una limitación de dominio (el pool de ambulancias se agota bajo carga
sostenida al no existir un flujo de liberación) que sesga cualquier comparación de throughput.

**Entorno**: ambas versiones corrieron en contenedores Docker idénticos (misma imagen base,
mismo Postgres 16 fresco, misma red Docker aislada), en la misma máquina, secuencialmente
(nunca en simultáneo, para no compartir CPU entre corridas).

**Carga**: k6, rampa 0→20 VUs en 10s, sostenido 20 VUs por 30s, baja a 0 en 5s. Cada request
es un `POST /triage` con un `report_id` único.

**Métricas**: p99/avg de `http_req_duration` (k6), CPU%/memoria de `docker stats` muestreado
cada 2s durante toda la corrida.

## Resultados

| Métrica | Baseline (sin OTel) | Instrumentado (con OTel) | Overhead |
|---|---|---|---|
| Requests completados (45s) | 13,549 | 12,487 | -7.8% throughput |
| Throughput (req/s) | 301.1 | 277.4 | -7.9% |
| Latencia promedio | 54.72 ms | 59.38 ms | **+4.66 ms (+8.5%)** |
| Latencia p90 | 119.37 ms | 126.13 ms | +6.76 ms (+5.7%) |
| Latencia p95 | 143.63 ms | 148.91 ms | +5.28 ms (+3.7%) |
| **Latencia p99** | 197.86 ms | 209.46 ms | **+11.6 ms (+5.9%)** |
| CPU promedio (contenedor) | 157.6% | 161.5% | **+3.9 pp (+2.5%)** |
| CPU pico | 210.9% | 206.0% | -4.9 pp (ruido de muestreo) |
| Memoria promedio | 11.77 MiB | 14.53 MiB | **+2.76 MiB (+23.4%)** |
| Memoria pico | 12.57 MiB | 17.17 MiB | +4.60 MiB (+36.6%) |
| Tasa de error | 0% | 0% | sin cambio |

## Interpretación

- **Latencia**: overhead de un solo dígito en p99 (~6%), consistente con lo esperado para
  auto-instrumentación HTTP+DB con exportación por lotes (`batch` processor) — el SDK no
  bloquea el request path para exportar, solo agrega el costo de crear/cerrar spans y
  serializar atributos.
- **CPU**: overhead marginal (~2.5%) en promedio. El pico más alto del baseline es ruido de
  muestreo (`docker stats` con intervalo de 2s en una corrida de 45s captura pocas muestras),
  no una tendencia real.
- **Memoria**: el overhead relativo (+23% promedio, +37% pico) es el más notorio, esperable
  dado que ambos procesos parten de una base muy chica (~12 MiB) — el SDK de OTel (buffers de
  spans, batching, exportador gRPC) añade unos pocos MiB fijos que pesan proporcionalmente
  mucho sobre una base tan pequeña. En términos absolutos (~3-5 MiB) es irrelevante para el
  `otel_collector_memory`/`triage_memory` de 512 MB asignados en ECS.
- **Throughput**: la caída de ~8% a la misma cantidad de VUs es coherente con el aumento de
  latencia por request (a más tiempo por request, menos requests/s con VUs fijos) — no indica
  saturación ni errores.

## Conclusión

Con exportación funcionando (Collector alcanzable), el overhead de la instrumentación OTel
completa (HTTP + DB auto-instrumentado + span custom + logs correlacionados) es **de un solo
dígito en latencia y CPU**, y unos pocos MiB de memoria en términos absolutos. Para el tamaño
de task definido en ECS (`triage_memory = 512`, `triage_cpu = 256`), el overhead medido no
justifica cambiar el sizing.
