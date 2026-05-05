from pathlib import Path

from document_sidecar.handlers.parse import DOCX_MIME, parse_document

FIXTURE = Path(__file__).parent / "fixtures" / "sample.docx"


def test_parse_docx_text_and_segments() -> None:
    body = FIXTURE.read_bytes()
    resp = parse_document(body, DOCX_MIME)

    assert resp.page_count == 1
    assert "MUTUAL NDA" in resp.text
    assert "Effective Date: 2026-05-01" in resp.text

    locators = [s.locator for s in resp.segments]
    assert locators == ["para:1", "para:2", "para:3", "para:4"]

    cursor = 0
    for s in resp.segments:
        assert s.char_start == cursor
        assert s.char_end >= s.char_start
        cursor = s.char_end
    assert cursor == len(resp.text)
