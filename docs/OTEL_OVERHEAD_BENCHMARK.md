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
(nunca en simultáneo).

Se corrieron **dos pruebas de carga distintas**, descritas abajo, con `script.js` (k6) haciendo
`POST /triage` con un `report_id` único por request.

---

## Prueba 1 — Carga ligera (20 VUs, 45s)

Rampa 0→20 VUs en 10s, sostenido 20 VUs por 30s, baja a 0 en 5s. Contenedores **sin** límite de
CPU/memoria explícito (usan lo que el host les da).

### Resultados

| Métrica | Baseline (sin OTel) | Instrumentado (con OTel) | Overhead |
|---|---|---|---|
| Requests completados (45s) | 13,549 | 12,487 | -7.8% throughput |
| Throughput (req/s) | 301.1 | 277.4 | -7.9% |
| Latencia promedio | 54.72 ms | 59.38 ms | +4.66 ms (+8.5%) |
| Latencia p90 | 119.37 ms | 126.13 ms | +6.76 ms (+5.7%) |
| Latencia p95 | 143.63 ms | 148.91 ms | +5.28 ms (+3.7%) |
| **Latencia p99** | 197.86 ms | 209.46 ms | **+11.6 ms (+5.9%)** |
| CPU promedio (contenedor) | 157.6% | 161.5% | +3.9 pp (+2.5%) |
| CPU pico | 210.9% | 206.0% | -4.9 pp (ruido de muestreo) |
| Memoria promedio | 11.77 MiB | 14.53 MiB | +2.76 MiB (+23.4%) |
| Memoria pico | 12.57 MiB | 17.17 MiB | +4.60 MiB (+36.6%) |
| Tasa de error | 0% | 0% | sin cambio |

---

## Prueba 2 — Carga realista (80 VUs concurrentes, 5 minutos)

Rampa 0→80 VUs en 30s, sostenido 80 VUs por **5 minutos**, baja a 0 en 15s — el rango de carga
recomendado (50-100 usuarios concurrentes) para que el overhead se mida bajo presión sostenida,
no solo en una ráfaga corta.

### Lección de metodología: limitar CPU por contenedor

El primer intento de esta prueba corrió **sin** límite de CPU por contenedor, igual que la
Prueba 1. A 80 VUs sostenidos por 5 minutos el resultado fue contraintuitivo: la versión
instrumentada dio **mejor** p99 y throughput que el baseline (imposible mecánicamente — instrumentar
no puede volver un proceso más rápido). La causa: sin límite de CPU, ambos contenedores compiten
por los recursos reales de la máquina (una laptop compartida con Docker Desktop, no un entorno
dedicado), y a 80 VUs sostenidos el ruido del host (picos de CPU de otros procesos, throttling
térmico, overhead del propio Docker Desktop) supera la señal real que se quiere medir — el
baseline incluso tuvo 44 requests fallidos bajo esa contención, el instrumentado ninguno.

La corrección: repetir ambas corridas con `docker run --cpus=2 --memory=1g`, fijando un techo de
recursos igual para ambas versiones. Esto no cambia lo que se mide (el comportamiento de la
app), pero aísla la medición del ruido del resto de procesos del host. Los resultados de abajo
son de esa segunda corrida (con límites).

### Resultados (con `--cpus=2 --memory=1g`)

| Métrica | Baseline (sin OTel) | Instrumentado (con OTel) | Overhead |
|---|---|---|---|
| Requests completados (5m45s) | 119,566 | 107,687 | -9.9% throughput |
| Throughput (req/s) | 346.6 | 312.2 | -9.9% |
| Latencia promedio | 215.35 ms | 239.11 ms | +23.76 ms (+11.0%) |
| Latencia p90 | 399.56 ms | 443.18 ms | +43.62 ms (+10.9%) |
| Latencia p95 | 448.28 ms | 507.68 ms | +59.40 ms (+13.2%) |
| **Latencia p99** | 579.72 ms | 681.71 ms | **+101.99 ms (+17.6%)** |
| CPU promedio (contenedor, cap 2 vCPU = 200%) | 145.6% | 144.9% | -0.5% (saturado en ambos) |
| CPU pico | 160.5% | 162.3% | +1.8% |
| Memoria promedio | 20.58 MiB | 26.75 MiB | **+6.17 MiB (+30.0%)** |
| Memoria pico | 23.54 MiB | 31.56 MiB | +8.02 MiB (+34.1%) |
| Tasa de error | 0% | 0% | sin cambio |

---

## Interpretación

- **El overhead crece con la carga**: en la Prueba 1 (20 VUs) el overhead en p99 fue ~6%; en la
  Prueba 2 (80 VUs sostenidos) subió a ~17.6%. Es el patrón esperado — a mayor concurrencia, más
  spans/exports simultáneos compitiendo por el mismo presupuesto de CPU, más se nota el costo
  fijo por request de crear/serializar spans y ejecutar las queries instrumentadas.
- **CPU se satura igual en ambas versiones bajo carga alta con límite fijo** (~145% de 200%
  disponibles, ambas casi idénticas): con el contenedor topeado en 2 vCPUs, el overhead de OTel
  no se ve como "más % de CPU" porque ya no hay más CPU para dar — se ve como **menos throughput
  y más latencia** en su lugar (la misma CPU tiene que repartirse entre más trabajo por request).
  Esto es un hallazgo más preciso que la Prueba 1, donde el CPU no estaba saturado y sí se
  alcanzaba a ver una diferencia directa en %CPU.
- **Memoria**: overhead consistente en ambas pruebas (+23-30% relativo, unos pocos MiB
  absolutos) — la parte más estable de la medición, menos sensible al ruido de scheduling que
  CPU/latencia.
- **Confiabilidad de la medición**: la Prueba 2 demuestra por qué limitar recursos por
  contenedor es necesario para benchmarks de overhead en hardware compartido — sin eso, el ruido
  del host puede invertir el resultado (como pasó en el primer intento sin límites).

## Conclusión

Bajo carga ligera (20 VUs) el overhead de OpenTelemetry es marginal (~6% p99, ~2.5% CPU). Bajo
carga sostenida realista (80 VUs por 5 minutos, con recursos acotados) el overhead es más
notorio pero sigue siendo moderado: **~17.6% en p99, memoria +30% relativo (pocos MiB
absolutos), CPU sin cambio neto por saturación del límite**. Para el tamaño de task definido en
ECS (`triage_memory = 512 MB`, `triage_cpu = 256`, muy por debajo de los 2 vCPU/1GB usados en
este benchmark), el overhead medido sigue sin justificar un cambio de sizing, aunque bajo tráfico
de producción sostenido valdría la pena monitorear p99 real vía el dashboard de Grafana en vez de
asumir que el comportamiento de este benchmark escala linealmente.
