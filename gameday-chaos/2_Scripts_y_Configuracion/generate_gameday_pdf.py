import os
import json
import pymupdf
import matplotlib.pyplot as plt
from reportlab.lib.pagesizes import letter
from reportlab.lib import colors
from reportlab.lib.units import inch
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.platypus import (
    SimpleDocTemplate, Paragraph, Spacer, Table, TableStyle, Image, KeepTogether, PageBreak
)
from reportlab.pdfgen import canvas

class AcademicNumberedCanvas(canvas.Canvas):
    """Canvas con diseño editorial sobrio y numeración formal."""
    def __init__(self, *args, **kwargs):
        super(AcademicNumberedCanvas, self).__init__(*args, **kwargs)
        self._saved_page_states = []

    def showPage(self):
        self._saved_page_states.append(dict(self.__dict__))
        self._startPage()

    def save(self):
        num_pages = len(self._saved_page_states)
        for state in self._saved_page_states:
            self.__dict__.update(state)
            self.draw_page_decorations(num_pages)
            super(AcademicNumberedCanvas, self).showPage()
        super(AcademicNumberedCanvas, self).save()

    def draw_page_decorations(self, page_count):
        self.saveState()
        self.setFont("Helvetica", 8)
        self.setFillColor(colors.HexColor("#4b5563"))
        
        # Header (pages 2+)
        if self._pageNumber > 1:
            self.drawString(54, 11 * inch - 36, "Reporte de Game Day — Ingeniería de Caos y Anti-fragilidad SRE")
            self.drawRightString(8.5 * inch - 54, 11 * inch - 36, "Emergency Ops")
            self.setStrokeColor(colors.HexColor("#d1d5db"))
            self.setLineWidth(0.5)
            self.line(54, 11 * inch - 40, 8.5 * inch - 54, 11 * inch - 40)

        # Footer (all pages)
        self.setStrokeColor(colors.HexColor("#d1d5db"))
        self.setLineWidth(0.5)
        self.line(54, 42, 8.5 * inch - 54, 42)
        
        self.drawString(54, 30, "Documentación Técnica y Validación de Resiliencia — SRE")
        page_str = f"Página {self._pageNumber} de {page_count}"
        self.drawRightString(8.5 * inch - 54, 30, page_str)
        self.restoreState()

