# pyright: reportUnknownMemberType=false, reportUnknownParameterType=false, reportMissingTypeArgument=false
"""AI sidecar — gRPC server.

Boots a grpc.aio.server, registers the AI service, and handles SIGTERM.

grpcio's stubs for aio servicer context are partial; pyright strict mode
is relaxed at the top of this file for that reason.
"""

from __future__ import annotations

import asyncio
import json
import logging
import os
import signal
from pathlib import Path
from typing import cast

import grpc

from ai_sidecar.handlers.extract import CitationError, extract_fields
from ai_sidecar.proto import ai_pb2, ai_pb2_grpc
from ai_sidecar.providers import InMemoryProvider, LiteLLMProvider, Provider


def _load_provider() -> Provider:
    """PACTLINE_AI_PROVIDER=stub returns the recorded fixture for offline e2e."""
    if os.environ.get("PACTLINE_AI_PROVIDER") == "stub":
        fixture = (
            Path(__file__).parent.parent.parent
            / "tests"
            / "fixtures"
            / "sample_extract_response.json"
        )
        raw = json.loads(fixture.read_text())
        if not isinstance(raw, dict):
            raise TypeError(f"stub fixture must be a JSON object, got {type(raw)}")
        return InMemoryProvider(cast(dict[str, object], raw))
    return LiteLLMProvider()


class AIService(ai_pb2_grpc.AIServicer):
    def __init__(self) -> None:
        self._provider: Provider = _load_provider()

    async def Health(  # noqa: N802 — gRPC method casing
        self,
        request: ai_pb2.HealthRequest,
        context: grpc.aio.ServicerContext,
    ) -> ai_pb2.HealthResponse:
        return ai_pb2.HealthResponse(status="ok")

    async def ExtractFields(  # noqa: N802 — gRPC method casing
        self,
        request: ai_pb2.ExtractRequest,
        context: grpc.aio.ServicerContext,
    ) -> ai_pb2.ExtractResponse:
        try:
            return await extract_fields(request, self._provider)
        except CitationError as e:
            await context.abort(grpc.StatusCode.FAILED_PRECONDITION, str(e))
            raise
        except ValueError as e:
            await context.abort(grpc.StatusCode.INVALID_ARGUMENT, str(e))
            raise


async def serve() -> None:
    port = os.environ.get("AI_SIDECAR_PORT", "50051")
    server = grpc.aio.server()
    ai_pb2_grpc.add_AIServicer_to_server(AIService(), server)
    server.add_insecure_port(f"[::]:{port}")
    await server.start()
    logging.info("ai-sidecar listening on :%s", port)

    stop = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, stop.set)
    await stop.wait()
    logging.info("ai-sidecar shutting down")
    await server.stop(grace=5)


def main() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    asyncio.run(serve())


if __name__ == "__main__":
    main()
