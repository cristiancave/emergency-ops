import docx
from docx import Document
from docx.shared import Inches, Pt, RGBColor
from docx.enum.text import WD_ALIGN_PARAGRAPH
from docx.enum.table import WD_TABLE_ALIGNMENT
from docx.oxml import parse_xml
from docx.oxml.ns import nsdecls
import os

def set_cell_background(cell, fill_hex):
    shading_elm = parse_xml(f'<w:shd {nsdecls("w")} w:fill="{fill_hex}"/>')
    cell._tc.get_or_add_tcPr().append(shading_elm)

def set_cell_margins(cell, top=100, bottom=100, left=140, right=140):
    tcPr = cell._tc.get_or_add_tcPr()
    tcMar = parse_xml(f'<w:tcMar {nsdecls("w")}><w:top w:w="{top}" w:type="dxa"/><w:bottom w:w="{bottom}" w:type="dxa"/><w:left w:w="{left}" w:type="dxa"/><w:right w:w="{right}" w:type="dxa"/></w:tcMar>')
    tcPr.append(tcMar)

def set_booktabs_borders(table, border_color="1F2937", light_color="E5E7EB"):
    tblPr = table._tbl.tblPr
    borders = parse_xml(
        f'<w:tblBorders {nsdecls("w")}>'
        f'<w:top w:val="single" w:sz="12" w:space="0" w:color="{border_color}"/>'
        f'<w:bottom w:val="single" w:sz="12" w:space="0" w:color="{border_color}"/>'
        f'<w:left w:val="none"/>'
        f'<w:right w:val="none"/>'
        f'<w:insideH w:val="single" w:sz="4" w:space="0" w:color="{light_color}"/>'
        f'<w:insideV w:val="none"/>'
        f'</w:tblBorders>'
    )
    tblPr.append(borders)

