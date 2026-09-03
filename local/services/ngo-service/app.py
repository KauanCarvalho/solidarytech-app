import os
import sys
import time
import psycopg2
from psycopg2.extras import RealDictCursor
from psycopg2.pool import SimpleConnectionPool
from flask import Flask, request, jsonify, g
from dotenv import load_dotenv
from opentelemetry import metrics
import logging

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


DATABASE_URL = os.getenv("DATABASE_URL")
if not DATABASE_URL:
    log.critical("Erro: DATABASE_URL não definida.")
    sys.exit(1)

try:
    pool = SimpleConnectionPool(1, 10, dsn=DATABASE_URL)
    log.info("Pool de conexões com o PostgreSQL (ngo-service) inicializado.")
except Exception as e:
    log.critical(f"Erro ao conectar ao PostgreSQL: {e}")
    sys.exit(1)


@app.route('/health')
def health():
    return jsonify({"status": "ok", "service": "ngo-service"})


@app.route('/ngos', methods=['POST'])
def create_ngo():
    data = request.get_json()
    if not data or not all(k in data for k in ('name', 'email', 'cause', 'city')):
        return jsonify({"error": "Campos obrigatórios ausentes"}), 400

    conn = pool.getconn()
    try:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                "INSERT INTO ngos (name, email, cause, city) VALUES (%s, %s, %s, %s) RETURNING *",
                (data['name'], data['email'], data['cause'], data['city'])
            )
            new_ngo = cur.fetchone()
            conn.commit()
            return jsonify(new_ngo), 201
    except psycopg2.IntegrityError:
        conn.rollback()
        return jsonify({"error": "E-mail já cadastrado"}), 409
    except Exception as e:
        conn.rollback()
        log.error(f"Erro ao criar ONG: {e}")
        return jsonify({"error": "Erro interno"}), 500
    finally:
        pool.putconn(conn)


@app.route('/ngos', methods=['GET'])
def get_ngos():
    conn = pool.getconn()
    try:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute("SELECT * FROM ngos ORDER BY id DESC")
            return jsonify(cur.fetchall()), 200
    except Exception as e:
        log.error(f"Erro ao buscar ONGs: {e}")
        return jsonify({"error": "Erro interno"}), 500
    finally:
        pool.putconn(conn)


if __name__ == '__main__':
    port = int(os.getenv("PORT", 8081))
    app.run(host='0.0.0.0', port=port)
