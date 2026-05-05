# pyright: reportUnknownMemberType=false, reportUnknownParameterType=false, reportMissingTypeArgument=false
"""Document sidecar — gRPC server.

Boots a grpc.aio.server, registers the Document service, and handles SIGTERM
gracefully.

grpcio's stubs for aio servicer context are partial; pyright strict mode is
relaxed at the top of this file for that reason.
"""

from __future__ import annotations

import asyncio
import logging
import os
import signal

import grpc

from document_sidecar.handlers.parse import parse_document
from document_sidecar.proto import document_pb2, document_pb2_grpc


class DocumentService(document_pb2_grpc.DocumentServicer):
    async def Health(  # noqa: N802 — gRPC method casing
        self,
        request: document_pb2.HealthRequest,
        context: grpc.aio.ServicerContext,
    ) -> document_pb2.HealthResponse:
        return document_pb2.HealthResponse(status="ok")

    async def Parse(  # noqa: N802 — gRPC method casing
        self,
        request: document_pb2.ParseRequest,
        context: grpc.aio.ServicerContext,
    ) -> document_pb2.ParseResponse:
        try:
            return parse_document(request.content, request.mime_type)
        except ValueError as e:
            await context.abort(grpc.StatusCode.INVALID_ARGUMENT, str(e))
            raise


async def serve() -> None:
    port = os.environ.get("DOCUMENT_SIDECAR_PORT", "50052")
    server = grpc.aio.server()
    document_pb2_grpc.add_DocumentServicer_to_server(DocumentService(), server)
    server.add_insecure_port(f"[::]:{port}")
    await server.start()
    logging.info("document-sidecar listening on :%s", port)

    stop = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, stop.set)
    await stop.wait()
    logging.info("document-sidecar shutting down")
    await server.stop(grace=5)


def main() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    asyncio.run(serve())


if __name__ == "__main__":
    main()
