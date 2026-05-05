import pytest

from ai_sidecar.handlers.extract import extract_fields
from ai_sidecar.proto import ai_pb2
from ai_sidecar.providers import InMemoryProvider


async def test_prompt_version_mismatch_rejected() -> None:
    req = ai_pb2.ExtractRequest(
        document_id="DOC",
        parsed_text="x",
        field_names=["parties"],
        model_id="anthropic/claude-3-5-sonnet",
        prompt_version="v0",  # mismatch with sidecar's "v1"
    )
    with pytest.raises(ValueError, match="prompt_version mismatch"):
        await extract_fields(req, InMemoryProvider({"fields": []}))
