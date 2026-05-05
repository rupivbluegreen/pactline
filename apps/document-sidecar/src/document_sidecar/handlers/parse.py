# pyright: reportUnknownMemberType=false, reportUnknownVariableType=false, reportUnknownArgumentType=false, reportArgumentType=false
"""Parse handler — PDF via PyMuPDF, DOCX via python-docx.

Returns a flat text body plus a TextSegment offset map keyed by
"page:N" (PDF) or "para:N" (DOCX). The offset map is what the AI
sidecar uses to bind Citation char offsets back to a locator the
frontend can resolve.

PyMuPDF and python-docx ship partial type stubs; pyright strict mode is
relaxed at the top of this file so callers still get accurate types.
"""

from __future__ import annotations

from io import BytesIO

import pymupdf
from docx import Document as DocxDocument

from document_sidecar.proto import document_pb2

PDF_MIME = "application/pdf"
DOCX_MIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"


def parse_document(content: bytes, mime_type: str) -> document_pb2.ParseResponse:
    if mime_type == PDF_MIME:
        return _parse_pdf(content)
    if mime_type == DOCX_MIME:
        return _parse_docx(content)
    raise ValueError(f"unsupported mime type: {mime_type}")


def _parse_pdf(content: bytes) -> document_pb2.ParseResponse:
    doc = pymupdf.open(stream=content, filetype="pdf")
    segments: list[document_pb2.TextSegment] = []
    pieces: list[str] = []
    cursor = 0
    for i, page in enumerate(doc):
        text = page.get_text("text") or ""
        seg = document_pb2.TextSegment(
            locator=f"page:{i + 1}",
            char_start=cursor,
            char_end=cursor + len(text),
        )
        segments.append(seg)
        pieces.append(text)
        cursor += len(text)
    body = "".join(pieces)
    return document_pb2.ParseResponse(text=body, page_count=len(doc), segments=segments)


def _parse_docx(content: bytes) -> document_pb2.ParseResponse:
    doc = DocxDocument(BytesIO(content))
    segments: list[document_pb2.TextSegment] = []
    pieces: list[str] = []
    cursor = 0
    for i, para in enumerate(doc.paragraphs):
        text = (para.text or "") + "\n"
        segments.append(
            document_pb2.TextSegment(
                locator=f"para:{i + 1}",
                char_start=cursor,
                char_end=cursor + len(text),
            )
        )
        pieces.append(text)
        cursor += len(text)
    body = "".join(pieces)
    return document_pb2.ParseResponse(text=body, page_count=1, segments=segments)
