import time
import json
import statistics
import concurrent.futures
import requests
import matplotlib.pyplot as plt
import os

DISPATCH_URL = "http://localhost:8080/dispatch"
TOXIPROXY_URL = "http://localhost:8474/proxies/triage_service"
METRICS_DISPATCH_URL = "http://localhost:8080/metrics"
METRICS_TRIAGE_URL = "http://localhost:8081/metrics"

def send_dispatch_request(index, timeout=10.0):
    start = time.time()
    payload = {
        "report_id": f"RPT-CHAOS-{index:04d}",
        "patient_age": 20 + (index % 60),
        "symptoms": ["dolor toracico", "dificultad respiratoria"] if index % 2 == 0 else ["fiebre", "tos"],
        "description": f"Prueba de carga y caos reporte {index}",
        "incident_latitude": 40.4168 + (index * 0.001),
        "incident_longitude": -3.7038 + (index * 0.001)
    }
    try:
        r = requests.post(DISPATCH_URL, json=payload, timeout=timeout)
        duration_ms = (time.time() - start) * 1000.0
        return {
            "index": index,
            "status_code": r.status_code,
            "duration_ms": duration_ms,
            "success": r.status_code == 201,
            "error": None if r.status_code == 201 else r.text,
            "response": r.json() if r.status_code == 201 else None
        }
    except Exception as e:
        duration_ms = (time.time() - start) * 1000.0
        return {
            "index": index,
            "status_code": 0,
            "duration_ms": duration_ms,
            "success": False,
            "error": str(e),
            "response": None
        }

def run_load_batch(name, num_requests=60, max_workers=6, timeout=10.0):
    print(f"\n[+] Iniciando lote: {name} ({num_requests} peticiones, concurrencia {max_workers})...")
    start_time = time.time()
    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = [executor.submit(send_dispatch_request, i, timeout) for i in range(1, num_requests + 1)]
        for future in concurrent.futures.as_completed(futures):
            results.append(future.result())
    
    total_time = time.time() - start_time
    durations = [r["duration_ms"] for r in results]
    successes = [r for r in results if r["success"]]
    failures = [r for r in results if not r["success"]]
    
    durations_sorted = sorted(durations)
    p50 = durations_sorted[int(len(durations) * 0.50)]
    p90 = durations_sorted[int(len(durations) * 0.90)]
    p95 = durations_sorted[int(len(durations) * 0.95)]
    p99 = durations_sorted[min(int(len(durations) * 0.99), len(durations)-1)]
    avg_lat = statistics.mean(durations)
    
    summary = {
        "phase": name,
        "total_requests": len(results),
        "successful_requests": len(successes),
        "failed_requests": len(failures),
        "error_rate_pct": (len(failures) / len(results)) * 100.0,
        "total_duration_sec": total_time,
        "throughput_req_sec": len(results) / total_time,
        "latency_avg_ms": avg_lat,
        "latency_min_ms": min(durations),
        "latency_max_ms": max(durations),
        "latency_p50_ms": p50,
        "latency_p90_ms": p90,
        "latency_p95_ms": p95,
        "latency_p99_ms": p99,
        "raw_results": results
    }
    
    print(f"    - Exitosos: {len(successes)} / {len(results)} (Error: {summary['error_rate_pct']:.1f}%)")
    print(f"    - Throughput: {summary['throughput_req_sec']:.2f} req/s")
    print(f"    - Latencia Avg: {avg_lat:.2f}ms | p50: {p50:.2f}ms | p90: {p90:.2f}ms | p99: {p99:.2f}ms")
    return summary

def inject_latency_toxic(latency_ms=900, jitter_ms=200):
    print(f"\n[!] INYECTANDO CAOS: Latencia de red {latency_ms}ms (±{jitter_ms}ms) en enlace dispatch -> triage...")
    toxic_payload = {
        "name": "triage_latency_chaos",
        "type": "latency",
        "stream": "upstream",
        "toxicity": 1.0,
        "attributes": {
            "latency": latency_ms,
            "jitter": jitter_ms
        }
    }
    r = requests.post(f"{TOXIPROXY_URL}/toxics", json=toxic_payload)
    print(f"    Toxiproxy response: {r.status_code}")
    return r.status_code in [200, 201]

def remove_latency_toxic():
    print("\n[+] ROLLBACK AUTOMÁTICO: Removiendo inyección de toxic en Toxiproxy...")
    r = requests.delete(f"{TOXIPROXY_URL}/toxics/triage_latency_chaos")
    print(f"    Rollback response: {r.status_code}")
    return r.status_code in [200, 204]

