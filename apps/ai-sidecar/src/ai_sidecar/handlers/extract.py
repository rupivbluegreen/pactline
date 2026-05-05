"""ExtractFields handler. Calls a Provider, validates the response with
Pydantic, and rejects responses where any requested field is missing a
complete citation tuple."""

from __future__ import annotations

from datetime import UTC, datetime

from ai_sidecar.prompts import (
    EXTRACT_FIELDS_PROMPT_VERSION,
    EXTRACT_FIELDS_SYSTEM_PROMPT,
)
from ai_sidecar.proto import ai_pb2
from ai_sidecar.providers import Provider
from ai_sidecar.schemas import ExtractResultModel


class CitationError(ValueError):
    """Raised when a provider response is missing or malformed citations."""


async def extract_fields(
    request: ai_pb2.ExtractRequest, provider: Provider
) -> ai_pb2.ExtractResponse:
    if request.prompt_version != EXTRACT_FIELDS_PROMPT_VERSION:
        raise ValueError(
            f"prompt_version mismatch: caller={request.prompt_version}, "
            f"sidecar={EXTRACT_FIELDS_PROMPT_VERSION}"
        )

    raw = await provider.extract(
        model_id=request.model_id,
        system_prompt=EXTRACT_FIELDS_SYSTEM_PROMPT,
        user_text=request.parsed_text,
        field_names=list(request.field_names),
    )
    parsed = ExtractResultModel.model_validate(raw)

    requested = set(request.field_names)
    timestamp_now = datetime.now(UTC).isoformat()
    fields_out: list[ai_pb2.ExtractedField] = []
    seen: set[str] = set()
    for f in parsed.fields:
        if f.field_name not in requested:
            continue
        c = f.citation
        if (
            not c.document_id
            or not c.locator
            or not c.model_id
            or not c.prompt_version
            or c.span_end < c.span_start
        ):
            raise CitationError(
                f"incomplete citation on field {f.field_name!r}: {c.model_dump()}"
            )
        seen.add(f.field_name)
        fields_out.append(
            ai_pb2.ExtractedField(
                field_name=f.field_name,
                value=f.value,
                value_json=f.value_json,
                citation=ai_pb2.Citation(
                    document_id=c.document_id,
                    locator=c.locator,
                    span_start=c.span_start,
                    span_end=c.span_end,
                    model_id=c.model_id,
                    prompt_version=c.prompt_version,
                    timestamp=c.timestamp or timestamp_now,
                ),
            )
        )

    missing = requested - seen
    if missing:
        raise CitationError(f"missing fields: {sorted(missing)}")

    return ai_pb2.ExtractResponse(fields=fields_out)
