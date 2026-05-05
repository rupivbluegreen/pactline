import json
from pathlib import Path

import pytest

from ai_sidecar.handlers.extract import CitationError, extract_fields
from ai_sidecar.proto import ai_pb2
from ai_sidecar.providers import InMemoryProvider

FIXTURE = Path(__file__).parent / "fixtures" / "sample_extract_response.json"


def _request() -> ai_pb2.ExtractRequest:
    return ai_pb2.ExtractRequest(
        document_id="DOC",
        parsed_text="MUTUAL NDA between Acme...",
        field_names=[
            "parties",
            "effective_date",
            "term",
            "value",
            "governing_law",
        ],
        model_id="anthropic/claude-3-5-sonnet",
        prompt_version="v1",
    )


async def test_extract_fields_returns_five_with_citations() -> None:
    payload = json.loads(FIXTURE.read_text())
    provider = InMemoryProvider(payload)

    resp = await extract_fields(_request(), provider)

    assert len(resp.fields) == 5
    names = sorted(f.field_name for f in resp.fields)
    assert names == [
        "effective_date",
        "governing_law",
        "parties",
        "term",
        "value",
    ]
    for f in resp.fields:
        assert f.citation.document_id == "DOC"
        assert f.citation.model_id == "anthropic/claude-3-5-sonnet"
        assert f.citation.prompt_version == "v1"
        assert f.citation.locator
        assert f.citation.span_end >= f.citation.span_start


async def test_extract_fields_missing_citation_raises() -> None:
    payload = json.loads(FIXTURE.read_text())
    payload["fields"][0]["citation"]["locator"] = ""
    provider = InMemoryProvider(payload)

    with pytest.raises(CitationError):
        await extract_fields(_request(), provider)


async def test_extract_fields_missing_field_raises() -> None:
    payload = json.loads(FIXTURE.read_text())
    payload["fields"] = [f for f in payload["fields"] if f["field_name"] != "term"]
    provider = InMemoryProvider(payload)

    with pytest.raises(CitationError):
        await extract_fields(_request(), provider)
