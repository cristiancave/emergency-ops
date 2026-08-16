// Benchmark de overhead OTel (Fase 4). Uso:
//
//   docker run --rm --network <net> -e TARGET_URL=http://<triage-host>:8081 \
//     -v "$(pwd)/benchmark:/scripts" grafana/k6:latest run /scripts/k6-triage-load.js
//
// Ver docs/OTEL_OVERHEAD_BENCHMARK.md para la metodología completa y resultados.

import http from 'k6/http';
import { check } from 'k6';

export const options = {
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
  scenarios: {
    load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 20 },
        { duration: '30s', target: 20 },
        { duration: '5s', target: 0 },
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
  const reportId = `BENCH-${__VU}-${__ITER}-${counter}-${Date.now()}`;

  const payload = JSON.stringify({
    report_id: reportId,
    patient_age: 30 + (counter % 50),
    symptoms: ['fiebre alta'],
    description: 'k6 overhead benchmark',
  });

  const res = http.post(`${__ENV.TARGET_URL}/triage`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'status is 201': (r) => r.status === 201,
  });
}