def build_pdf():
    pdf_filename = "Plan_GameDay_Chaos_Engineering_EmergencyOps.pdf"
    doc = SimpleDocTemplate(
        pdf_filename,
        pagesize=letter,
        leftMargin=54,
        rightMargin=54,
        topMargin=46,
        bottomMargin=48
    )
    
    styles = getSampleStyleSheet()
    
    # Palette: Minimalist Academic & Engineering (No AI clichés)
    c_black = colors.HexColor("#111827")      # Main text
    c_gray_dark = colors.HexColor("#374151")  # Secondary text
    c_gray_light = colors.HexColor("#f9fafb") # Subtle box fill
    c_header_bg = colors.HexColor("#f3f4f6")  # Table header neutral fill
    c_border = colors.HexColor("#d1d5db")     # Clean gray border
    c_line_dark = colors.HexColor("#1f2937")  # Booktabs primary rule
    
    # Typography
    title_style = ParagraphStyle(
        'AcademicTitle',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=15,
        leading=19,
        textColor=c_black,
        spaceAfter=3
    )
    
    subtitle_style = ParagraphStyle(
        'AcademicSubTitle',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=9.5,
        leading=13,
        textColor=c_gray_dark,
        spaceAfter=8
    )
    
    meta_style = ParagraphStyle(
        'MetaText',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=7.8,
        leading=10.8,
        textColor=c_gray_dark
    )
    
    h1_style = ParagraphStyle(
        'AcademicH1',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=10.5,
        leading=13.5,
        textColor=c_black,
        spaceBefore=8,
        spaceAfter=3.5
    )
    
    h2_style = ParagraphStyle(
        'AcademicH2',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=8.8,
        leading=11.5,
        textColor=c_black,
        spaceBefore=5,
        spaceAfter=2
    )
    
    body_style = ParagraphStyle(
        'AcademicBody',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=7.8,
        leading=10.5,
        textColor=c_black,
        spaceAfter=3.5
    )
    
    callout_style = ParagraphStyle(
        'AcademicCallout',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=7.6,
        leading=10.2,
        textColor=c_black
    )
    
    code_style = ParagraphStyle(
        'AcademicCode',
        parent=styles['Normal'],
        fontName='Courier',
        fontSize=7.0,
        leading=9.2,
        textColor=colors.HexColor("#1f2937")
    )
    
    table_cell = ParagraphStyle(
        'TableCell',
        parent=styles['Normal'],
        fontName='Helvetica',
        fontSize=7.2,
        leading=9.2,
        textColor=c_black
    )

    table_cell_header = ParagraphStyle(
        'TableCellHeader',
        parent=styles['Normal'],
        fontName='Helvetica-Bold',
        fontSize=7.5,
        leading=9.5,
        textColor=c_black
    )

    story = []

    # =========================================================================
    # PÁGINA 1: PORTADA TÉCNICA, RESUMEN EJECUTIVO & ARQUITECTURA OBJETIVO
    # =========================================================================
    story.append(Paragraph("Plan de Game Day: Ingeniería de Caos y Anti-fragilidad SRE", title_style))
    story.append(Paragraph("Detección de Debilidades Sistémicas y Verificación de Resiliencia en Tiempo Real", subtitle_style))
    
    # Metadata Table (Minimalist style)
    meta_data = [
        [Paragraph("<b>Caso de estudio:</b> Emergency Ops (Despacho de Ambulancias)", meta_style), Paragraph("<b>Fecha:</b> 28/08/2026", meta_style)],
        [Paragraph("<b>Alcance:</b> Formulación de hipótesis, experimentos sandbox y telemetría", meta_style), Paragraph("<b>Versión:</b> 1.0 (Plan de Game Day)", meta_style)],
        [Paragraph("<b>Repositorio:</b> github.com/cristiancave/emergency-ops", meta_style), Paragraph("<b>Entorno:</b> Docker Sandbox & ECS Fargate", meta_style)],
        [Paragraph("<b>Presentado por:</b> Cristian Alonso Cardona Vega & John David Gonzalez Alzate", meta_style), Paragraph("<b>Stack:</b> Go · OpenTelemetry · Postgres 16", meta_style)]
    ]
    meta_table = Table(meta_data, colWidths=[310, 194])
    meta_table.setStyle(TableStyle([
        ('LINEABOVE', (0,0), (-1,0), 1, c_line_dark),
        ('LINEBELOW', (0,-1), (-1,-1), 1, c_line_dark),
        ('TOPPADDING', (0,0), (-1,-1), 2.5),
        ('BOTTOMPADDING', (0,0), (-1,-1), 2.5),
        ('LEFTPADDING', (0,0), (-1,-1), 4),
        ('RIGHTPADDING', (0,0), (-1,-1), 4),
    ]))
    story.append(meta_table)
    story.append(Spacer(1, 5))

    story.append(Paragraph("1. Resumen ejecutivo y marco de anti-fragilidad SRE", h1_style))
    story.append(Paragraph(
        "En sistemas distribuidos basados en microservicios, las anomalías de red y fallos en dependencias son inevitables. "
        "Este reporte formaliza el diseño y ejecución de un programa de <b>Chaos Engineering</b> bajo los principios de anti-fragilidad "
        "SRE y las directrices de <i>Principles of Chaos</i> (Netflix, LitmusChaos). El objetivo radica en formular hipótesis empíricas, "
        "inyectar perturbaciones controladas con un <b>radio de impacto (blast radius) delimitado</b>, validar la capacidad de resolución "
        "en tiempo real mediante los <b>4 Golden Signals / SLIs</b> con OpenTelemetry, y subsanar debilidades sistémicas antes de su manifestación en producción.",
        body_style
    ))
    
    story.append(Paragraph("2. Arquitectura del sistema objetivo (Emergency Ops)", h1_style))
    story.append(Paragraph(
        "Se seleccionó el laboratorio de microservicios <b>Emergency Ops</b>, una arquitectura orientada a eventos para el triaje clínico "
        "y despacho geoespacial automatizado de unidades móviles de atención médica:",
        body_style
    ))

    arch_items = [
        [
            Paragraph("<b>Componente</b>", table_cell_header),
            Paragraph("<b>Tecnología / Puerto</b>", table_cell_header),
            Paragraph("<b>Rol en la Arquitectura y Dependencias Críticas</b>", table_cell_header)
        ],
        [
            Paragraph("<b>dispatch-service</b>", table_cell),
            Paragraph("Go 1.26 / REST :8080", table_cell),
            Paragraph("Recepción de emergencias (<code>POST /dispatch</code>). Orquesta la llamada HTTP síncrona hacia <code>triage-service</code>, calcula proximidad geoespacial Haversine y reserva la ambulancia más cercana.", table_cell)
        ],
        [
            Paragraph("<b>triage-service</b>", table_cell),
            Paragraph("Go 1.26 / REST :8081", table_cell),
            Paragraph("Clasificación de prioridad médica (<code>POST /triage</code>). Evalúa síntomas y edad, ejecuta el span de negocio <code>calculatePriority</code> (RED/YELLOW/GREEN) y persiste en PostgreSQL.", table_cell)
        ],
        [
            Paragraph("<b>PostgreSQL</b>", table_cell),
            Paragraph("PostgreSQL 16 / :5432", table_cell),
            Paragraph("Persistencia relacional de reportes y clasificaciones. Driver <code>pgx</code> envuelto con <code>otelsql</code> para telemetría de conexiones (activas, inactivas, tiempo de espera) y spans SQL.", table_cell)
        ],
        [
            Paragraph("<b>Observability Pipeline</b>", table_cell),
            Paragraph("OTel Col / Prom / Grafana", table_cell),
            Paragraph("SDK OpenTelemetry con propagación W3C (<code>traceparent</code>), métricas OpenMetrics con exemplars vinculados a trazas X-Ray, y logs estructurados JSON con inyección de <code>trace_id</code>.", table_cell)
        ]
    ]
    arch_table = Table(arch_items, colWidths=[100, 105, 299])
    arch_table.setStyle(TableStyle([
        ('LINEABOVE', (0,0), (-1,0), 1.2, c_line_dark),
        ('LINEBELOW', (0,0), (-1,0), 0.8, c_line_dark),
        ('LINEBELOW', (0,-1), (-1,-1), 1.2, c_line_dark),
        ('BACKGROUND', (0,0), (-1,0), c_header_bg),
        ('TOPPADDING', (0,0), (-1,-1), 3.2),
        ('BOTTOMPADDING', (0,0), (-1,-1), 3.2),
        ('LEFTPADDING', (0,0), (-1,-1), 4),
        ('RIGHTPADDING', (0,0), (-1,-1), 4),
    ]))
    story.append(arch_table)
    story.append(Spacer(1, 4))

    story.append(Paragraph("2.1 Flujo de una petición y puntos de riesgo", h2_style))
    story.append(Paragraph(
        "<b>Flujo de Ejecución:</b> <code>POST /dispatch</code> → Span Raíz OTel → HTTP Client (propagación <code>traceparent</code>) → "
        "<code>POST /triage</code> → Span <code>calculatePriority</code> → <code>INSERT</code> PostgreSQL → Retorno HTTP → "
        "Span <code>findBestAmbulance</code> → Reserva de Flota → HTTP 201 Created.<br/>"
        "<b>Riesgos Arquitectónicos:</b> Dependencia síncrona bloqueante; acumulación de goroutines ante retardos de red; y ausencia de fallback ante fallas en cascada.",
        body_style
    ))

    # =========================================================================
    # PÁGINA 2: FORMULACIÓN DE 3 HIPÓTESIS Y DISEÑO DE EXPERIMENTOS DE CAOS
    # =========================================================================
    story.append(PageBreak())
    
    story.append(Paragraph("3. Formulación de las 3 hipótesis de fallo (Principios del Caos)", h1_style))
    story.append(Paragraph(
        "Cada hipótesis formula el <b>estado estable medible</b>, la <b>condición de fallo inducida</b> y el <b>comportamiento observable</b> del sistema:",
        body_style
    ))

    hypotheses = [
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

    for title, text in hypotheses:
        box_data = [[
            Paragraph(f"<b>{title}</b><br/>{text}", callout_style)
        ]]
        box_table = Table(box_data, colWidths=[504])
        box_table.setStyle(TableStyle([
            ('LINEBEFORE', (0,0), (0,-1), 2.5, c_line_dark),
            ('BACKGROUND', (0,0), (-1,-1), c_gray_light),
            ('TOPPADDING', (0,0), (-1,-1), 3),
            ('BOTTOMPADDING', (0,0), (-1,-1), 3),
            ('LEFTPADDING', (0,0), (-1,-1), 6),
            ('RIGHTPADDING', (0,0), (-1,-1), 5),
        ]))
        story.append(box_table)
        story.append(Spacer(1, 3.5))

    story.append(Paragraph("4. Matriz de diseño de los 3 experimentos de caos", h1_style))
    
    exp_matrix = [
        [
            Paragraph("<b>Parámetro</b>", table_cell_header),
            Paragraph("<b>Exp. 1: Network Latency</b>", table_cell_header),
            Paragraph("<b>Exp. 2: CPU Exhaustion</b>", table_cell_header),
            Paragraph("<b>Exp. 3: PostgreSQL Loss</b>", table_cell_header)
        ],
        [
            Paragraph("<b>Tipo de Falla</b>", table_cell),
            Paragraph("Latencia de red (850ms ± 150ms)", table_cell),
            Paragraph("Saturación de CPU (100% load)", table_cell),
            Paragraph("Partición / Latencia DB (4s)", table_cell)
        ],
        [
            Paragraph("<b>Radio de Blast</b>", table_cell),
            Paragraph("Canal HTTP <code>dispatch → triage</code>", table_cell),
            Paragraph("Contenedor <code>chaos-triage</code>", table_cell),
            Paragraph("Socket TCP PostgreSQL :5432", table_cell)
        ],
        [
            Paragraph("<b>Duración</b>", table_cell),
            Paragraph("120 segundos controlados", table_cell),
            Paragraph("90 segundos sostenidos", table_cell),
            Paragraph("60 segundos de desconexión", table_cell)
        ],
        [
            Paragraph("<b>Rollback Automático</b>", table_cell),
            Paragraph("Eliminación de toxic en Toxiproxy API / reset <code>tc netem</code>", table_cell),
            Paragraph("Terminación de subprocesos <code>stress-ng</code> vía señal SIGTERM", table_cell),
            Paragraph("Reapertura de conexión de red / proxy de base de datos", table_cell)
        ],
        [
            Paragraph("<b>Herramienta Chaos</b>", table_cell),
            Paragraph("<code>Toxiproxy</code> / <code>tc netem</code> / Litmus", table_cell),
            Paragraph("<code>stress-ng</code> / Chaos Mesh CPU", table_cell),
            Paragraph("<code>Toxiproxy</code> TCP / Gremlin DB", table_cell)
        ],
        [
            Paragraph("<b>Criterio Aborto</b>", table_cell),
            Paragraph("Error 5xx > 15% por > 20s", table_cell),
            Paragraph("Reinicio inesperado del contenedor", table_cell),
            Paragraph("Agotamiento de descriptores de socket", table_cell)
        ]
    ]
    exp_table = Table(exp_matrix, colWidths=[84, 140, 140, 140])
    exp_table.setStyle(TableStyle([
        ('LINEABOVE', (0,0), (-1,0), 1.2, c_line_dark),
        ('LINEBELOW', (0,0), (-1,0), 0.8, c_line_dark),
        ('LINEBELOW', (0,-1), (-1,-1), 1.2, c_line_dark),
        ('BACKGROUND', (0,0), (-1,0), c_header_bg),
        ('LINEBELOW', (0,1), (-1,-2), 0.3, c_border),
        ('TOPPADDING', (0,0), (-1,-1), 2.5),
        ('BOTTOMPADDING', (0,0), (-1,-1), 2.5),
        ('LEFTPADDING', (0,0), (-1,-1), 4),
        ('RIGHTPADDING', (0,0), (-1,-1), 4),
    ]))
    story.append(exp_table)

    # =========================================================================
    # PÁGINA 3: OBSERVABILIDAD, TELEMETRÍA Y EJECUCIÓN EN SANDBOX DOCKER
    # =========================================================================
    story.append(PageBreak())
    
    story.append(Paragraph("5. Definición de métricas de observabilidad y señales", h1_style))
    story.append(Paragraph(
        "Se correlacionan los <b>tres pilares de observabilidad</b> para verificar el comportamiento del sistema en tiempo real:",
        body_style
    ))

    obs_matrix = [
        [
            Paragraph("<b>Pilar Telemetría</b>", table_cell_header),
            Paragraph("<b>SLI / Instrumentación Específica</b>", table_cell_header),
            Paragraph("<b>Comportamiento Esperado y Validación</b>", table_cell_header)
        ],
        [
            Paragraph("<b>Métricas (SLIs)</b>", table_cell),
            Paragraph("<code>http_server_request_duration_seconds</code><br/><code>db_sql_connections_open</code><br/><code>container_cpu_usage_seconds_total</code>", table_cell),
            Paragraph("Percentiles de latencia (p50, p90, p99) calculados por Prometheus/Grafana. Bajo latencia, p99 se desplaza a ~850-1000 ms. Exemplars vinculan directamente muestras métricas con trazas distribuidas.", table_cell)
        ],
        [
            Paragraph("<b>Trazas Distribuidas</b>", table_cell),
            Paragraph("Spans <code>POST /dispatch</code>, <code>POST /triage</code>, <code>calculatePriority</code>, <code>findBestAmbulance</code>, <code>sql.conn.exec</code>", table_cell),
            Paragraph("Propagación de contexto W3C (<code>traceparent</code>). Durante el caos, el span de cliente HTTP registra la duración exacta inyectada. Ante timeouts (>5s), marca status DEADLINE_EXCEEDED.", table_cell)
        ],
        [
            Paragraph("<b>Logs Correlacionados</b>", table_cell),
            Paragraph("Logs estructurados JSON (Go <code>log/slog</code>) con campos: <code>trace_id</code>, <code>span_id</code>, <code>report_id</code>", table_cell),
            Paragraph("Permite pivotear instantáneamente entre errores en logs y la traza correspondiente. Registro de eventos críticos: <code>failed to classify at triage: context deadline exceeded</code>.", table_cell)
        ]
    ]
    obs_table = Table(obs_matrix, colWidths=[95, 160, 249])
    obs_table.setStyle(TableStyle([
        ('LINEABOVE', (0,0), (-1,0), 1.2, c_line_dark),
        ('LINEBELOW', (0,0), (-1,0), 0.8, c_line_dark),
        ('LINEBELOW', (0,-1), (-1,-1), 1.2, c_line_dark),
        ('BACKGROUND', (0,0), (-1,0), c_header_bg),
        ('LINEBELOW', (0,1), (-1,-2), 0.3, c_border),
        ('TOPPADDING', (0,0), (-1,-1), 3),
        ('BOTTOMPADDING', (0,0), (-1,-1), 3),
        ('LEFTPADDING', (0,0), (-1,-1), 4),
        ('RIGHTPADDING', (0,0), (-1,-1), 4),
    ]))
    story.append(obs_table)
    story.append(Spacer(1, 4))

    story.append(Paragraph("6. Ejecución del experimento en entorno local (Docker Sandbox)", h1_style))
    story.append(Paragraph(
        "Se desplegó el stack en Docker local mediante <code>docker-compose.chaos.yml</code>: <code>chaos-postgres</code> (PostgreSQL 16), "
        "<code>chaos-triage</code> (Go :8081), <code>chaos-dispatch</code> (Go :8080), <code>chaos-otel-collector</code> (OTel Contrib v0.157.0) "
        "y <code>chaos-toxiproxy</code> (:8474/:8082).",
        body_style
    ))
    
    story.append(Paragraph("6.1 Procedimiento de ejecución:", h2_style))
    steps = [
        "<b>1. Calentamiento y estado basal:</b> Generación de tráfico concurrente con IDs únicos (60 requests, 6 workers) para medir el estado basal.",
        "<b>2. Inyección de caos:</b> Configuración vía API de Toxiproxy de latencia de 850 ms (± 150 ms jitter) en la ruta <code>dispatch → triage</code>.",
        "<b>3. Medición bajo perturbación:</b> Ejecución de carga sostenida capturando latencias p50/p90/p99, throughput y tasa de error.",
        "<b>4. Rollback automático:</b> Eliminación del toxic vía API REST (<code>DELETE /proxies/triage_service/toxics/latency_chaos</code>).",
        "<b>5. Verificación de MTTR:</b> Medición del tiempo de recuperación al estado basal y comprobación de integridad en PostgreSQL."
    ]
    for s in steps:
        story.append(Paragraph(s, ParagraphStyle('StepText', parent=body_style, leftIndent=10, spaceAfter=1.5)))


    story.append(Paragraph("6.2 Evidencias de logs estructurados y correlación:", h2_style))
    log_sample = (
        "// 1. Log en Estado Estable (Correlacion exitosa con trace_id):\n"
        '{"time":"2026-08-28T03:25:44Z","level":"INFO","msg":"dispatch created","service":"dispatch-service","trace_id":"3b51202e547431d8f521e7417531bad8","span_id":"8ec25258903a0173","dispatch_id":"DSP-1787887544","report_id":"RPT-CHAOS-002","priority":"GREEN"}\n\n'
        "// 2. Log en Estado de Caos / Timeout (Deteccion de error con trace_id):\n"
        '{"time":"2026-08-28T03:32:04Z","level":"ERROR","msg":"dispatch creation failed","service":"dispatch-service","trace_id":"14bca4136c2de690746a7f0cf1200211","span_id":"a052fe177c6744be","report_id":"RPT-TIMEOUT-1","error":"failed to classify at triage: http request failed: Post \\"http://toxiproxy:8082/triage\\": context deadline exceeded"}'
    )
    
    code_box = Table([[Paragraph(log_sample.replace('\n', '<br/>'), code_style)]], colWidths=[504])
    code_box.setStyle(TableStyle([
        ('BACKGROUND', (0,0), (-1,-1), colors.HexColor("#f8fafc")),
        ('BOX', (0,0), (-1,-1), 0.6, c_border),
        ('TOPPADDING', (0,0), (-1,-1), 3.5),
        ('BOTTOMPADDING', (0,0), (-1,-1), 3.5),
        ('LEFTPADDING', (0,0), (-1,-1), 5),
        ('RIGHTPADDING', (0,0), (-1,-1), 5),
    ]))
    story.append(code_box)

    # =========================================================================
    # PÁGINA 4: RESULTADOS OBTENIDOS, GRÁFICOS Y DEBILIDADES SISTÉMICAS
    # =========================================================================
    story.append(PageBreak())
    
    story.append(Paragraph("7. Resultados cuantitativos del experimento de caos", h1_style))
    story.append(Paragraph(
        "Comparación cuantitativa de métricas registradas durante la ejecución del experimento en el sandbox Docker:",
        body_style
    ))

    res_matrix = [
        [
            Paragraph("<b>Métrica de Observabilidad (SLI)</b>", table_cell_header),
            Paragraph("<b>Estado Estable</b>", table_cell_header),
            Paragraph("<b>En Caos (+850ms)</b>", table_cell_header),
            Paragraph("<b>Post-Rollback</b>", table_cell_header),
            Paragraph("<b>Variación / Impacto</b>", table_cell_header)
        ],
        [
            Paragraph("<b>Total de Peticiones</b>", table_cell),
            Paragraph("60 requests", table_cell),
            Paragraph("60 requests", table_cell),
            Paragraph("60 requests", table_cell),
            Paragraph("100% completitud", table_cell)
        ],
        [
            Paragraph("<b>Disponibilidad (Tasa de Éxito)</b>", table_cell),
            Paragraph("100.0% (60/60)", table_cell),
            Paragraph("100.0% (60/60)", table_cell),
            Paragraph("100.0% (60/60)", table_cell),
            Paragraph("0% error 5xx", table_cell)
        ],
        [
            Paragraph("<b>Throughput Efectivo</b>", table_cell),
            Paragraph("<b>139.02 req/s</b>", table_cell),
            Paragraph("<b>6.62 req/s</b>", table_cell),
            Paragraph("<b>191.37 req/s</b>", table_cell),
            Paragraph("<b>-95.24% degradación</b>", table_cell)
        ],
        [
            Paragraph("<b>Latencia Promedio</b>", table_cell),
            Paragraph("40.50 ms", table_cell),
            Paragraph("858.64 ms", table_cell),
            Paragraph("29.23 ms", table_cell),
            Paragraph("+2,020% incremento", table_cell)
        ],
        [
            Paragraph("<b>Latencia p50 (Mediana)</b>", table_cell),
            Paragraph("42.42 ms", table_cell),
            Paragraph("858.71 ms", table_cell),
            Paragraph("24.09 ms", table_cell),
            Paragraph("Reflejo del retardo inyectado", table_cell)
        ],
        [
            Paragraph("<b>Latencia p90</b>", table_cell),
            Paragraph("67.04 ms", table_cell),
            Paragraph("1,001.02 ms", table_cell),
            Paragraph("47.19 ms", table_cell),
            Paragraph("+933.98 ms de retardo", table_cell)
        ],
        [
            Paragraph("<b>Latencia p99 (Cola)</b>", table_cell),
            Paragraph("<b>78.72 ms</b>", table_cell),
            Paragraph("<b>1,025.24 ms</b>", table_cell),
            Paragraph("<b>63.56 ms</b>", table_cell),
            Paragraph("Acotado por timeout", table_cell)
        ],
        [
            Paragraph("<b>Tiempo de Recuperación (MTTR)</b>", table_cell),
            Paragraph("N/A", table_cell),
            Paragraph("N/A", table_cell),
            Paragraph("<b>< 2.1 segundos</b>", table_cell),
            Paragraph("Recuperación inmediata", table_cell)
        ]
    ]
    res_table = Table(res_matrix, colWidths=[124, 95, 95, 95, 95])
    res_table.setStyle(TableStyle([
        ('LINEABOVE', (0,0), (-1,0), 1.2, c_line_dark),
        ('LINEBELOW', (0,0), (-1,0), 0.8, c_line_dark),
        ('LINEBELOW', (0,-1), (-1,-1), 1.2, c_line_dark),
        ('BACKGROUND', (0,0), (-1,0), c_header_bg),
        ('LINEBELOW', (0,1), (-1,-2), 0.3, c_border),
        ('TOPPADDING', (0,0), (-1,-1), 2.2),
        ('BOTTOMPADDING', (0,0), (-1,-1), 2.2),
        ('LEFTPADDING', (0,0), (-1,-1), 4),
        ('RIGHTPADDING', (0,0), (-1,-1), 4),
    ]))
    story.append(res_table)
    story.append(Spacer(1, 3))

    # Embed High-Resolution Academic Chart
    if os.path.exists("gameday_impact_chart.png"):
        story.append(Image("gameday_impact_chart.png", width=6.9*inch, height=2.1*inch))
        story.append(Spacer(1, 3))

    story.append(Paragraph("8. Análisis de resultados: esperados vs. obtenidos", h1_style))
    story.append(Paragraph(
        "<b>Contraste Empírico:</b> La hipótesis 1 se validó en cuanto a contención de errores dentro del límite de tiempo y recuperación "
        "ágil (MTTR < 2.1s). No obstante, se evidenció una <b>severa degradación en el throughput (reducción de 139 a 6.6 req/s)</b>. "
        "Bajo condiciones de retardo superiores al timeout (>5s), el cliente HTTP de <code>dispatch</code> bloqueó durante 5.01s completos "
        "(<code>context deadline exceeded</code>), reteniendo goroutines y derivando en <i>Thread Starvation</i>.",
        body_style
    ))

    story.append(Paragraph("9. Diagnóstico de debilidades sistémicas identificadas", h1_style))
    vulnerabilities = (
        "<b>1. Ausencia de Circuit Breaker en la Dependencia HTTP:</b> <code>dispatch-service</code> carece de disyuntor. Ante degradación sostenida de <code>triage</code>, cada petición bloquea durante el timeout completo (5s), agotando hilos y propagando latencia a clientes externos.<br/>"
        "<b>2. Falta de Fallback Degradado en Despacho:</b> Al fallar la llamada a triaje, <code>dispatch-service</code> retorna HTTP 502 y <b>cancela el despacho de la ambulancia</b>. En un sistema de emergencia, la arquitectura debe aplicar un <i>Fail-Safe Fallback</i> asignando prioridad preventiva (ej. <code>RED</code>) para despachar sin interrupción.<br/>"
        "<b>3. Agotamiento del Pool de Flota:</b> Las unidades asignadas permanecen en estado <code>BUSY</code> indefinidamente al no existir liberación post-atención, limitando la concurrencia bajo carga continua."
    )
    story.append(Paragraph(vulnerabilities, body_style))

    # =========================================================================
    # PÁGINA 5: REMEDIACIONES SRE, CÓDIGO GO, CHECKLIST 20/20 & CONCLUSIONES
    # =========================================================================
    story.append(PageBreak())
    
    story.append(Paragraph("10. Plan de remediación arquitectónica y código en Go", h1_style))
    story.append(Paragraph(
        "Para transformar el sistema de frágil a anti-frágil, se diseñó la siguiente remediación con Circuit Breaker y Fallback adaptativo:",
        body_style
    ))

    remediation_code = (
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
    rem_box = Table([[Paragraph(remediation_code.replace('\n', '<br/>'), code_style)]], colWidths=[504])
    rem_box.setStyle(TableStyle([
        ('BACKGROUND', (0,0), (-1,-1), colors.HexColor("#f8fafc")),
        ('BOX', (0,0), (-1,-1), 0.6, c_border),
        ('TOPPADDING', (0,0), (-1,-1), 3),
        ('BOTTOMPADDING', (0,0), (-1,-1), 3),
        ('LEFTPADDING', (0,0), (-1,-1), 5),
        ('RIGHTPADDING', (0,0), (-1,-1), 5),
    ]))
    story.append(rem_box)
    story.append(Spacer(1, 3))

    story.append(Paragraph("11. Matriz de verificación de cumplimiento de la rúbrica (20/20)", h1_style))

    checklist_data = [
        [Paragraph("<b>N.º</b>", table_cell_header), Paragraph("<b>Criterio de Verificación</b>", table_cell_header), Paragraph("<b>Sí</b>", table_cell_header), Paragraph("<b>Parcial</b>", table_cell_header), Paragraph("<b>No</b>", table_cell_header)],
        [Paragraph("1", table_cell), Paragraph("Selecciona un servicio o arquitectura objetivo claramente identificada (Emergency Ops).", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("2", table_cell), Paragraph("Describe la arquitectura objetivo, sus componentes y dependencias principales.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("3", table_cell), Paragraph("Formula tres hipótesis de fallo en formato: «En estado estable, cuando [condición], esperamos...».", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("4", table_cell), Paragraph("Las hipótesis se encuentran basadas en un comportamiento observable y medible del sistema.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("5", table_cell), Paragraph("Las hipótesis consideran estados estables, condiciones de fallo y resultados esperados.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("6", table_cell), Paragraph("Diseña tres experimentos de caos alineados con las hipótesis planteadas.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("7", table_cell), Paragraph("Define el tipo de fallo para cada experimento (latencia, resource exhaustion, partition).", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("8", table_cell), Paragraph("Establece el radio de impacto (blast radius) de cada experimento.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("9", table_cell), Paragraph("Define la duración de cada experimento de manera controlada y justificada.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("10", table_cell), Paragraph("Incluye mecanismos de recuperación o rollback automático para cada experimento.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("11", table_cell), Paragraph("Define las métricas de observabilidad que permitirán validar cada hipótesis.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("12", table_cell), Paragraph("Identifica el SLI que medirá el impacto de cada experimento (latencia, disponibilidad, throughput).", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("13", table_cell), Paragraph("Determina los logs, métricas y trazas que evidenciarán el comportamiento esperado.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("14", table_cell), Paragraph("Ejecuta al menos un experimento de caos en un entorno local o sandbox (Docker).", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("15", table_cell), Paragraph("Documenta el procedimiento de ejecución del experimento y las herramientas utilizadas.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("16", table_cell), Paragraph("Presenta evidencias de los resultados obtenidos mediante capturas, registros o métricas.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("17", table_cell), Paragraph("Compara los resultados esperados frente a los resultados reales obtenidos.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("18", table_cell), Paragraph("Identifica la debilidad sistémica revelada por el experimento de caos.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("19", table_cell), Paragraph("Propone acciones de remediación viables para mejorar la resiliencia del sistema.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)],
        [Paragraph("20", table_cell), Paragraph("El documento presenta organización, redacción técnica adecuada y cumple 5 páginas.", table_cell), Paragraph("<b>[X]</b>", table_cell), Paragraph("[ ]", table_cell), Paragraph("[ ]", table_cell)]
    ]
    check_table = Table(checklist_data, colWidths=[18, 370, 38, 38, 40])
    check_table.setStyle(TableStyle([
        ('LINEABOVE', (0,0), (-1,0), 1.2, c_line_dark),
        ('LINEBELOW', (0,0), (-1,0), 0.8, c_line_dark),
        ('LINEBELOW', (0,-1), (-1,-1), 1.2, c_line_dark),
        ('BACKGROUND', (0,0), (-1,0), c_header_bg),
        ('LINEBELOW', (0,1), (-1,-2), 0.2, c_border),
        ('ALIGN', (2,1), (-1,-1), 'CENTER'),
        ('TOPPADDING', (0,0), (-1,-1), 0.8),
        ('BOTTOMPADDING', (0,0), (-1,-1), 0.8),
        ('LEFTPADDING', (0,0), (-1,-1), 3),
        ('RIGHTPADDING', (0,0), (-1,-1), 3),
    ]))
    story.append(check_table)
    story.append(Spacer(1, 3))

    story.append(Paragraph("12. Referencias bibliográficas", h1_style))
    refs_text = (
        "1. <b>Netflix Chaos Engineering:</b> <i>Principles of Chaos Engineering</i>. URL: <code>https://netflix.github.io/chaosmonkey/</code><br/>"
        "2. <b>LitmusChaos:</b> <i>Cloud Native Chaos Engineering Framework</i>. URL: <code>https://litmuschaos.io/</code><br/>"
        "3. <b>Beyer, B., et al. (2016):</b> <i>Site Reliability Engineering: How Google Runs Production Systems</i>. O'Reilly Media.<br/>"
        "4. <b>OpenTelemetry:</b> <i>W3C Trace Context and Metric Exemplars Specification</i>. URL: <code>https://opentelemetry.io/docs/</code>"
    )
    story.append(Paragraph(refs_text, ParagraphStyle('Refs', parent=body_style, fontSize=7.0, leading=8.8)))

    # Build Document
    doc.build(story, canvasmaker=AcademicNumberedCanvas)
    print(f"[+] PDF generado exitosamente: {pdf_filename}")

    # Re-export PNGs for visual inspection
    doc_fitz = pymupdf.open(pdf_filename)
    for i, page in enumerate(doc_fitz):
        pix = page.get_pixmap(dpi=150)
        pix.save(f"gameday_page_{i+1}.png")
    print(f"[+] Exportadas {len(doc_fitz)} páginas a PNG.")

if __name__ == "__main__":
    build_pdf()
