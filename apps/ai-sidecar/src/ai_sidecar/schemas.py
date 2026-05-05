"""Pydantic models for AI sidecar I/O.

These mirror the proto messages but exist for two reasons: (1) LiteLLM's
structured-output path returns Python dicts, and Pydantic validates them;
(2) round-tripping through Pydantic catches missing-citation bugs at the
sidecar boundary, before they cross the gRPC line.
"""

from __future__ import annotations

from pydantic import BaseModel, Field


class CitationModel(BaseModel):
    document_id: str
    locator: str
    span_start: int = Field(ge=0)
    span_end: int = Field(ge=0)
    model_id: str
    prompt_version: str
    timestamp: str  # RFC3339


class ExtractedFieldModel(BaseModel):
    field_name: str
    value: str
    value_json: str = ""
    citation: CitationModel


class ExtractResultModel(BaseModel):
    fields: list[ExtractedFieldModel]
