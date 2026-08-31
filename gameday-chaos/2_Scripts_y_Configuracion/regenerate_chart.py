import matplotlib.pyplot as plt

def generate_publication_charts():
    # Set clean academic / engineering style
    plt.rcParams['font.family'] = 'sans-serif'
    plt.rcParams['font.sans-serif'] = ['DejaVu Sans', 'Arial', 'Helvetica']
    plt.rcParams['text.color'] = '#111827'
    plt.rcParams['axes.labelcolor'] = '#111827'
    plt.rcParams['xtick.color'] = '#374151'
    plt.rcParams['ytick.color'] = '#374151'

    baseline = {
        'latency_p50_ms': 42.42, 'latency_p90_ms': 67.04, 'latency_p99_ms': 78.72, 'throughput_req_sec': 139.02
    }
    chaos = {
        'latency_p50_ms': 858.71, 'latency_p90_ms': 1001.02, 'latency_p99_ms': 1025.24, 'throughput_req_sec': 6.62
    }
    recovery = {
        'latency_p50_ms': 24.09, 'latency_p90_ms': 47.19, 'latency_p99_ms': 63.56, 'throughput_req_sec': 191.37
    }

    fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(10.5, 3.8), dpi=300)
    fig.patch.set_facecolor('#ffffff')
    ax1.set_facecolor('#ffffff')
    ax2.set_facecolor('#ffffff')

    # Executive / Academic Palette: Muted Slate, Warm Ochre, Crimson
    c_p50 = '#475569'    # Slate
    c_p90 = '#d97706'    # Muted Amber
    c_p99 = '#b91c1c'    # Deep Crimson
    c_tp = '#1e293b'     # Dark Slate

    phases = ['Estado Estable\n(Basal)', 'Inyección Caos\n(+850ms latency)', 'Post-Rollback\n(Recuperado)']
    x = [0, 1, 2]
    w = 0.22

    p50s = [baseline['latency_p50_ms'], chaos['latency_p50_ms'], recovery['latency_p50_ms']]
    p90s = [baseline['latency_p90_ms'], chaos['latency_p90_ms'], recovery['latency_p90_ms']]
    p99s = [baseline['latency_p99_ms'], chaos['latency_p99_ms'], recovery['latency_p99_ms']]

    ax1.bar([i - w for i in x], p50s, width=w, label='p50 (Mediana)', color=c_p50, edgecolor='#1e293b', linewidth=0.6)
    ax1.bar(x, p90s, width=w, label='p90 Latencia', color=c_p90, edgecolor='#78350f', linewidth=0.6)
    ax1.bar([i + w for i in x], p99s, width=w, label='p99 Latencia', color=c_p99, edgecolor='#7f1d1d', linewidth=0.6)

    ax1.set_title('Figura 1: Impacto en Latencia (SLI: Latency en ms)', fontsize=10, fontweight='bold', pad=10, color='#111827')
    ax1.set_ylabel('Tiempo de Respuesta (ms)', fontsize=9)
    ax1.set_xticks(x)
    ax1.set_xticklabels(phases, fontsize=8.5)
    ax1.grid(axis='y', linestyle='--', alpha=0.4, color='#94a3b8')
    ax1.legend(loc='upper left', fontsize=8, frameon=True, edgecolor='#cbd5e1')
    ax1.spines['top'].set_visible(False)
    ax1.spines['right'].set_visible(False)

    tps = [baseline['throughput_req_sec'], chaos['throughput_req_sec'], recovery['throughput_req_sec']]
    ax2.plot(phases, tps, marker='o', markersize=6, linewidth=1.8, color=c_tp, label='Throughput real', markeredgecolor='#000000')
    ax2.set_title('Figura 2: Throughput del Sistema (req/s)', fontsize=10, fontweight='bold', pad=10, color='#111827')
    ax2.set_ylabel('Peticiones / Segundo (req/s)', fontsize=9)
    ax2.set_ylim(-15, max(tps) * 1.30)

    offsets = [(0, 10), (0, 10), (0, 10)]
    for i, (val, off) in enumerate(zip(tps, offsets)):
        ax2.annotate(f'{val:.1f} r/s', (i, val), textcoords="offset points", xytext=off, ha='center', fontsize=8, fontweight='bold', color='#1e293b')
    
    ax2.grid(True, linestyle='--', alpha=0.4, color='#94a3b8')
    ax2.legend(loc='upper right', fontsize=8, frameon=True, edgecolor='#cbd5e1')
    ax2.spines['top'].set_visible(False)
    ax2.spines['right'].set_visible(False)

    plt.tight_layout()
    plt.savefig('gameday_impact_chart.png', dpi=300)
    print("[+] Gráfico regenerado con estilo sobrio y académico.")

if __name__ == "__main__":
    generate_publication_charts()
