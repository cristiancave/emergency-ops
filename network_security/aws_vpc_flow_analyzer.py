import json
import random
from datetime import datetime, timezone

class AWSVPCFlowAnalyzer:
    """
    Analizador de AWS VPC Flow Logs (versión 5) para observabilidad L4/L7 y seguridad.
    Clasifica el tráfico en East-West (inter-servicio microservicios) y North-South (Ingreso/Egreso de la VPC).
    """
    def __init__(self, account_id="149511939303", vpc_id="vpc-0a8bf21d4c89"):
        self.account_id = account_id
        self.vpc_id = vpc_id
        self.flow_records = []

    def generate_flow_logs(self, count=50):
        interfaces = {
            'eni-01a2b3c4d5e6': ('10.0.1.15', 'dispatch-service'),
            'eni-02b3c4d5e6f7': ('10.0.1.20', 'triage-service'),
            'eni-03c4d5e6f7a8': ('10.0.2.45', 'data-service-rds'),
            'eni-04d5e6f7a8b9': ('10.0.0.5', 'alb-ingress')
        }
        
        now = int(datetime.now(timezone.utc).timestamp())
        
        for _ in range(count):
            eni, (src_ip, src_name) = random.choice(list(interfaces.items()))
            dst_eni, (dst_ip, dst_name) = random.choice(list(interfaces.items()))
            
            # Determine traffic direction
            if src_name == 'alb-ingress' or dst_name == 'alb-ingress':
                direction = 'North-South (N-S)'
            else:
                direction = 'East-West (E-W)'
                
            dst_port = random.choice([8080, 8081, 8083, 5432, 4317])
            packets = random.randint(10, 500)
            bytes_transferred = packets * random.randint(64, 1460)
            
            # 5% chance of REJECT security event
            action = 'REJECT' if random.random() < 0.05 else 'ACCEPT'
            
            record = {
                'version': 5,
                'account_id': self.account_id,
                'interface_id': eni,
                'srcaddr': src_ip,
                'dstaddr': dst_ip,
                'srcport': random.randint(32768, 60000),
                'dstport': dst_port,
                'protocol': 6, # TCP
                'packets': packets,
                'bytes': bytes_transferred,
                'start': now - 60,
                'end': now,
                'action': action,
                'log_status': 'OK',
                'flow_direction': direction,
                'src_service': src_name,
                'dst_service': dst_name
            }
            self.flow_records.append(record)

    def analyze_network_and_security(self):
        total_records = len(self.flow_records)
        ew_bytes = sum(r['bytes'] for r in self.flow_records if r['flow_direction'] == 'East-West (E-W)')
        ns_bytes = sum(r['bytes'] for r in self.flow_records if r['flow_direction'] == 'North-South (N-S)')
        
        accept_count = sum(1 for r in self.flow_records if r['action'] == 'ACCEPT')
        reject_count = sum(1 for r in self.flow_records if r['action'] == 'REJECT')
        
        reject_events = [r for r in self.flow_records if r['action'] == 'REJECT']
        
        return {
            'vpc_id': self.vpc_id,
            'account_id': self.account_id,
            'total_flow_logs_processed': total_records,
            'traffic_volume_bytes': {
                'east_west_inter_service': ew_bytes,
                'north_south_ingress_egress': ns_bytes,
                'total_bytes': ew_bytes + ns_bytes
            },
            'security_traffic_ratio': {
                'accept_count': accept_count,
                'reject_count': reject_count,
                'reject_rate_pct': round((reject_count / max(1, total_records)) * 100.0, 2)
            },
            'detected_security_anomalies': [
                {
                    'event': 'UNAUTHORIZED_PORT_SCAN_REJECT',
                    'src_ip': r['srcaddr'],
                    'dst_ip': r['dstaddr'],
                    'target_port': r['dstport'],
                    'action': r['action']
                } for r in reject_events[:3]
            ]
        }

if __name__ == '__main__':
    analyzer = AWSVPCFlowAnalyzer()
    analyzer.generate_flow_logs(100)
    res = analyzer.analyze_network_and_security()
    print("[+] Análisis de AWS VPC Flow Logs y Seguridad L4/L7:")
    print(json.dumps(res, indent=2))
