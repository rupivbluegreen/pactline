"""AI sidecar entry point.

Phase 0: idle stub. Boots, logs the port it would listen on, waits for
SIGTERM. Phase 1 fills in a real grpc.aio.server with handlers
implementing the proto contract in proto/ai.proto.
"""

import asyncio
import logging
import os
import signal


async def serve() -> None:
    port = os.environ.get("AI_SIDECAR_PORT", "50051")
    logging.info(
        "ai-sidecar phase-0 stub idling on :%s (no gRPC server bound yet)", port
    )

    stop = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, stop.set)
    await stop.wait()
    logging.info("ai-sidecar shutting down")


def main() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    asyncio.run(serve())


if __name__ == "__main__":
    main()
