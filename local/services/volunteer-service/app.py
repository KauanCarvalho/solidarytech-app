import os
import sys
import uuid
import time
import logging
import boto3
from flask import Flask, request, jsonify, g
from dotenv import load_dotenv
from opentelemetry import metrics

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
log = logging.getLogger(__name__)

load_dotenv()

app = Flask(__name__)

# --- Métricas HTTP ---
meter = metrics.get_meter(__name__)

http_requests_total = meter.create_counter(
    name="http_requests_total",
    description="Total de requisições HTTP recebidas",
)
http_request_duration_seconds = meter.create_histogram(
    name="http_request_duration_seconds",
    description="Duração das requisições HTTP em segundos",
)


@app.before_request
def _start_metrics_timer():
    g._metrics_start_time = time.time()


@app.after_request
def _record_http_metrics(response):
    duration = time.time() - g.get("_metrics_start_time", time.time())
    path = request.url_rule.rule if request.url_rule else request.path
    attributes = {
        "method": request.method,
        "path": path,
        "status": str(response.status_code),
    }
    http_requests_total.add(1, attributes)
    http_request_duration_seconds.record(duration, attributes)
    return response


AWS_REGION = os.getenv("AWS_REGION", "us-east-1")
DYNAMODB_TABLE = os.getenv("AWS_DYNAMODB_TABLE")
AWS_ENDPOINT_URL = os.getenv("AWS_ENDPOINT_URL")  # setado apenas em dev (LocalStack); ausente em prod

if not DYNAMODB_TABLE:
    log.critical("Erro: AWS_DYNAMODB_TABLE não definida.")
    sys.exit(1)

try:
    dynamodb = boto3.resource("dynamodb", region_name=AWS_REGION, endpoint_url=AWS_ENDPOINT_URL)
    table = dynamodb.Table(DYNAMODB_TABLE)
    log.info(f"Conectado à tabela DynamoDB: {DYNAMODB_TABLE}")
except Exception as e:
    log.critical(f"Falha ao conectar no DynamoDB: {e}")
    sys.exit(1)


@app.route('/health')
def health():
    return jsonify({"status": "ok", "service": "volunteer-service"})


@app.route('/volunteers', methods=['POST'])
def register_volunteer():
    data = request.get_json()
    if not data or not all(k in data for k in ('name', 'email', 'ngo_id')):
        return jsonify({"error": "Campos obrigatórios ausentes"}), 400

    volunteer_id = str(uuid.uuid4())
    item = {
        'volunteer_id': volunteer_id,
        'name': data['name'],
        'email': data['email'],
        'ngo_id': int(data['ngo_id']),
        'registered_at': str(int(time.time()))
    }

    try:
        table.put_item(Item=item)
        return jsonify(item), 201
    except Exception as e:
        log.error(f"Erro ao salvar voluntário no DynamoDB: {e}")
        return jsonify({"error": "Erro interno ao processar dados"}), 500


@app.route('/volunteers/<int:ngo_id>', methods=['GET'])
def get_volunteers_by_ngo(ngo_id):
    try:
        # Nota para avaliação dos alunos: Operação Scan simplificada para fins de desenvolvimento.
        # Em cenários complexos de produção, índices globais secundários (GSI) seriam exigidos.
        response = table.scan(
            FilterExpression=boto3.dynamodb.conditions.Attr('ngo_id').eq(ngo_id)
        )
        return jsonify(response.get('Items', [])), 200
    except Exception as e:
        log.error(f"Erro ao buscar dados no DynamoDB: {e}")
        return jsonify({"error": "Erro interno"}), 500


if __name__ == '__main__':
    port = int(os.getenv("PORT", 8083))
    app.run(host='0.0.0.0', port=port)