def main():
    print("================================================================")
    print("   PROGRAMA DE CHAOS ENGINEERING - SISTEMA EMERGENCY OPS      ")
    print("================================================================")
    
    # 1. Baseline: Steady State
    baseline = run_load_batch("1. Estado Estable (Baseline)", num_requests=50, max_workers=5)
    
    # 2. Chaos Injection
    inject_latency_toxic(latency_ms=1100, jitter_ms=250)
    time.sleep(2) # estabilizar inyección
    
    # 3. Chaos Execution
    chaos = run_load_batch("2. Estado en Caos (Latencia + Jitter)", num_requests=50, max_workers=5)
    
    # 4. Rollback
    rollback_start = time.time()
    remove_latency_toxic()
    time.sleep(1) # drenar buffers
    recovery_time = time.time() - rollback_start
    
    # 5. Recovery Verification
    recovery = run_load_batch("3. Estado Recuperado (Post-Rollback)", num_requests=50, max_workers=5)
    
    # Summary report
    print("\n================================================================")
    print("                 RESUMEN COMPARATIVO DEL GAME DAY               ")
    print("================================================================")
    print(f"{'Métrica':<25} | {'Estado Estable':<15} | {'En Caos':<15} | {'Recuperado':<15}")
    print("-" * 75)
    print(f"{'Total Requests':<25} | {baseline['total_requests']:<15} | {chaos['total_requests']:<15} | {recovery['total_requests']:<15}")
    print(f"{'Tasa de Error (%)':<25} | {baseline['error_rate_pct']:<14.1f}% | {chaos['error_rate_pct']:<14.1f}% | {recovery['error_rate_pct']:<14.1f}%")
    print(f"{'Throughput (req/s)':<25} | {baseline['throughput_req_sec']:<15.2f} | {chaos['throughput_req_sec']:<15.2f} | {recovery['throughput_req_sec']:<15.2f}")
    print(f"{'Latencia Promedio (ms)':<25} | {baseline['latency_avg_ms']:<15.2f} | {chaos['latency_avg_ms']:<15.2f} | {recovery['latency_avg_ms']:<15.2f}")
    print(f"{'Latencia p50 (ms)':<25} | {baseline['latency_p50_ms']:<15.2f} | {chaos['latency_p50_ms']:<15.2f} | {recovery['latency_p50_ms']:<15.2f}")
    print(f"{'Latencia p90 (ms)':<25} | {baseline['latency_p90_ms']:<15.2f} | {chaos['latency_p90_ms']:<15.2f} | {recovery['latency_p90_ms']:<15.2f}")
    print(f"{'Latencia p99 (ms)':<25} | {baseline['latency_p99_ms']:<15.2f} | {chaos['latency_p99_ms']:<15.2f} | {recovery['latency_p99_ms']:<15.2f}")
    print("=" * 75)
    
    # Save experiment results to JSON
    output_data = {
        "experiment_id": "EXP-CHAOS-01-NET-LATENCY",
        "date": "2026-08-28",
        "system": "Emergency Ops (dispatch-service -> triage-service)",
        "recovery_time_sec": recovery_time,
        "baseline": baseline,
        "chaos": chaos,
        "recovery": recovery
    }
    
    with open("chaos_experiment_results.json", "w", encoding="utf-8") as f:
        json.dump(output_data, f, indent=2)
    print("\n[+] Resultados exportados a chaos_experiment_results.json")
    
    # Generate Charts
    generate_charts(baseline, chaos, recovery)

def generate_charts(baseline, chaos, recovery):
    phases = ['Estado Estable', 'En Caos (Latencia)', 'Recuperación']
    p50_vals = [baseline['latency_p50_ms'], chaos['latency_p50_ms'], recovery['latency_p50_ms']]
    p90_vals = [baseline['latency_p90_ms'], chaos['latency_p90_ms'], recovery['latency_p90_ms']]
    p99_vals = [baseline['latency_p99_ms'], chaos['latency_p99_ms'], recovery['latency_p99_ms']]
    throughput_vals = [baseline['throughput_req_sec'], chaos['throughput_req_sec'], recovery['throughput_req_sec']]
    
    fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(12, 5))
    
    # Latency percentiles chart
    x = range(len(phases))
    width = 0.25
    ax1.bar([i - width for i in x], p50_vals, width=width, label='p50 Latencia', color='#2b5c8f')
    ax1.bar(x, p90_vals, width=width, label='p90 Latencia', color='#e67e22')
    ax1.bar([i + width for i in x], p99_vals, width=width, label='p99 Latencia', color='#c0392b')
    ax1.set_title('Impacto en Latencia (SLI) por Fase de Caos', fontsize=12, fontweight='bold', pad=10)
    ax1.set_ylabel('Latencia (ms)', fontsize=10)
    ax1.set_xticks(x)
    ax1.set_xticklabels(phases, fontsize=9)
    ax1.legend(loc='upper left', frameon=True)
    ax1.grid(axis='y', linestyle='--', alpha=0.6)
    
    # Throughput chart
    ax2.plot(phases, throughput_vals, marker='o', color='#27ae60', linewidth=2.5, markersize=8)
    ax2.set_title('Throughput de Despacho (req/s)', fontsize=12, fontweight='bold', pad=10)
    ax2.set_ylabel('Peticiones por Segundo', fontsize=10)
    ax2.set_ylim(0, max(throughput_vals) * 1.3)
    for i, v in enumerate(throughput_vals):
        ax2.text(i, v + (max(throughput_vals)*0.04), f"{v:.2f} r/s", ha='center', fontweight='bold', color='#1e824c')
    ax2.grid(True, linestyle='--', alpha=0.6)
    
    plt.tight_layout()
    plt.savefig('chaos_experiment_chart.png', dpi=300)
    print("[+] Gráfico de observabilidad guardado en chaos_experiment_chart.png")

if __name__ == "__main__":
    main()
