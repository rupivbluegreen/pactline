# pyright: reportUnknownMemberType=false, reportUnknownVariableType=false, reportUnknownArgumentType=false, reportMissingTypeStubs=false, reportIndexIssue=false
"""Provider abstraction. v1 is LiteLLM. A test-only InMemoryProvider lets
unit tests exercise the handler without a live LLM.

LiteLLM's runtime-typed call surface is partially typed; pyright strict
mode is relaxed at the top of this file for that reason.
"""

from __future__ import annotations

import json
from typing import Protocol


class Provider(Protocol):
    async def extract(
        self,
        *,
        model_id: str,
        system_prompt: str,
        user_text: str,
        field_names: list[str],
    ) -> dict[str, object]: ...


class LiteLLMProvider:
    """Wraps litellm.acompletion with structured-output + JSON mode."""

    async def extract(
        self,
        *,
        model_id: str,
        system_prompt: str,
        user_text: str,
        field_names: list[str],
    ) -> dict[str, object]:
        import litellm

        instruction = (
            "Return JSON with this shape:\n"
            '{"fields":[{"field_name":"...","value":"...","value_json":"...",'
            '"citation":{"document_id":"...","locator":"page:N or para:N",'
            '"span_start":0,"span_end":0,"model_id":"...","prompt_version":"...",'
            '"timestamp":"RFC3339"}}]}\n'
            f"Requested field_names: {field_names}.\n"
            "Document text follows:\n---\n" + user_text + "\n---\n"
        )

        result = await litellm.acompletion(
            model=model_id,
            messages=[
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": instruction},
            ],
            response_format={"type": "json_object"},
        )
        content = result["choices"][0]["message"]["content"]
        parsed = json.loads(content)
        if not isinstance(parsed, dict):
            raise TypeError(f"LLM did not return JSON object, got {type(parsed)}")
        return parsed


class InMemoryProvider:
    """Returns a fixed dict; used by tests."""

    def __init__(self, payload: dict[str, object]) -> None:
        self._payload = payload

    async def extract(
        self,
        *,
        model_id: str,
        system_prompt: str,
        user_text: str,
        field_names: list[str],
    ) -> dict[str, object]:
        return self._payload
