// Benchmark de overhead OTel (Fase 4) - carga REALISTA (Prueba 2: 80 VUs, 5 min).
// Uso:
//
//   docker run --rm --network <net> -e TARGET_URL=http://<triage-host>:8081 \
//     -v "$(pwd)/benchmark:/scripts" grafana/k6:latest run /scripts/k6-triage-load-realistic.js
//
// Para una medición confiable en hardware compartido (laptop, no un runner dedicado),
// correr el contenedor de triage con límites fijos de CPU/memoria (ej. `docker run
// --cpus=2 --memory=1g`) en AMBAS corridas (baseline e instrumentada) — sin eso, el
// ruido del host puede superar la señal real que se quiere medir bajo esta carga
// sostenida. Ver la sección "Lección de metodología" en docs/OTEL_OVERHEAD_BENCHMARK.md.

import http from 'k6/http';
import { check } from 'k6';

export const options = {
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
  scenarios: {
    load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 80 },
        { duration: '5m', target: 80 },
        { duration: '15s', target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'],
  },
};

let counter = 0;

export default function () {
  counter += 1;
  const reportId = `BENCH-REALISTIC-${__VU}-${__ITER}-${counter}-${Date.now()}`;

  const payload = JSON.stringify({
    report_id: reportId,
    patient_age: 30 + (counter % 50),
    symptoms: ['fiebre alta'],
    description: 'k6 overhead benchmark - carga realista',
  });

  const res = http.post(`${__ENV.TARGET_URL}/triage`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'status is 201': (r) => r.status === 201,
  });
}
