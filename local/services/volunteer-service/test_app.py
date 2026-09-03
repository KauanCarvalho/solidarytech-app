"""Testes unitários do volunteer-service.

app.py conecta ao DynamoDB (boto3.resource) em tempo de importação e chama
sys.exit(1) se falhar — por isso o mock de boto3.resource precisa estar ativo
ANTES do `import app`. boto3.dynamodb.conditions é importado explicitamente
aqui porque app.py acessa boto3.dynamodb.conditions.Attr em runtime, e o
submódulo só fica disponível como atributo se algo o importar — o que
normalmente aconteceria dentro do boto3.resource("dynamodb", ...) real, que
está mockado neste teste.
"""
import os
import uuid
from unittest.mock import MagicMock, patch

import boto3.dynamodb.conditions  # noqa: F401

os.environ["AWS_DYNAMODB_TABLE"] = "SolidaryTechVolunteersTest"
os.environ["AWS_REGION"] = "us-east-1"

_boto_patcher = patch("boto3.resource")
_mock_boto_resource = _boto_patcher.start()
_mock_table = MagicMock()
_mock_boto_resource.return_value.Table.return_value = _mock_table

import app as volunteer_app  # noqa: E402 — import proposital após o mock do boto3

_boto_patcher.stop()


def _client():
    return volunteer_app.app.test_client()


def test_health_returns_ok():
    resp = _client().get("/health")

    assert resp.status_code == 200
    assert resp.get_json() == {"status": "ok", "service": "volunteer-service"}


def test_register_volunteer_missing_fields_returns_400():
    resp = _client().post("/volunteers", json={"name": "Fulano"})

    assert resp.status_code == 400
    assert "error" in resp.get_json()


def test_register_volunteer_success_returns_201():
    volunteer_app.table.put_item.side_effect = None

    resp = _client().post("/volunteers", json={
        "name": "Fulano de Tal",
        "email": "fulano@example.com",
        "ngo_id": 1,
    })

    assert resp.status_code == 201
    body = resp.get_json()
    assert body["name"] == "Fulano de Tal"
    assert body["ngo_id"] == 1
    uuid.UUID(body["volunteer_id"])  # deve ser um UUID válido
    volunteer_app.table.put_item.assert_called_once()


def test_register_volunteer_dynamodb_failure_returns_500():
    volunteer_app.table.put_item.side_effect = Exception("DynamoDB indisponível")

    resp = _client().post("/volunteers", json={
        "name": "Fulano",
        "email": "fulano@example.com",
        "ngo_id": 2,
    })

    assert resp.status_code == 500

    volunteer_app.table.put_item.side_effect = None  # limpa para não afetar outros testes


def test_get_volunteers_by_ngo_returns_list():
    fake_items = [{"volunteer_id": "abc", "ngo_id": 3, "name": "Voluntária C"}]
    volunteer_app.table.scan.return_value = {"Items": fake_items}

    resp = _client().get("/volunteers/3")

    assert resp.status_code == 200
    assert resp.get_json() == fake_items
