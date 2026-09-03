"""Testes unitários do ngo-service.

app.py cria o pool de conexões com o Postgres (SimpleConnectionPool) em tempo
de importação e chama sys.exit(1) se a conexão falhar — por isso o mock de
psycopg2.pool.SimpleConnectionPool precisa estar ativo ANTES do `import app`,
senão o módulo tenta abrir uma conexão real e o teste nunca chega a rodar.
"""
import os
from unittest.mock import MagicMock, patch

os.environ["DATABASE_URL"] = "postgresql://test:test@localhost:5432/test_db"

_pool_patcher = patch("psycopg2.pool.SimpleConnectionPool")
_mock_pool_cls = _pool_patcher.start()
_mock_pool_cls.return_value = MagicMock()

import app as ngo_app  # noqa: E402 — import proposital após o mock do pool

_pool_patcher.stop()


def _client():
    return ngo_app.app.test_client()


def _cursor_ctx(fetchone_return=None, fetchall_return=None, execute_side_effect=None):
    """Mock de cursor psycopg2 usável em 'with conn.cursor(...) as cur:'."""
    cursor = MagicMock()
    cursor.fetchone.return_value = fetchone_return
    cursor.fetchall.return_value = fetchall_return or []
    if execute_side_effect:
        cursor.execute.side_effect = execute_side_effect

    ctx = MagicMock()
    ctx.__enter__.return_value = cursor
    ctx.__exit__.return_value = False
    return ctx, cursor


def test_health_returns_ok():
    resp = _client().get("/health")

    assert resp.status_code == 200
    assert resp.get_json() == {"status": "ok", "service": "ngo-service"}


def test_create_ngo_missing_fields_returns_400():
    resp = _client().post("/ngos", json={"name": "ONG Sem Dados"})

    assert resp.status_code == 400
    assert "error" in resp.get_json()


def test_create_ngo_success_returns_201():
    fake_row = {
        "id": 1,
        "name": "ONG Amiga",
        "email": "contato@ongamiga.org",
        "cause": "Educação",
        "city": "São Paulo",
    }
    cursor_ctx, cursor = _cursor_ctx(fetchone_return=fake_row)

    mock_conn = MagicMock()
    mock_conn.cursor.return_value = cursor_ctx
    ngo_app.pool.getconn.return_value = mock_conn

    resp = _client().post("/ngos", json={
        "name": "ONG Amiga",
        "email": "contato@ongamiga.org",
        "cause": "Educação",
        "city": "São Paulo",
    })

    assert resp.status_code == 201
    assert resp.get_json() == fake_row
    cursor.execute.assert_called_once()
    mock_conn.commit.assert_called_once()
    ngo_app.pool.putconn.assert_called_with(mock_conn)


def test_create_ngo_duplicate_email_returns_409():
    import psycopg2

    cursor_ctx, cursor = _cursor_ctx(execute_side_effect=psycopg2.IntegrityError("duplicate key"))

    mock_conn = MagicMock()
    mock_conn.cursor.return_value = cursor_ctx
    ngo_app.pool.getconn.return_value = mock_conn

    resp = _client().post("/ngos", json={
        "name": "ONG Duplicada",
        "email": "ja@existe.org",
        "cause": "Saúde",
        "city": "Rio de Janeiro",
    })

    assert resp.status_code == 409
    mock_conn.rollback.assert_called_once()


def test_get_ngos_returns_list():
    fake_rows = [{"id": 2, "name": "ONG B"}, {"id": 1, "name": "ONG A"}]
    cursor_ctx, cursor = _cursor_ctx(fetchall_return=fake_rows)

    mock_conn = MagicMock()
    mock_conn.cursor.return_value = cursor_ctx
    ngo_app.pool.getconn.return_value = mock_conn

    resp = _client().get("/ngos")

    assert resp.status_code == 200
    assert resp.get_json() == fake_rows