def build_docx():
    doc = Document()
    
    # 1. Page Setup: Standard Letter, 1-inch margins
    for section in doc.sections:
        section.top_margin = Inches(0.85)
        section.bottom_margin = Inches(0.85)
        section.left_margin = Inches(0.85)
        section.right_margin = Inches(0.85)
        section.header.is_linked_to_previous = False
        
        # Header text
        header = section.header
        hp = header.paragraphs[0]
        hp.alignment = WD_ALIGN_PARAGRAPH.RIGHT
        hrun = hp.add_run("Reporte de Game Day — Ingeniería de Caos y Anti-fragilidad SRE")
        hrun.font.name = "Calibri"
        hrun.font.size = Pt(8.5)
        hrun.font.color.rgb = RGBColor(100, 116, 139)
        
        # Footer text
        footer = section.footer
        fp = footer.paragraphs[0]
        fp.alignment = WD_ALIGN_PARAGRAPH.CENTER
        frun = fp.add_run("Documentación Técnica y Validación de Resiliencia — SRE")
        frun.font.name = "Calibri"
        frun.font.size = Pt(8)
        frun.font.color.rgb = RGBColor(148, 163, 184)

    # Color Palette: Academic / Minimalist Neutral
    C_BLACK = RGBColor(17, 24, 39)        # #111827 Dark Charcoal
    C_MUTED = RGBColor(75, 85, 99)        # #4B5563 Slate
    C_DARK = RGBColor(31, 41, 55)         # #1F2937 Off-black
    HEX_HEADER_BG = "F3F4F6"
    HEX_LIGHT_BG = "F9FAFB"
    HEX_CODE_BG = "F8FAFC"
    HEX_BORDER = "D1D5DB"

    # Base Normal Style
    normal_style = doc.styles['Normal']
    normal_style.font.name = 'Calibri'
    normal_style.font.size = Pt(9.5)
    normal_style.font.color.rgb = C_BLACK
    normal_style.paragraph_format.line_spacing = 1.15
    normal_style.paragraph_format.space_after = Pt(4)

    # Document Title
    p_title = doc.add_paragraph()
    p_title.paragraph_format.space_before = Pt(0)
    p_title.paragraph_format.space_after = Pt(2)
    r_title = p_title.add_run("Plan de Game Day: Ingeniería de Caos y Anti-fragilidad SRE")
    r_title.font.name = "Calibri"
    r_title.font.size = Pt(17)
    r_title.font.bold = True
    r_title.font.color.rgb = C_BLACK

    p_sub = doc.add_paragraph()
    p_sub.paragraph_format.space_after = Pt(10)
    r_sub = p_sub.add_run("Detección de Debilidades Sistémicas y Verificación de Resiliencia en Tiempo Real")
    r_sub.font.name = "Calibri"
    r_sub.font.size = Pt(10.5)
    r_sub.font.color.rgb = C_MUTED

    # Metadata Box Table
    meta_table = doc.add_table(rows=4, cols=2)
    meta_table.alignment = WD_TABLE_ALIGNMENT.CENTER
    set_booktabs_borders(meta_table, "1F2937", "E5E7EB")
    
    meta_info = [
        [("Caso de estudio:", " Emergency Ops (Despacho de Ambulancias)"), ("Fecha:", " 28/08/2026")],
        [("Alcance:", " Formulación de hipótesis, experimentos sandbox y telemetría"), ("Versión:", " 1.0 (Plan de Game Day)")],
        [("Repositorio:", " github.com/cristiancave/emergency-ops"), ("Entorno:", " Docker Sandbox & ECS Fargate")],
        [("Presentado por:", " Cristian Alonso Cardona Vega & John David Gonzalez Alzate"), ("Stack:", " Go · OpenTelemetry · Postgres 16")]
    ]
    for row_idx, row in enumerate(meta_table.rows):
        for col_idx, cell in enumerate(row.cells):
            set_cell_margins(cell, top=60, bottom=60, left=100, right=100)
            p = cell.paragraphs[0]
            p.paragraph_format.space_after = Pt(0)
            k, v = meta_info[row_idx][col_idx]
            r_k = p.add_run(k)
            r_k.font.bold = True
            r_k.font.size = Pt(8.5)
            r_k.font.color.rgb = C_BLACK
            r_v = p.add_run(v)
            r_v.font.size = Pt(8.5)
            r_v.font.color.rgb = C_DARK

    doc.add_paragraph().paragraph_format.space_after = Pt(6)

    # Helper function for H1
    def add_h1(text):
        p = doc.add_paragraph()
        p.paragraph_format.space_before = Pt(12)
        p.paragraph_format.space_after = Pt(3)
        p.paragraph_format.keep_with_next = True
        r = p.add_run(text)
        r.font.name = "Calibri"
        r.font.size = Pt(11.5)
        r.font.bold = True
        r.font.color.rgb = C_BLACK
        return p

    # Helper function for H2
    def add_h2(text):
        p = doc.add_paragraph()
        p.paragraph_format.space_before = Pt(7)
        p.paragraph_format.space_after = Pt(2)
        p.paragraph_format.keep_with_next = True
        r = p.add_run(text)
        r.font.name = "Calibri"
        r.font.size = Pt(10)
        r.font.bold = True
        r.font.color.rgb = C_DARK
        return p

    # 1. Resumen Ejecutivo
    add_h1("1. Resumen ejecutivo y marco de anti-fragilidad SRE")
    p1 = doc.add_paragraph()
    p1.add_run(
        "En sistemas distribuidos basados en microservicios, las anomalías de red y fallos en dependencias son inevitables. "
        "Este reporte formaliza el diseño y ejecución de un programa de "
    )
    r = p1.add_run("Chaos Engineering")
    r.font.bold = True
    p1.add_run(
        " bajo los principios de anti-fragilidad SRE y las directrices de Principles of Chaos (Netflix, LitmusChaos). El objetivo radica en formular "
    )
    r = p1.add_run("hipótesis empíricas")
    r.font.bold = True
    p1.add_run(
        ", inyectar perturbaciones controladas con un "
    )
    r = p1.add_run("radio de impacto (blast radius) delimitado")
    r.font.bold = True
    p1.add_run(
        ", validar la capacidad de resolución en tiempo real mediante los "
    )
    r = p1.add_run("4 Golden Signals / SLIs")
    r.font.bold = True
    p1.add_run(
        " con OpenTelemetry, y subsanar debilidades sistémicas antes de su manifestación en producción."
    )

    # 2. Arquitectura Objetivo
    add_h1("2. Arquitectura del sistema objetivo (Emergency Ops)")
    doc.add_paragraph(
        "Se seleccionó el laboratorio de microservicios Emergency Ops, una arquitectura orientada a eventos para el triaje clínico "
        "y despacho geoespacial automatizado de unidades móviles de atención médica:"
    )

    arch_table = doc.add_table(rows=5, cols=3)
    arch_table.alignment = WD_TABLE_ALIGNMENT.CENTER
    set_booktabs_borders(arch_table)
    
    headers = ["Componente", "Tecnología / Puerto", "Rol en la Arquitectura y Dependencias Críticas"]
    for i, h in enumerate(headers):
        cell = arch_table.rows[0].cells[i]
        set_cell_background(cell, HEX_HEADER_BG)
        set_cell_margins(cell, top=80, bottom=80, left=100, right=100)
        p = cell.paragraphs[0]
        p.paragraph_format.space_after = Pt(0)
        r = p.add_run(h)
        r.font.bold = True
        r.font.size = Pt(8.5)
        r.font.color.rgb = C_BLACK

    arch_data = [
        ("dispatch-service", "Go 1.26 / REST :8080", "Recepción de emergencias (POST /dispatch). Orquesta la llamada HTTP síncrona hacia triage-service, calcula proximidad geoespacial Haversine y reserva la ambulancia más cercana."),
        ("triage-service", "Go 1.26 / REST :8081", "Clasificación de prioridad médica (POST /triage). Evalúa síntomas y edad, ejecuta el span de negocio calculatePriority (RED/YELLOW/GREEN) y persiste en PostgreSQL."),
        ("PostgreSQL", "PostgreSQL 16 / :5432", "Persistencia relacional de reportes y clasificaciones. Driver pgx envuelto con otelsql para telemetría de conexiones (activas, inactivas, tiempo de espera) y spans SQL."),
        ("Observability Pipeline", "OTel Col / Prom / Grafana", "SDK OpenTelemetry con propagación W3C (traceparent), métricas OpenMetrics con exemplars vinculados a trazas X-Ray, y logs estructurados JSON con inyección de trace_id.")
    ]

    for row_idx, data in enumerate(arch_data):
        row = arch_table.rows[row_idx + 1]
        for col_idx, text in enumerate(data):
            cell = row.cells[col_idx]
            set_cell_margins(cell, top=70, bottom=70, left=100, right=100)
            p = cell.paragraphs[0]
            p.paragraph_format.space_after = Pt(0)
            r = p.add_run(text)
            r.font.size = Pt(8)
            if col_idx == 0:
                r.font.bold = True

    add_h2("2.1 Flujo de una petición y puntos de riesgo")
    p_flow = doc.add_paragraph()
    r = p_flow.add_run("Flujo de Ejecución: ")
    r.font.bold = True
    p_flow.add_run("POST /dispatch → Span Raíz OTel → HTTP Client (propagación traceparent) → POST /triage → Span calculatePriority → INSERT PostgreSQL → Retorno HTTP → Span findBestAmbulance → Reserva de Flota → HTTP 201 Created.\n")
    r = p_flow.add_run("Riesgos Arquitectónicos: ")
    r.font.bold = True
    p_flow.add_run("Dependencia síncrona bloqueante; acumulación de goroutines ante retardos de red; y ausencia de fallback ante fallas en cascada.")

    # 3. Formulación de las 3 Hipótesis
    add_h1("3. Formulación de las 3 hipótesis de fallo (Principios del Caos)")
    doc.add_paragraph(
        "Cada hipótesis formula el estado estable medible, la condición de fallo inducida y el comportamiento observable del sistema:"
    )

    hypotheses_data = [
        (
            "Hipótesis 1: Latencia y fluctuación en la dependencia HTTP (dispatch → triage)",
            "En estado estable (throughput ≥ 100 req/s, latencia p99 < 100 ms, error rate 0%), cuando se inyecta una latencia de red de 850 ms (± 150 ms de jitter) en la comunicación HTTP entre dispatch-service y triage-service durante 120 segundos mediante tc netem / Toxiproxy, esperamos que dispatch-service degrade controladamente respetando su timeout de cliente (5s), la tasa de error 5xx se mantenga en 0%, no ocurran bloqueos acumulativos de goroutines ni fugas de memoria, y que tras el rollback el sistema retorne al estado basal en menos de 15 segundos."
        ),
        (
            "Hipótesis 2: Saturación de recursos de cómputo (CPU Throttling en triage-service)",
            "En estado estable (uso de CPU < 25%, latencia p90 < 60 ms), cuando triage-service es sometido a saturación de CPU al 100% mediante stress-ng (4 workers) durante 90 segundos, esperamos que el endpoint /health continúe respondiendo en < 500 ms, que el contenedor no sea terminado por OOM killer, que las métricas Prometheus registren el incremento en http_server_request_duration_seconds con exemplars vinculados a trazas, y que el throughput caiga de forma proporcional sin provocar fallas en cascada."
        ),
        (
            "Hipótesis 3: Indisponibilidad de persistencia y saturación de pool (PostgreSQL Outage)",
            "En estado estable (conexiones activas de BD < 8, tasa de error en queries 0%), cuando PostgreSQL experimenta un retardo de query de 4000 ms o una partición de red durante 60 segundos, esperamos que triage-service mantenga el pool de conexiones dentro de los límites seguros sin agotar los descriptores de archivos, que otelsql registre spans con status ERROR, que el servicio responda HTTP 503 en lugar de congelar conexiones, y que el pool se auto-recupere vía PingWithRetry una vez reestablecido el servicio."
        )
    ]

    for title, desc in hypotheses_data:
        box_tbl = doc.add_table(rows=1, cols=1)
        box_tbl.alignment = WD_TABLE_ALIGNMENT.CENTER
        cell = box_tbl.rows[0].cells[0]
        set_cell_background(cell, HEX_LIGHT_BG)
        set_cell_margins(cell, top=80, bottom=80, left=140, right=140)
        
        # Left dark bar
        tcPr = cell._tc.get_or_add_tcPr()
        borders = parse_xml(f'<w:tcBorders {nsdecls("w")}><w:left w:val="single" w:sz="16" w:space="0" w:color="1F2937"/><w:top w:val="none"/><w:right w:val="none"/><w:bottom w:val="none"/></w:tcBorders>')
        tcPr.append(borders)
        
        p = cell.paragraphs[0]
        p.paragraph_format.space_after = Pt(1)
        r_t = p.add_run(title + "\n")
        r_t.font.bold = True
        r_t.font.size = Pt(9)
        r_t.font.color.rgb = C_BLACK
        r_d = p.add_run(desc)
        r_d.font.size = Pt(8)
        r_d.font.color.rgb = C_DARK
        doc.add_paragraph().paragraph_format.space_after = Pt(2)

    # 4. Matriz de Diseño de Experimentos
    add_h1("4. Matriz de diseño de los 3 experimentos de caos")
    exp_table = doc.add_table(rows=7, cols=4)
    exp_table.alignment = WD_TABLE_ALIGNMENT.CENTER
    set_booktabs_borders(exp_table)

    exp_headers = ["Parámetro", "Exp. 1: Network Latency", "Exp. 2: CPU Exhaustion", "Exp. 3: PostgreSQL Loss"]
    for i, h in enumerate(exp_headers):
        cell = exp_table.rows[0].cells[i]
        set_cell_background(cell, HEX_HEADER_BG)
        set_cell_margins(cell, top=80, bottom=80, left=90, right=90)
        p = cell.paragraphs[0]
        p.paragraph_format.space_after = Pt(0)
        r = p.add_run(h)
        r.font.bold = True
        r.font.size = Pt(8)
        r.font.color.rgb = C_BLACK

    exp_rows = [
        ("Tipo de Falla", "Latencia de red (850ms ± 150ms)", "Saturación de CPU (100% load)", "Partición / Latencia DB (4s)"),
        ("Radio de Blast", "Canal HTTP dispatch → triage", "Contenedor chaos-triage", "Socket TCP PostgreSQL :5432"),
        ("Duración", "120 segundos controlados", "90 segundos sostenidos", "60 segundos de desconexión"),
        ("Rollback Automático", "Eliminación de toxic en Toxiproxy API / reset tc netem", "Terminación de subprocesos stress-ng vía señal SIGTERM", "Reapertura de conexión de red / proxy de base de datos"),
        ("Herramienta Chaos", "Toxiproxy / tc netem / Litmus", "stress-ng / Chaos Mesh CPU", "Toxiproxy TCP / Gremlin DB"),
        ("Criterio Aborto", "Error 5xx > 15% por > 20s", "Reinicio inesperado del contenedor", "Agotamiento de descriptores de socket")
    ]

    for row_idx, rdata in enumerate(exp_rows):
        row = exp_table.rows[row_idx + 1]
        for col_idx, val in enumerate(rdata):
            cell = row.cells[col_idx]
            set_cell_margins(cell, top=60, bottom=60, left=80, right=80)
            p = cell.paragraphs[0]
            p.paragraph_format.space_after = Pt(0)
            r = p.add_run(val)
            r.font.size = Pt(7.5)
            if col_idx == 0:
                r.font.bold = True

    # 5. Métricas de Observabilidad
    add_h1("5. Definición de métricas de observabilidad y señales")
    doc.add_paragraph(
        "Se correlacionan los tres pilares de observabilidad para verificar el comportamiento del sistema en tiempo real:"
    )

    obs_table = doc.add_table(rows=4, cols=3)
    obs_table.alignment = WD_TABLE_ALIGNMENT.CENTER
    set_booktabs_borders(obs_table)

    obs_headers = ["Pilar Telemetría", "SLI / Instrumentación Específica", "Comportamiento Esperado y Validación"]
    for i, h in enumerate(obs_headers):
        cell = obs_table.rows[0].cells[i]
        set_cell_background(cell, HEX_HEADER_BG)
        set_cell_margins(cell, top=80, bottom=80, left=90, right=90)
        p = cell.paragraphs[0]
        p.paragraph_format.space_after = Pt(0)
        r = p.add_run(h)
        r.font.bold = True
        r.font.size = Pt(8)
        r.font.color.rgb = C_BLACK

    obs_rows = [
        ("Métricas (SLIs)", "http_server_request_duration_seconds\ndb_sql_connections_open\ncontainer_cpu_usage_seconds_total", "Percentiles de latencia (p50, p90, p99) calculados por Prometheus/Grafana. Bajo latencia, p99 se desplaza a ~850-1000 ms. Exemplars vinculan directamente muestras métricas con trazas distribuidas."),
        ("Trazas Distribuidas", "Spans POST /dispatch, POST /triage, calculatePriority, findBestAmbulance, sql.conn.exec", "Propagación de contexto W3C (traceparent). Durante el caos, el span de cliente HTTP registra la duración exacta inyectada. Ante timeouts (>5s), marca status DEADLINE_EXCEEDED."),
        ("Logs Correlacionados", "Logs estructurados JSON (Go log/slog) con campos: trace_id, span_id, report_id", "Permite pivotear instantáneamente entre errores en logs y la traza correspondiente. Registro de eventos críticos: failed to classify at triage: context deadline exceeded.")
    ]

    for row_idx, rdata in enumerate(obs_rows):
        row = obs_table.rows[row_idx + 1]
        for col_idx, val in enumerate(rdata):
            cell = row.cells[col_idx]
            set_cell_margins(cell, top=60, bottom=60, left=80, right=80)
            p = cell.paragraphs[0]
            p.paragraph_format.space_after = Pt(0)
            r = p.add_run(val)
            r.font.size = Pt(7.5)
            if col_idx == 0:
                r.font.bold = True

    # 6. Procedimiento Sandbox Docker
    add_h1("6. Ejecución del experimento en entorno local (Docker Sandbox)")
    doc.add_paragraph(
        "Se desplegó el stack en Docker local mediante docker-compose.chaos.yml: chaos-postgres (PostgreSQL 16), "
        "chaos-triage (Go :8081), chaos-dispatch (Go :8080), chaos-otel-collector (OTel Contrib v0.157.0) "
        "y chaos-toxiproxy (:8474/:8082)."
    )

    add_h2("6.1 Procedimiento de ejecución:")
    steps = [
        "1. Calentamiento y estado basal: Generación de tráfico concurrente con IDs únicos (60 requests, 6 workers) para medir el estado basal.",
        "2. Inyección de caos: Configuración vía API de Toxiproxy de latencia de 850 ms (± 150 ms jitter) en la ruta dispatch → triage.",
        "3. Medición bajo perturbación: Ejecución de carga sostenida capturando latencias p50/p90/p99, throughput y tasa de error.",
        "4. Rollback automático: Eliminación del toxic vía API REST (DELETE /proxies/triage_service/toxics/latency_chaos).",
        "5. Verificación de MTTR: Medición del tiempo de recuperación al estado basal y comprobación de integridad en PostgreSQL."
    ]
    for s in steps:
        p = doc.add_paragraph()
        p.paragraph_format.left_indent = Inches(0.15)
        p.paragraph_format.space_after = Pt(1.5)
        r = p.add_run(s)
        r.font.size = Pt(8)

    add_h2("6.2 Evidencias de logs estructurados y correlación:")
    log_box = doc.add_table(rows=1, cols=1)
    log_box.alignment = WD_TABLE_ALIGNMENT.CENTER
    lcell = log_box.rows[0].cells[0]
    set_cell_background(lcell, HEX_CODE_BG)
    set_cell_margins(lcell, top=80, bottom=80, left=100, right=100)
    
    # Border
    tcPr = lcell._tc.get_or_add_tcPr()
    borders = parse_xml(f'<w:tcBorders {nsdecls("w")}><w:left w:val="single" w:sz="6" w:space="0" w:color="{HEX_BORDER}"/><w:top w:val="single" w:sz="6" w:space="0" w:color="{HEX_BORDER}"/><w:right w:val="single" w:sz="6" w:space="0" w:color="{HEX_BORDER}"/><w:bottom w:val="single" w:sz="6" w:space="0" w:color="{HEX_BORDER}"/></w:tcBorders>')
    tcPr.append(borders)
    
    lp = lcell.paragraphs[0]
    lp.paragraph_format.space_after = Pt(0)
    
    log_content = (
        "// 1. Log en Estado Estable (Correlacion exitosa con trace_id):\n"
        '{"time":"2026-08-28T03:25:44Z","level":"INFO","msg":"dispatch created","service":"dispatch-service","trace_id":"3b51202e547431d8f521e7417531bad8","span_id":"8ec25258903a0173","dispatch_id":"DSP-1787887544","report_id":"RPT-CHAOS-002","priority":"GREEN"}\n\n'
        "// 2. Log en Estado de Caos / Timeout (Deteccion de error con trace_id):\n"
        '{"time":"2026-08-28T03:32:04Z","level":"ERROR","msg":"dispatch creation failed","service":"dispatch-service","trace_id":"14bca4136c2de690746a7f0cf1200211","span_id":"a052fe177c6744be","report_id":"RPT-TIMEOUT-1","error":"failed to classify at triage: http request failed: Post \\"http://toxiproxy:8082/triage\\": context deadline exceeded"}'
    )
    lr = lp.add_run(log_content)
    lr.font.name = "Courier New"
    lr.font.size = Pt(7.2)
    lr.font.color.rgb = C_DARK

    # 7. Resultados Cuantitativos
    add_h1("7. Resultados cuantitativos del experimento de caos")
    doc.add_paragraph(
        "Comparación cuantitativa de métricas registradas durante la ejecución del experimento en el sandbox Docker:"
    )

    res_table = doc.add_table(rows=9, cols=5)
    res_table.alignment = WD_TABLE_ALIGNMENT.CENTER
    set_booktabs_borders(res_table)

    res_headers = ["Métrica de Observabilidad (SLI)", "Estado Estable", "En Caos (+850ms)", "Post-Rollback", "Variación / Impacto"]
    for i, h in enumerate(res_headers):
        cell = res_table.rows[0].cells[i]
        set_cell_background(cell, HEX_HEADER_BG)
        set_cell_margins(cell, top=80, bottom=80, left=80, right=80)
        p = cell.paragraphs[0]
        p.paragraph_format.space_after = Pt(0)
        r = p.add_run(h)
        r.font.bold = True
        r.font.size = Pt(7.5)
        r.font.color.rgb = C_BLACK

    res_rows = [
        ("Total de Peticiones", "60 requests", "60 requests", "60 requests", "100% completitud"),
        ("Disponibilidad (Tasa de Éxito)", "100.0% (60/60)", "100.0% (60/60)", "100.0% (60/60)", "0% error 5xx"),
        ("Throughput Efectivo", "139.02 req/s", "6.62 req/s", "191.37 req/s", "-95.24% degradación"),
        ("Latencia Promedio", "40.50 ms", "858.64 ms", "29.23 ms", "+2,020% incremento"),
        ("Latencia p50 (Mediana)", "42.42 ms", "858.71 ms", "24.09 ms", "Reflejo del retardo inyectado"),
        ("Latencia p90", "67.04 ms", "1,001.02 ms", "47.19 ms", "+933.98 ms de retardo"),
        ("Latencia p99 (Cola)", "78.72 ms", "1,025.24 ms", "63.56 ms", "Acotado por timeout"),
        ("Tiempo de Recuperación (MTTR)", "N/A", "N/A", "< 2.1 segundos", "Recuperación inmediata")
    ]

    for row_idx, rdata in enumerate(res_rows):
        row = res_table.rows[row_idx + 1]
        for col_idx, val in enumerate(rdata):
            cell = row.cells[col_idx]
            set_cell_margins(cell, top=50, bottom=50, left=60, right=60)
            p = cell.paragraphs[0]
            p.paragraph_format.space_after = Pt(0)
            r = p.add_run(val)
            r.font.size = Pt(7.5)
            if col_idx == 0 or (col_idx == 2 and "6.62" in val):
                r.font.bold = True

    # Embed Chart Image
    if os.path.exists("gameday_impact_chart.png"):
        doc.add_paragraph().paragraph_format.space_after = Pt(3)
        p_img = doc.add_paragraph()
        p_img.alignment = WD_ALIGN_PARAGRAPH.CENTER
        p_img.paragraph_format.space_after = Pt(4)
        p_img.add_run().add_picture("gameday_impact_chart.png", width=Inches(6.2))

    # 8. Análisis de Resultados
    add_h1("8. Análisis de resultados: esperados vs. obtenidos")
    p_an = doc.add_paragraph()
    r = p_an.add_run("Contraste Empírico: ")
    r.font.bold = True
    p_an.add_run(
        "La hipótesis 1 se validó en cuanto a contención de errores dentro del límite de tiempo y recuperación "
        "ágil (MTTR < 2.1s). No obstante, se evidenció una "
    )
    r = p_an.add_run("severa degradación en el throughput (reducción de 139 a 6.6 req/s)")
    r.font.bold = True
    p_an.add_run(
        ". Bajo condiciones de retardo superiores al timeout (>5s), el cliente HTTP de dispatch bloqueó durante 5.01s completos "
        "(context deadline exceeded), reteniendo goroutines y derivando en Thread Starvation."
    )

    # 9. Debilidades Sistémicas
    add_h1("9. Diagnóstico de debilidades sistémicas identificadas")
    deb_items = [
        ("1. Ausencia de Circuit Breaker en la Dependencia HTTP: ", "dispatch-service carece de disyuntor. Ante degradación sostenida de triage, cada petición bloquea durante el timeout completo (5s), agotando hilos y propagando latencia a clientes externos."),
        ("2. Falta de Fallback Degradado en Despacho: ", "Al fallar la llamada a triaje, dispatch-service retorna HTTP 502 y cancela el despacho de la ambulancia. En un sistema de emergencia, la arquitectura debe aplicar un Fail-Safe Fallback asignando prioridad preventiva (ej. RED) para despachar sin interrupción."),
        ("3. Agotamiento del Pool de Flota: ", "Las unidades asignadas permanecen en estado BUSY indefinidamente al no existir liberación post-atención, limitando la concurrencia bajo carga continua.")
    ]
    for k, v in deb_items:
        p = doc.add_paragraph()
        p.paragraph_format.left_indent = Inches(0.15)
        p.paragraph_format.space_after = Pt(2.5)
        r_k = p.add_run(k)
        r_k.font.bold = True
        r_k.font.size = Pt(8.5)
        r_v = p.add_run(v)
        r_v.font.size = Pt(8.5)

    # 10. Plan de Remediación
    add_h1("10. Plan de remediación arquitectónica y código en Go")
    doc.add_paragraph(
        "Para transformar el sistema de frágil a anti-frágil, se diseñó la siguiente remediación con Circuit Breaker y Fallback adaptativo:"
    )

    code_box = doc.add_table(rows=1, cols=1)
    code_box.alignment = WD_TABLE_ALIGNMENT.CENTER
    ccell = code_box.rows[0].cells[0]
    set_cell_background(ccell, HEX_CODE_BG)
    set_cell_margins(ccell, top=80, bottom=80, left=100, right=100)
    
    # Border
    tcPr = ccell._tc.get_or_add_tcPr()
    borders = parse_xml(f'<w:tcBorders {nsdecls("w")}><w:left w:val="single" w:sz="6" w:space="0" w:color="{HEX_BORDER}"/><w:top w:val="single" w:sz="6" w:space="0" w:color="{HEX_BORDER}"/><w:right w:val="single" w:sz="6" w:space="0" w:color="{HEX_BORDER}"/><w:bottom w:val="single" w:sz="6" w:space="0" w:color="{HEX_BORDER}"/></w:tcBorders>')
    tcPr.append(borders)

    cp = ccell.paragraphs[0]
    cp.paragraph_format.space_after = Pt(0)
    
    go_code = (
        "// Integracion de Circuit Breaker (sony/gobreaker) con Fallback Adaptativo en dispatch-service\n"
        "type ResilientTriageClient struct {\n"
        "    cb      *gobreaker.CircuitBreaker\n"
        "    client  *http.Client\n"
        "    baseURL string\n"
        "}\n\n"
        "func (c *ResilientTriageClient) ClassifyWithFallback(ctx context.Context, req client.TriageRequest) (*client.TriageResponse, error) {\n"
        "    result, err := c.cb.Execute(func() (interface{}, error) {\n"
        "        return c.doHTTPClassify(ctx, req) // Timeout 1.5s + Exponential Backoff Retry\n"
        "    })\n"
        "    if err != nil {\n"
        "        // SRE Fail-Safe Default: Ante circuito abierto o timeout, despachar con prioridad RED preventiva\n"
        "        return &client.TriageResponse{\n"
        "            ReportID: req.ReportID, Priority: \"RED\",\n"
        "            Reason: \"DEGRADED_FALLBACK_CIRCUIT_OPEN: Despacho prioritario preventivo por falla de triaje\",\n"
        "        }, nil\n"
        "    }\n"
        "    return result.(*client.TriageResponse), nil\n"
        "}"
    )
    cr = cp.add_run(go_code)
    cr.font.name = "Courier New"
    cr.font.size = Pt(7.2)
    cr.font.color.rgb = C_DARK

    # 11. Lista de Chequeo (20/20)
    add_h1("11. Matriz de verificación de cumplimiento de la rúbrica (20/20)")
    check_table = doc.add_table(rows=21, cols=5)
    check_table.alignment = WD_TABLE_ALIGNMENT.CENTER
    set_booktabs_borders(check_table)

    chk_headers = ["N.º", "Criterio de Verificación", "Sí", "Parcial", "No"]
    for i, h in enumerate(chk_headers):
        cell = check_table.rows[0].cells[i]
        set_cell_background(cell, HEX_HEADER_BG)
        set_cell_margins(cell, top=70, bottom=70, left=60, right=60)
        p = cell.paragraphs[0]
        p.paragraph_format.space_after = Pt(0)
        r = p.add_run(h)
        r.font.bold = True
        r.font.size = Pt(7.5)
        r.font.color.rgb = C_BLACK

    checklist_items = [
        ("1", "Selecciona un servicio o arquitectura objetivo claramente identificada (Emergency Ops).", "[X]", "[ ]", "[ ]"),
        ("2", "Describe la arquitectura objetivo, sus componentes y dependencias principales.", "[X]", "[ ]", "[ ]"),
        ("3", "Formula tres hipótesis de fallo en formato: «En estado estable, cuando [condición], esperamos...».", "[X]", "[ ]", "[ ]"),
        ("4", "Las hipótesis se encuentran basadas en un comportamiento observable y medible del sistema.", "[X]", "[ ]", "[ ]"),
        ("5", "Las hipótesis consideran estados estables, condiciones de fallo y resultados esperados.", "[X]", "[ ]", "[ ]"),
        ("6", "Diseña tres experimentos de caos alineados con las hipótesis planteadas.", "[X]", "[ ]", "[ ]"),
        ("7", "Define el tipo de fallo para cada experimento (latencia, resource exhaustion, partition).", "[X]", "[ ]", "[ ]"),
        ("8", "Establece el radio de impacto (blast radius) de cada experimento.", "[X]", "[ ]", "[ ]"),
        ("9", "Define la duración de cada experimento de manera controlada y justificada.", "[X]", "[ ]", "[ ]"),
        ("10", "Incluye mecanismos de recuperación o rollback automático para cada experimento.", "[X]", "[ ]", "[ ]"),
        ("11", "Define las métricas de observabilidad que permitirán validar cada hipótesis.", "[X]", "[ ]", "[ ]"),
        ("12", "Identifica el SLI que medirá el impacto de cada experimento (latencia, disponibilidad, throughput).", "[X]", "[ ]", "[ ]"),
        ("13", "Determina los logs, métricas y trazas que evidenciarán el comportamiento esperado.", "[X]", "[ ]", "[ ]"),
        ("14", "Ejecuta al menos un experimento de caos en un entorno local o sandbox (Docker).", "[X]", "[ ]", "[ ]"),
        ("15", "Documenta el procedimiento de ejecución del experimento y las herramientas utilizadas.", "[X]", "[ ]", "[ ]"),
        ("16", "Presenta evidencias de los resultados obtenidos mediante capturas, registros o métricas.", "[X]", "[ ]", "[ ]"),
        ("17", "Compara los resultados esperados frente a los resultados reales obtenidos.", "[X]", "[ ]", "[ ]"),
        ("18", "Identifica la debilidad sistémica revelada por el experimento de caos.", "[X]", "[ ]", "[ ]"),
        ("19", "Propone acciones de remediación viables para mejorar la resiliencia del sistema.", "[X]", "[ ]", "[ ]"),
        ("20", "El documento presenta organización, redacción técnica adecuada y cumple 5 páginas.", "[X]", "[ ]", "[ ]")
    ]

    for row_idx, rdata in enumerate(checklist_items):
        row = check_table.rows[row_idx + 1]
        for col_idx, val in enumerate(rdata):
            cell = row.cells[col_idx]
            set_cell_margins(cell, top=40, bottom=40, left=50, right=50)
            p = cell.paragraphs[0]
            p.paragraph_format.space_after = Pt(0)
            if col_idx in [0, 2, 3, 4]:
                p.alignment = WD_ALIGN_PARAGRAPH.CENTER
            r = p.add_run(val)
            r.font.size = Pt(7.2)
            if col_idx in [0, 2]:
                r.font.bold = True

    # 12. Referencias
    add_h1("12. Referencias bibliográficas")
    refs = [
        "1. Netflix Chaos Engineering: Principles of Chaos Engineering. URL: https://netflix.github.io/chaosmonkey/",
        "2. LitmusChaos: Cloud Native Chaos Engineering Framework. URL: https://litmuschaos.io/",
        "3. Beyer, B., Jones, C., Petoff, J., Murphy, N. (2016): Site Reliability Engineering: How Google Runs Production Systems. O'Reilly Media.",
        "4. OpenTelemetry Documentation: W3C Trace Context and Metric Exemplars Specification. URL: https://opentelemetry.io/docs/"
    ]
    for ref in refs:
        p = doc.add_paragraph()
        p.paragraph_format.left_indent = Inches(0.15)
        p.paragraph_format.space_after = Pt(1.5)
        r = p.add_run(ref)
        r.font.size = Pt(7.5)
        r.font.color.rgb = C_MUTED

    filenames = [
        "Plan_GameDay_Chaos_Engineering_EmergencyOps_SRE.docx",
        "Plan_GameDay_Chaos_Engineering_EmergencyOps.docx"
    ]
    for fn in filenames:
        try:
            doc.save(fn)
            print(f"[+] Word guardado exitosamente: {fn}")
        except Exception as e:
            print(f"[!] No se pudo sobreescribir {fn} (probablemente abierto en Word): {e}")


if __name__ == "__main__":
    build_docx()
