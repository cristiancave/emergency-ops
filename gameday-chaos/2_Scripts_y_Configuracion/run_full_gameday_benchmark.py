import time
import json
import uuid
import statistics
import concurrent.futures
import requests
import matplotlib.pyplot as plt
import os

TOXIPROXY_URL = "http://localhost:8474/proxies/triage_service"
TRIAGE_PROXY_URL = "http://localhost:8082/triage"
TRIAGE_DIRECT_URL = "http://localhost:8081/triage"
DISPATCH_URL = "http://localhost:8080/dispatch"

def send_triage_request(index, prefix="REQ", url=TRIAGE_PROXY_URL, timeout=10.0):
    start = time.time()
    rep_id = f"RPT-{prefix}-{index:04d}-{uuid.uuid4().hex[:6]}"
    payload = {
        "report_id": rep_id,
        "patient_age": 20 + (index % 60),
        "symptoms": ["dolor toracico opresivo", "disnea"] if index % 3 == 0 else ["fiebre alta", "cefalea"],
        "description": f"Reporte de prueba triage {index}"
    }
    try:
        r = requests.post(url, json=payload, timeout=timeout)
        duration_ms = (time.time() - start) * 1000.0
        return {
            "index": index,
            "report_id": rep_id,
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
            "report_id": rep_id,
            "status_code": 0,
            "duration_ms": duration_ms,
            "success": False,
            "error": str(e),
            "response": None
        }

def run_triage_batch(name, prefix, num_requests=60, max_workers=6, timeout=10.0):
    print(f"\n[+] Ejecutando lote: {name} ({num_requests} reqs, concurrencia {max_workers})...")
    start_time = time.time()
    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = [executor.submit(send_triage_request, i, prefix, TRIAGE_PROXY_URL, timeout) for i in range(1, num_requests + 1)]
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
        "samples": results[:5]
    }
    
    print(f"    - Exitosos: {len(successes)} / {len(results)} (Error: {summary['error_rate_pct']:.1f}%)")
    print(f"    - Throughput: {summary['throughput_req_sec']:.2f} req/s")
    print(f"    - Latencia Avg: {avg_lat:.2f}ms | p50: {p50:.2f}ms | p90: {p90:.2f}ms | p99: {p99:.2f}ms")
    return summary

def inject_latency_toxic(latency_ms=850, jitter_ms=150):
    print(f"\n[!] INYECCIÓN DE CAOS: Añadiendo toxic de latencia ({latency_ms}ms ± {jitter_ms}ms) en Toxiproxy...")
    try:
        requests.delete(f"{TOXIPROXY_URL}/toxics/latency_chaos")
    except Exception:
        pass
    toxic_payload = {
        "name": "latency_chaos",
        "type": "latency",
        "stream": "upstream",
        "toxicity": 1.0,
        "attributes": {
            "latency": latency_ms,
            "jitter": jitter_ms
        }
    }
    r = requests.post(f"{TOXIPROXY_URL}/toxics", json=toxic_payload)
    print(f"    Toxiproxy respuesta: {r.status_code}")
    return r.status_code in [200, 201]

def remove_latency_toxic():
    print("\n[+] ROLLBACK AUTOMÁTICO: Removiendo toxic de latencia...")
    try:
        r = requests.delete(f"{TOXIPROXY_URL}/toxics/latency_chaos")
        print(f"    Rollback status: {r.status_code}")
        return r.status_code in [200, 204]
    except Exception as e:
        print(f"    Rollback exception: {e}")
        return False

def main():
    print("==========================================================================")
    print("   GAME DAY CHAOS EXPERIMENT: RESILIENCIA EN MICROSERVICIOS EMERGENCY OPS ")
    print("==========================================================================")
    
    # 1. Baseline
    remove_latency_toxic()
    time.sleep(1)
    baseline = run_triage_batch("1. Estado Estable (Baseline)", "BASE", num_requests=60, max_workers=6)
    
    # 2. Chaos Injection
    inject_latency_toxic(latency_ms=850, jitter_ms=150)
    time.sleep(2)
    chaos = run_triage_batch("2. Inyección de Caos (Latencia + Jitter)", "CHAOS", num_requests=60, max_workers=6)
    
    # 3. Rollback & Recovery
    rollback_t0 = time.time()
    remove_latency_toxic()
    time.sleep(1)
    mttr = time.time() - rollback_t0
    recovery = run_triage_batch("3. Estado Post-Rollback (Recuperado)", "REC", num_requests=60, max_workers=6)
    
    print("\n==========================================================================")
    print("                        RESULTADOS FINALES DEL GAME DAY                   ")
    print("==========================================================================")
    print(f"{'Métrica':<25} | {'Estado Estable':<16} | {'Estado en Caos':<16} | {'Recuperado':<16}")
    print("-" * 79)
    print(f"{'Total Peticiones':<25} | {baseline['total_requests']:<16} | {chaos['total_requests']:<16} | {recovery['total_requests']:<16}")
    print(f"{'Peticiones Exitosas':<25} | {baseline['successful_requests']:<16} | {chaos['successful_requests']:<16} | {recovery['successful_requests']:<16}")
    print(f"{'Tasa de Error (%)':<25} | {baseline['error_rate_pct']:<15.1f}% | {chaos['error_rate_pct']:<15.1f}% | {recovery['error_rate_pct']:<15.1f}%")
    print(f"{'Throughput (req/s)':<25} | {baseline['throughput_req_sec']:<16.2f} | {chaos['throughput_req_sec']:<16.2f} | {recovery['throughput_req_sec']:<16.2f}")
    print(f"{'Latencia Promedio (ms)':<25} | {baseline['latency_avg_ms']:<16.2f} | {chaos['latency_avg_ms']:<16.2f} | {recovery['latency_avg_ms']:<16.2f}")
    print(f"{'Latencia p50 (ms)':<25} | {baseline['latency_p50_ms']:<16.2f} | {chaos['latency_p50_ms']:<16.2f} | {recovery['latency_p50_ms']:<16.2f}")
    print(f"{'Latencia p90 (ms)':<25} | {baseline['latency_p90_ms']:<16.2f} | {chaos['latency_p90_ms']:<16.2f} | {recovery['latency_p90_ms']:<16.2f}")
    print(f"{'Latencia p99 (ms)':<25} | {baseline['latency_p99_ms']:<16.2f} | {chaos['latency_p99_ms']:<16.2f} | {recovery['latency_p99_ms']:<16.2f}")
    print("=" * 79)
    
    # Export full dataset
    dataset = {
        "experiment_id": "EXP-01-CHAOS-GAME-DAY",
        "service": "Emergency Ops (triage-service & dispatch-service)",
        "injected_fault": "Network Latency 850ms (±150ms jitter) via Toxiproxy/tc simulation",
        "mttr_seconds": mttr,
        "metrics": {
            "baseline": baseline,
            "chaos": chaos,
            "recovery": recovery
        }
    }
    
    with open("gameday_results_summary.json", "w", encoding="utf-8") as f:
        json.dump(dataset, f, indent=2)
    print("\n[+] Resumen JSON guardado en gameday_results_summary.json")
    
    # Plot high-resolution publication-quality graphs
    generate_publication_charts(baseline, chaos, recovery)

def generate_publication_charts(baseline, chaos, recovery):
    fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(11, 4.2), dpi=300)
    
    c_blue = '#2563eb'
    c_orange = '#ea580c'
    c_red = '#dc2626'
    c_green = '#16a34a'
    
    phases = ['Estado Estable\n(Basal)', 'Inyección Caos\n(+850ms latency)', 'Post-Rollback\n(Recuperado)']
    x = [0, 1, 2]
    w = 0.22
    
    p50s = [baseline['latency_p50_ms'], chaos['latency_p50_ms'], recovery['latency_p50_ms']]
    p90s = [baseline['latency_p90_ms'], chaos['latency_p90_ms'], recovery['latency_p90_ms']]
    p99s = [baseline['latency_p99_ms'], chaos['latency_p99_ms'], recovery['latency_p99_ms']]
    
    ax1.bar([i - w for i in x], p50s, width=w, label='p50 (Mediana)', color=c_blue)
    ax1.bar(x, p90s, width=w, label='p90 Latencia', color=c_orange)
    ax1.bar([i + w for i in x], p99s, width=w, label='p99 Latencia', color=c_red)
    
    ax1.set_title('Impacto en Latencia por Percentil (SLI: Latency)', fontsize=11, fontweight='bold', pad=8)
    ax1.set_ylabel('Tiempo de Respuesta (ms)', fontsize=10)
    ax1.set_xticks(x)
    ax1.set_xticklabels(phases, fontsize=9)
    ax1.grid(axis='y', linestyle=':', alpha=0.7)
    ax1.legend(loc='upper left', fontsize=8, framealpha=0.9)
    
    tps = [baseline['throughput_req_sec'], chaos['throughput_req_sec'], recovery['throughput_req_sec']]
    ax2.plot(phases, tps, marker='s', markersize=8, linewidth=2.5, color=c_green, label='Throughput (req/s)')
    ax2.set_title('Throughput de Despacho (req/s)', fontsize=11, fontweight='bold', pad=8)
    ax2.set_ylabel('Throughput Efectivo (req/s)', fontsize=10)
    ax2.set_ylim(0, max(tps) * 1.35)
    for i, val in enumerate(tps):
        ax2.annotate(f'{val:.1f} req/s', (i, val), textcoords="offset points", xytext=(0, 10), ha='center', fontweight='bold', fontsize=9, color='#145a32')
    ax2.grid(True, linestyle=':', alpha=0.7)
    ax2.legend(loc='upper right', fontsize=8, framealpha=0.9)
    
    plt.tight_layout()
    plt.savefig('gameday_impact_chart.png', dpi=300)
    print("[+] Gráficos de alta resolución guardados en gameday_impact_chart.png")

if __name__ == "__main__":
    main()
